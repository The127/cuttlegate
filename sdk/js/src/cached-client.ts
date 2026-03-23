import type { CuttlegateConfig, EvalContext } from './types.js';
import type { CuttlegateClient, EvaluationResult, FlagResult } from './client.js';
import { CuttlegateError } from './client.js';
import { connectStream } from './streaming.js';
import type { FlagStateChangedEvent, StreamConnection } from './streaming.js';

/**
 * Options for createCachedClient.
 */
export interface CachedClientOptions {
  /**
   * Called on terminal SSE auth errors (401/403) that occur after successful
   * hydration. The cache retains its last-known values. No reconnect is
   * attempted after a terminal error.
   */
  onError?: (err: CuttlegateError) => void;
}

/**
 * A Cuttlegate client that maintains an in-memory flag cache backed by a
 * single SSE connection.
 *
 * Implements CuttlegateClient — drop-in replaceable with createClient.
 *
 * evaluateFlag and evaluate serve from cache without HTTP round-trips.
 * Unknown flag keys return { enabled: false, value: null, valueKey: "",
 * reason: "not_found" } — there is no HTTP fallback.
 *
 * Await ready before calling evaluateFlag/evaluate to ensure the cache is
 * seeded. Calls made before ready resolves return not_found from an empty
 * cache.
 */
export interface CachedClient extends CuttlegateClient {
  /** Terminates the SSE connection. Cached values are retained after close. */
  close(): void;
  /**
   * Resolves when the initial HTTP hydration completes successfully.
   * Rejects with CuttlegateError on HTTP 401 (code: "unauthorized"),
   * 403 (code: "forbidden"), or timeout (code: "timeout") during hydration.
   * The SSE connection is also closed on hydration failure.
   */
  ready: Promise<void>;
}

// Compile-time assertion: CachedClient must extend CuttlegateClient.
// Enforced structurally by the interface extension above.

/** Wire shape returned by the bulk evaluate endpoint. */
interface BulkEvaluateResponse {
  flags: Array<{
    key: string;
    enabled: boolean;
    value: string | null;
    value_key: string;
    reason: string;
    type: string;
  }>;
  evaluated_at: string;
}

/**
 * Create an in-memory flag cache backed by a single SSE connection.
 *
 * On construction:
 * 1. Opens the SSE stream (SSE-first ordering — see ADR 0025).
 * 2. SSE events received during hydration are buffered.
 * 3. Calls evaluate() for HTTP hydration.
 * 4. On hydration success: drains the buffer on top of the hydration result,
 *    then resolves ready.
 * 5. On hydration failure (401/403/timeout): closes the SSE stream, rejects
 *    ready.
 *
 * The token is closure-scoped and does not appear on any enumerable property
 * of the returned CachedClient object.
 */
