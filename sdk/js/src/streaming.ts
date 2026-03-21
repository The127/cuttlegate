import { z } from 'zod';
import type { CuttlegateConfig } from './types.js';
import { CuttlegateError } from './client.js';

/** Options for connecting to the SSE flag state stream. */
export interface StreamOptions {
  /** Called when a flag state changes. */
  onFlagChange: (event: FlagStateChangedEvent) => void;
  /** Called on connection error or parse failure. Optional. */
  onError?: (error: CuttlegateError) => void;
  /** Called when the SSE connection is established. Optional. */
  onConnected?: () => void;
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

/**
 * Connect to the SSE stream for real-time flag state updates.
 *
 * Uses a fetch-based SSE reader with Authorization header — no EventSource,
 * no token in URL. Requires a runtime with ReadableStream support (Node 20+,
 * all modern browsers).
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

  startStream(url, config.token, fetchFn, controller.signal, options);

  return {
    close() {
      controller.abort();
    },
  };
}

async function startStream(
  url: string,
  token: string,
  fetchFn: typeof fetch,
  signal: AbortSignal,
  options: StreamOptions,
): Promise<void> {
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
    if (signal.aborted) return;
    options.onError?.(
      new CuttlegateError(
        'network_error',
        err instanceof Error ? err.message : 'Failed to connect to SSE stream',
      ),
    );
    return;
  }

  if (!res.ok) {
    options.onError?.(
      new CuttlegateError(
        res.status === 401 ? 'unauthorized' : 'network_error',
        `Server returned ${res.status}`,
      ),
    );
    return;
  }

  if (!res.body) {
    options.onError?.(
      new CuttlegateError('network_error', 'Response body is not readable'),
    );
    return;
  }

  options.onConnected?.();

  try {
    await readSSEStream(res.body, signal, options);
  } catch (err: unknown) {
    if (signal.aborted) return;
    options.onError?.(
      new CuttlegateError(
        'network_error',
        err instanceof Error ? err.message : 'SSE stream read failed',
      ),
    );
  }
}

async function readSSEStream(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
  options: StreamOptions,
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  // Cancel the reader when the abort signal fires — this unblocks any
  // pending reader.read() call that would otherwise wait indefinitely.
  const onAbort = () => reader.cancel();
  signal.addEventListener('abort', onAbort);

  try {
    while (!signal.aborted) {
      const { done, value } = await reader.read();
      if (done || signal.aborted) break;

      buffer += decoder.decode(value, { stream: true });

      // SSE events are terminated by a blank line (\n\n).
      // Process all complete events in the buffer.
      let boundary: number;
      while ((boundary = buffer.indexOf('\n\n')) !== -1) {
        const block = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);

        processSSEBlock(block, options);
      }
    }
  } finally {
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
