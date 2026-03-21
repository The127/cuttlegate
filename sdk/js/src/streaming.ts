import { z } from 'zod';
import type { CuttlegateConfig } from './types.js';
import { CuttlegateError } from './client.js';

/** Options for connecting to the SSE flag state stream. */
export interface StreamOptions {
  /** Called when a flag state changes. */
  onFlagChange: (event: FlagStateChangedEvent) => void;
  /** Called when retries are exhausted or a terminal error occurs (401/403). */
  onError?: (error: CuttlegateError) => void;
  /** Called when the SSE connection is established (first or reconnect). */
  onConnected?: (reconnect: boolean) => void;
  /** Called when the connection drops. */
  onDisconnect?: () => void;
}

/** A flag state change event received from the SSE stream. */
export interface FlagStateChangedEvent {
  type: 'flag.state_changed';
  project: string;
  environment: string;
  flagKey: string;
  enabled: boolean;
  occurredAt: string;
}

/** Handle to an active SSE stream connection. */
export interface StreamConnection {
  /** Close the SSE connection. */
  close(): void;
}

/** Wire format schema — validates incoming SSE JSON before conversion. */
const FlagStateChangedWireSchema = z.object({
  type: z.literal('flag.state_changed'),
  project: z.string(),
  environment: z.string(),
  flag_key: z.string(),
  enabled: z.boolean(),
  occurred_at: z.string(),
});

const INITIAL_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30_000;
const HEARTBEAT_TIMEOUT_MS = 90_000;

/**
 * Connect to the SSE stream for real-time flag state updates.
 *
 * Uses a fetch-based SSE reader with Authorization header — no EventSource,
 * no token in URL. Automatically reconnects with exponential backoff on
 * transient failures. Auth errors (401/403) are terminal.
 */
export function connectStream(
  config: CuttlegateConfig,
  options: StreamOptions,
): StreamConnection {
  if (!config.baseUrl) {
    throw new CuttlegateError('invalid_config', 'baseUrl is required');
  }
  if (!config.token) {
    throw new CuttlegateError('invalid_config', 'token is required');
  }
  if (!config.project) {
    throw new CuttlegateError('invalid_config', 'project is required');
  }
  if (!config.environment) {
    throw new CuttlegateError('invalid_config', 'environment is required');
  }

  const baseUrl = config.baseUrl.replace(/\/+$/, '');
  const project = encodeURIComponent(config.project);
  const environment = encodeURIComponent(config.environment);
  const url = `${baseUrl}/api/v1/projects/${project}/environments/${environment}/flags/stream`;
  const fetchFn = config.fetch ?? globalThis.fetch;

  const controller = new AbortController();

  runConnectionLoop(url, config.token, fetchFn, controller.signal, options);

  return {
    close() {
      controller.abort();
    },
  };
}

/** Compute backoff delay with full jitter. */
function backoffDelay(attempt: number): number {
  const base = Math.min(INITIAL_BACKOFF_MS * 2 ** attempt, MAX_BACKOFF_MS);
  return Math.random() * base;
}

/**
 * Result of a single stream attempt.
 * - 'reconnect': transient failure, should retry
 * - 'terminal': auth error, stop retrying
 * - 'closed': consumer called close(), stop
 */
type StreamResult = 'reconnect' | 'terminal' | 'closed';

async function runConnectionLoop(
  url: string,
  token: string,
  fetchFn: typeof fetch,
  signal: AbortSignal,
  options: StreamOptions,
): Promise<void> {
  let attempt = 0;
  let hasConnectedBefore = false;

  while (!signal.aborted) {
    if (attempt > 0) {
      const delay = backoffDelay(attempt - 1);
      await new Promise<void>((resolve) => {
        const timer = setTimeout(resolve, delay);
        const onAbort = () => {
          clearTimeout(timer);
          resolve();
        };
        signal.addEventListener('abort', onAbort, { once: true });
      });
      if (signal.aborted) break;
    }

    const result = await attemptStream(
      url,
      token,
      fetchFn,
      signal,
      options,
      hasConnectedBefore,
    );

    if (result === 'closed' || result === 'terminal') break;

    // Transient failure — reconnect
    if (hasConnectedBefore) {
      options.onDisconnect?.();
    }
    attempt++;
    hasConnectedBefore = true;
  }
}