export function createCachedClient(
  config: CuttlegateConfig,
  options?: CachedClientOptions,
): CachedClient {
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
  const project = config.project;
  const environment = config.environment;
  const timeout = config.timeout ?? 5000;
  const fetchFn = config.fetch ?? globalThis.fetch;
  // token is captured in the closure only — not on any property
  const token = config.token;

  const evalUrl = `${baseUrl}/api/v1/projects/${encodeURIComponent(project)}/environments/${encodeURIComponent(environment)}/evaluate`;

  // In-memory cache: flagKey → EvaluationResult (as seeded by hydration).
  const cache = new Map<string, EvaluationResult>();

  // Buffer for SSE events that arrive before hydration completes.
  const sseBuffer: FlagStateChangedEvent[] = [];
  let hydrationComplete = false;
  let closed = false;

  let sseConnection: StreamConnection | undefined;

  // Resolve/reject handles for the ready promise.
  let resolveReady!: () => void;
  let rejectReady!: (err: CuttlegateError) => void;

  const ready = new Promise<void>((resolve, reject) => {
    resolveReady = resolve;
    rejectReady = reject;
  });

  /** Populate the cache from a parsed bulk-evaluate response. */
  function populateCache(data: BulkEvaluateResponse): void {
    cache.clear();
    for (const flag of data.flags) {
      cache.set(flag.key, {
        key: flag.key,
        enabled: flag.enabled,
        value: flag.value,
        valueKey: flag.value_key || flag.value || '',
        reason: flag.reason,
        evaluatedAt: data.evaluated_at,
      });
    }
  }

  // Apply a single SSE event to the cache.
  // Preserves valueKey from hydration; sets reason to "default" (consistent
  // with CuttlegateProvider in react.ts). Ignores unknown keys.
  function applyCacheEvent(event: FlagStateChangedEvent): void {
    const existing = cache.get(event.flagKey);
    if (!existing) return; // unknown key — ignore
    cache.set(event.flagKey, { ...existing, enabled: event.enabled, reason: 'default' });
  }

  // Called when an SSE event arrives.
  function onFlagChange(event: FlagStateChangedEvent): void {
    if (closed) return;
    if (!hydrationComplete) {
      sseBuffer.push(event);
      return;
    }
    applyCacheEvent(event);
  }

  // Called on terminal SSE auth errors (post-hydration).
  function onSSEError(err: CuttlegateError): void {
    if (!closed) options?.onError?.(err);
  }

  // Called when SSE reconnects — re-hydrate to close the missed-events gap.
  function onConnected(reconnect: boolean): void {
    if (!reconnect || closed) return;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeout);

    fetchFn(evalUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ context: { user_id: '', attributes: {} } }),
      signal: controller.signal,
    })
      .then(async (res) => {
        clearTimeout(timer);
        if (closed) return;
        if (!res.ok) {
          if (res.status === 401) {
            options?.onError?.(new CuttlegateError('unauthorized', 'Re-hydration returned 401'));
          } else if (res.status === 403) {
            options?.onError?.(new CuttlegateError('forbidden', 'Re-hydration returned 403'));
          }
          return;
        }
        const json = (await res.json()) as BulkEvaluateResponse;
        if (!closed) populateCache(json);
      })
      .catch(() => {
        clearTimeout(timer);
      });
  }

  /** Fail hydration: close SSE, mark closed, reject ready. */
  function failHydration(err: CuttlegateError): void {
    sseConnection?.close();
    closed = true;
    rejectReady(err);
  }

  // Step 1: Open the SSE stream (SSE-first ordering per ADR 0025).
  sseConnection = connectStream(config, { onFlagChange, onError: onSSEError, onConnected });

  // Step 2: HTTP hydration.
  const hydrationController = new AbortController();
  const hydrationTimer = setTimeout(() => hydrationController.abort(), timeout);

  fetchFn(evalUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ context: { user_id: '', attributes: {} } }),
    signal: hydrationController.signal,
  })
    .then(async (res) => {
      clearTimeout(hydrationTimer);

      if (closed) {
        rejectReady(new CuttlegateError('closed', 'Client was closed before hydration completed'));
        return;
      }

      if (res.status === 401) {
        failHydration(new CuttlegateError('unauthorized', 'Hydration returned 401 Unauthorized'));
        return;
      }
      if (res.status === 403) {
        failHydration(new CuttlegateError('forbidden', 'Hydration returned 403 Forbidden'));
        return;
      }
      if (!res.ok) {
        failHydration(new CuttlegateError('network_error', `Hydration returned ${res.status}`));
        return;
      }

      let json: unknown;
      try {
        json = await res.json();
      } catch {
        failHydration(new CuttlegateError('invalid_response', 'Hydration response is not valid JSON'));
        return;
      }

      if (closed) {
        rejectReady(new CuttlegateError('closed', 'Client was closed before hydration completed'));
        return;
      }

      // Step 3: Populate cache, then drain the SSE buffer on top.
      // Synchronous drain — no new SSE events can interleave here.
      populateCache(json as BulkEvaluateResponse);
      hydrationComplete = true;
      for (const event of sseBuffer) {
        applyCacheEvent(event);
      }
      sseBuffer.length = 0;

      resolveReady();
    })
    .catch((err: unknown) => {
      clearTimeout(hydrationTimer);
      if (closed) {
        rejectReady(new CuttlegateError('closed', 'Client was closed before hydration completed'));
        return;
      }
      if (err instanceof DOMException && err.name === 'AbortError') {
        failHydration(new CuttlegateError('timeout', `Hydration timed out after ${timeout}ms`));
        return;
      }
      failHydration(
        new CuttlegateError(
          'network_error',
          err instanceof Error ? err.message : 'Hydration network request failed',
        ),
      );
    });

  // Public API — token is not on any enumerable property.
  function evaluateFlag(key: string, _context: EvalContext): Promise<FlagResult> {
    const entry = cache.get(key);
    if (!entry) {
      return Promise.resolve({ enabled: false, value: null, valueKey: '', reason: 'not_found' });
    }
    return Promise.resolve({
      enabled: entry.enabled,
      value: entry.value,
      valueKey: entry.valueKey,
      reason: entry.reason,
    });
  }

  function evaluate(_context: EvalContext): Promise<EvaluationResult[]> {
    return Promise.resolve(Array.from(cache.values()));
  }

  function close(): void {
    if (closed) return;
    closed = true;
    hydrationController.abort();
    sseConnection?.close();
  }

  return { evaluate, evaluateFlag, close, ready };
}