async function attemptStream(
  url: string,
  token: string,
  fetchFn: typeof fetch,
  signal: AbortSignal,
  options: StreamOptions,
  isReconnect: boolean,
): Promise<StreamResult> {
  let res: Response;
  try {
    res = await fetchFn(url, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'text/event-stream',
      },
      signal,
    });
  } catch (err: unknown) {
    if (signal.aborted) return 'closed';
    return 'reconnect';
  }

  if (res.status === 401 || res.status === 403) {
    options.onError?.(
      new CuttlegateError(
        res.status === 401 ? 'unauthorized' : 'forbidden',
        `Server returned ${res.status}`,
      ),
    );
    return 'terminal';
  }

  if (!res.ok) {
    return 'reconnect';
  }

  if (!res.body) {
    return 'reconnect';
  }

  options.onConnected?.(isReconnect);

  try {
    await readSSEStream(res.body, signal, options);
  } catch (err: unknown) {
    if (signal.aborted) return 'closed';
  }

  // Stream ended (server closed connection or heartbeat timeout) — reconnect
  if (signal.aborted) return 'closed';
  return 'reconnect';
}

async function readSSEStream(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
  options: StreamOptions,
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  // Heartbeat timeout: if no data arrives within 90s, throw to trigger reconnect.
  let heartbeatTimer: ReturnType<typeof setTimeout> | undefined;
  const resetHeartbeat = () => {
    if (heartbeatTimer !== undefined) clearTimeout(heartbeatTimer);
    heartbeatTimer = setTimeout(() => {
      reader.cancel();
    }, HEARTBEAT_TIMEOUT_MS);
  };

  // Cancel the reader when the abort signal fires.
  const onAbort = () => {
    if (heartbeatTimer !== undefined) clearTimeout(heartbeatTimer);
    reader.cancel();
  };
  signal.addEventListener('abort', onAbort);
  resetHeartbeat();

  try {
    while (!signal.aborted) {
      const { done, value } = await reader.read();
      if (done || signal.aborted) break;

      resetHeartbeat();
      buffer += decoder.decode(value, { stream: true });

      // SSE events are terminated by a blank line (\n\n).
      let boundary: number;
      while ((boundary = buffer.indexOf('\n\n')) !== -1) {
        const block = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);

        processSSEBlock(block, options);
      }
    }
  } finally {
    if (heartbeatTimer !== undefined) clearTimeout(heartbeatTimer);
    signal.removeEventListener('abort', onAbort);
    reader.releaseLock();
  }
}

function processSSEBlock(block: string, options: StreamOptions): void {
  const lines = block.split('\n');
  for (const line of lines) {
    // Skip heartbeat comments and empty lines.
    if (line.startsWith(':') || line.trim() === '') continue;

    if (line.startsWith('data:')) {
      const jsonStr = line.slice(5).trim();
      if (!jsonStr) continue;

      let parsed: unknown;
      try {
        parsed = JSON.parse(jsonStr);
      } catch {
        options.onError?.(
          new CuttlegateError('invalid_response', 'SSE data is not valid JSON'),
        );
        continue;
      }

      const result = FlagStateChangedWireSchema.safeParse(parsed);
      if (!result.success) {
        options.onError?.(
          new CuttlegateError(
            'invalid_response',
            'SSE event has unexpected shape',
          ),
        );
        continue;
      }

      const wire = result.data;
      options.onFlagChange({
        type: wire.type,
        project: wire.project,
        environment: wire.environment,
        flagKey: wire.flag_key,
        enabled: wire.enabled,
        occurredAt: wire.occurred_at,
      });
    }
  }
}
