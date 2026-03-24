import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createCachedClient } from './cached-client.js';
import { CuttlegateError } from './client.js';

const validConfig = {
  baseUrl: 'https://cuttlegate.example.com',
  token: 'svc_secret_token',
  project: 'my-project',
  environment: 'production',
};

const defaultContext = { user_id: 'u1', attributes: {} };

/** Build a bulk-evaluate JSON response body. */
function bulkResponse(
  flags: Array<{ key: string; enabled: boolean; variant?: string; reason?: string }>,
  evaluatedAt = '2026-03-23T12:00:00Z',
): string {
  return JSON.stringify({
    flags: flags.map((f) => ({
      key: f.key,
      enabled: f.enabled,
      value: null,
      value_key: f.variant ?? (f.enabled ? 'true' : 'false'),
      reason: f.reason ?? 'default',
      type: 'bool',
    })),
    evaluated_at: evaluatedAt,
  });
}

/** Build a wire SSE event string (data: line + blank line). */
function wireSSEEvent(flagKey: string, enabled: boolean): string {
  return `data: ${JSON.stringify({
    type: 'flag.state_changed',
    project: 'my-project',
    environment: 'production',
    flag_key: flagKey,
    enabled,
    occurred_at: '2026-03-23T12:01:00Z',
  })}\n\n`;
}

/** Encode a string as a ReadableStream of Uint8Array chunks. */
function sseStream(...chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  let index = 0;
  return new ReadableStream({
    pull(controller) {
      if (index < chunks.length) {
        controller.enqueue(encoder.encode(chunks[index]));
        index++;
      } else {
        controller.close();
      }
    },
  });
}

/** An SSE stream that never delivers any data. */
function silentSSEStream(): ReadableStream<Uint8Array> {
  return new ReadableStream({ start() {} });
}

/** Wait for async tasks to settle. */
function settle(ms = 30): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Count how many fetch calls were made to the /evaluate endpoint. */
function evaluateCallCount(fetchFn: ReturnType<typeof vi.fn>): number {
  return (fetchFn.mock.calls as unknown[][]).filter(
    (args) => typeof args[0] === 'string' && (args[0] as string).endsWith('/evaluate'),
  ).length;
}

type MockResponse = {
  ok: boolean;
  status: number;
  json?: () => Promise<unknown>;
  body?: ReadableStream<Uint8Array> | null;
};

/** Build a mock fetch covering both /evaluate and SSE endpoints. */
function buildFetch(options: {
  hydrationStatus?: number;
  hydrationBody?: string;
  sseStatus?: number;
  sseChunks?: string[];
  hydrationDelay?: number;
}): typeof fetch {
  const {
    hydrationStatus = 200,
    hydrationBody = bulkResponse([{ key: 'my-flag', enabled: true }]),
    sseStatus = 200,
    sseChunks = [],
    hydrationDelay = 0,
  } = options;

  const fn = vi.fn().mockImplementation((url: unknown) => {
    const urlStr = url as string;
    if (urlStr.endsWith('/evaluate')) {
      const response: MockResponse = {
        ok: hydrationStatus >= 200 && hydrationStatus < 300,
        status: hydrationStatus,
        json: () => Promise.resolve(JSON.parse(hydrationBody)),
      };
      if (hydrationDelay > 0) {
        return new Promise<MockResponse>((resolve) =>
          setTimeout(() => resolve(response), hydrationDelay),
        );
      }
      return Promise.resolve(response);
    }
    // SSE stream endpoint
    if (sseStatus !== 200) {
      return Promise.resolve({ ok: false, status: sseStatus, body: null });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      body: sseChunks.length > 0 ? sseStream(...sseChunks) : silentSSEStream(),
    });
  });

  return fn as unknown as typeof fetch;
}

// ────────────────────────────────────────────────────────────────────────────
// Cache hit after hydration
// ────────────────────────────────────────────────────────────────────────────

describe('cache hit after hydration', () => {
  it('evaluate returns cached value without HTTP call after ready resolves', async () => {
    const fetchFn = vi.fn(
      buildFetch({
        hydrationBody: bulkResponse([
          { key: 'my-flag', enabled: true, variant: 'true', reason: 'rule_match' },
        ]),
      }),
    );

    const client = createCachedClient({ ...validConfig, fetch: fetchFn as unknown as typeof fetch });
    await client.ready;

    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.enabled).toBe(true);
    expect(result.variant).toBe('true');
    expect(result.reason).toBe('rule_match');

    // Only one call to the evaluate endpoint (initial hydration)
    expect(evaluateCallCount(fetchFn)).toBe(1);

    client.close();
  });

  it('evaluateAll() returns all cached flags without HTTP call', async () => {
    const fetchFn = vi.fn(
      buildFetch({
        hydrationBody: bulkResponse([
          { key: 'flag-a', enabled: true, reason: 'rule_match' },
          { key: 'flag-b', enabled: false, reason: 'disabled' },
          { key: 'flag-c', enabled: true, reason: 'rollout' },
        ]),
      }),
    );

    const client = createCachedClient({ ...validConfig, fetch: fetchFn as unknown as typeof fetch });
    await client.ready;

    const results = await client.evaluateAll(defaultContext);
    expect(results).toHaveLength(3);
    expect(evaluateCallCount(fetchFn)).toBe(1);

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// SSE update behaviour
// ────────────────────────────────────────────────────────────────────────────

describe('SSE update behaviour', () => {
  it('SSE update applies enabled; preserves variant from hydration; sets reason to "default"', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([
        { key: 'my-flag', enabled: false, variant: 'false', reason: 'disabled' },
      ]),
      sseChunks: [wireSSEEvent('my-flag', true)],
    });

    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;
    await settle();

    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.enabled).toBe(true);
    expect(result.variant).toBe('false'); // preserved from hydration
    expect(result.reason).toBe('default'); // set to "default" after SSE update

    client.close();
  });

  it('SSE update for unknown flag key is ignored', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([{ key: 'my-flag', enabled: true }]),
      sseChunks: [wireSSEEvent('unknown-flag', true)],
    });

    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;
    await settle();

    const result = await client.evaluate('unknown-flag', defaultContext);
    expect(result.enabled).toBe(false);
    expect(result.reason).toBe('not_found');

    client.close();
  });

  it('last SSE event wins on rapid updates for the same flag', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([{ key: 'my-flag', enabled: true }]),
      sseChunks: [
        wireSSEEvent('my-flag', false),
        wireSSEEvent('my-flag', true),
        wireSSEEvent('my-flag', false),
      ],
    });

    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;
    await settle();

    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.enabled).toBe(false); // last event wins

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// ready promise
// ────────────────────────────────────────────────────────────────────────────

describe('ready promise', () => {
  it('resolves after HTTP hydration completes', async () => {
    const fetchFn = buildFetch({});
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });

    await expect(client.ready).resolves.toBeUndefined();

    client.close();
  });

  it('rejects with code "unauthorized" on HTTP 401 during hydration', async () => {
    const fetchFn = buildFetch({ hydrationStatus: 401 });
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });

    await expect(client.ready).rejects.toSatisfy((err: unknown) => {
      return err instanceof CuttlegateError && err.code === 'unauthorized';
    });
  });

  it('rejects with code "forbidden" on HTTP 403 during hydration', async () => {
    const fetchFn = buildFetch({ hydrationStatus: 403 });
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });

    await expect(client.ready).rejects.toSatisfy((err: unknown) => {
      return err instanceof CuttlegateError && err.code === 'forbidden';
    });
  });

  it('rejects with code "timeout" when hydration fetch times out', async () => {
    const hangingFetch = vi.fn().mockImplementation((url: unknown, opts: unknown) => {
      const urlStr = url as string;
      if (urlStr.endsWith('/evaluate')) {
        return new Promise<never>((_, reject) => {
          const reqOpts = opts as RequestInit;
          const signal = reqOpts.signal as AbortSignal;
          const onAbort = () => {
            reject(new DOMException('The operation was aborted.', 'AbortError'));
          };
          if (signal?.aborted) {
            onAbort();
          } else {
            signal?.addEventListener('abort', onAbort, { once: true });
          }
        });
      }
      return Promise.resolve({ ok: true, status: 200, body: silentSSEStream() });
    });

    const client = createCachedClient({
      ...validConfig,
      fetch: hangingFetch as unknown as typeof fetch,
      timeout: 50,
    });

    await expect(client.ready).rejects.toSatisfy((err: unknown) => {
      return err instanceof CuttlegateError && err.code === 'timeout';
    });
  });
});

// ────────────────────────────────────────────────────────────────────────────
// SSE-first ordering: buffer-drain
// ────────────────────────────────────────────────────────────────────────────

describe('SSE-first ordering — buffer drain', () => {
  it('SSE event received during hydration is applied on top of hydration result', async () => {
    const encoder = new TextEncoder();

    let sseEnqueue: ((chunk: Uint8Array) => void) | undefined;
    const sseStreamInstance = new ReadableStream<Uint8Array>({
      start(controller) {
        sseEnqueue = (chunk) => controller.enqueue(chunk);
      },
    });

    const fetchFn = vi.fn().mockImplementation((url: unknown) => {
      const urlStr = url as string;
      if (urlStr.endsWith('/evaluate')) {
        // Delay hydration by 80ms so SSE event arrives first
        return new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                ok: true,
                status: 200,
                json: () =>
                  Promise.resolve(
                    JSON.parse(
                      bulkResponse([
                        { key: 'my-flag', enabled: false, variant: 'false', reason: 'disabled' },
                      ]),
                    ),
                  ),
              }),
            80,
          ),
        );
      }
      // SSE stream
      return Promise.resolve({ ok: true, status: 200, body: sseStreamInstance });
    });

    const client = createCachedClient({
      ...validConfig,
      fetch: fetchFn as unknown as typeof fetch,
    });

    // Let SSE connection establish
    await settle(20);
    // Send SSE event before hydration completes (SSE arrives first)
    sseEnqueue!(encoder.encode(wireSSEEvent('my-flag', true)));

    // Wait for hydration (arrives at 80ms)
    await client.ready;

    // The SSE event (enabled: true) was buffered, then applied on top of
    // the hydration result (enabled: false). Buffer drain wins.
    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.enabled).toBe(true);

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// close()
// ────────────────────────────────────────────────────────────────────────────

describe('close()', () => {
  it('terminates the SSE connection; cached values are retained after close', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([{ key: 'my-flag', enabled: true }]),
    });

    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;

    client.close();
    await settle();

    // Cache should still be accessible after close
    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.enabled).toBe(true);
  });

  it('is safe to call multiple times', async () => {
    const fetchFn = buildFetch({});
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;

    expect(() => {
      client.close();
      client.close();
      client.close();
    }).not.toThrow();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Cache miss
// ────────────────────────────────────────────────────────────────────────────

describe('cache miss', () => {
  it('returns not_found for unknown flag key with no HTTP fallback', async () => {
    const fetchFn = vi.fn(
      buildFetch({
        hydrationBody: bulkResponse([{ key: 'flag-a', enabled: true }]),
      }),
    );

    const client = createCachedClient({ ...validConfig, fetch: fetchFn as unknown as typeof fetch });
    await client.ready;

    const result = await client.evaluate('flag-not-in-cache', defaultContext);
    expect(result.enabled).toBe(false);
    expect(result.value).toBeNull();
    expect(result.variant).toBe('');
    expect(result.reason).toBe('not_found');

    // No extra HTTP calls after hydration
    expect(evaluateCallCount(fetchFn)).toBe(1);

    client.close();
  });

  it('evaluate before ready resolves returns not_found (empty cache)', async () => {
    const fetchFn = buildFetch({});
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });

    // Call before ready resolves — cache is empty
    const result = await client.evaluate('my-flag', defaultContext);
    expect(result.reason).toBe('not_found');

    await client.ready;
    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Edge cases
// ────────────────────────────────────────────────────────────────────────────

describe('edge cases', () => {
  it('handles empty flag list from hydration', async () => {
    const fetchFn = buildFetch({ hydrationBody: bulkResponse([]) });
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;

    const results = await client.evaluateAll(defaultContext);
    expect(results).toHaveLength(0);

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Terminal SSE auth error calls onError
// ────────────────────────────────────────────────────────────────────────────

describe('SSE terminal auth error', () => {
  it('calls onError with CuttlegateError { code: "unauthorized" } on SSE 401 after hydration', async () => {
    const sseErrors: CuttlegateError[] = [];

    const fetchFn = vi.fn().mockImplementation((url: unknown) => {
      const urlStr = url as string;
      if (urlStr.endsWith('/evaluate')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () =>
            Promise.resolve(JSON.parse(bulkResponse([{ key: 'my-flag', enabled: true }]))),
        });
      }
      // SSE: terminal 401
      return Promise.resolve({ ok: false, status: 401, body: null });
    });

    const client = createCachedClient(
      { ...validConfig, fetch: fetchFn as unknown as typeof fetch },
      { onError: (err) => sseErrors.push(err) },
    );

    await client.ready;
    await settle();

    expect(sseErrors).toHaveLength(1);
    expect(sseErrors[0]).toBeInstanceOf(CuttlegateError);
    expect(sseErrors[0].code).toBe('unauthorized');

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Token security
// ────────────────────────────────────────────────────────────────────────────

describe('token security', () => {
  it('token is not present on any enumerable property of the returned CachedClient', async () => {
    const fetchFn = buildFetch({});
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });

    const serialised = JSON.stringify(client);
    expect(serialised).not.toContain('svc_secret_token');

    const keys = Object.keys(client);
    for (const key of keys) {
      const value = (client as unknown as Record<string, unknown>)[key];
      expect(String(value)).not.toContain('svc_secret_token');
    }

    await client.ready;
    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Config validation
// ────────────────────────────────────────────────────────────────────────────

describe('config validation', () => {
  it('throws synchronously on missing required config fields', () => {
    expect(() => createCachedClient({ ...validConfig, baseUrl: '' })).toThrow('baseUrl is required');
    expect(() => createCachedClient({ ...validConfig, token: '' })).toThrow('token is required');
    expect(() => createCachedClient({ ...validConfig, project: '' })).toThrow('project is required');
    expect(() => createCachedClient({ ...validConfig, environment: '' })).toThrow(
      'environment is required',
    );
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Reconnect re-hydration
// ────────────────────────────────────────────────────────────────────────────

describe('reconnect re-hydration', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('re-hydrates when SSE reconnects with onConnected(reconnect=true)', async () => {
    let sseCallCount = 0;
    let hydrateCallCount = 0;

    const fetchFn = vi.fn().mockImplementation((url: unknown) => {
      const urlStr = url as string;
      if (urlStr.endsWith('/evaluate')) {
        hydrateCallCount++;
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () =>
            Promise.resolve(JSON.parse(bulkResponse([{ key: 'my-flag', enabled: true }]))),
        });
      }
      // SSE stream
      sseCallCount++;
      if (sseCallCount === 1) {
        // First SSE: closes immediately to trigger reconnect
        return Promise.resolve({
          ok: true,
          status: 200,
          body: new ReadableStream<Uint8Array>({
            start(controller) {
              setTimeout(() => controller.close(), 10);
            },
          }),
        });
      }
      // Subsequent SSE: silent stream
      return Promise.resolve({ ok: true, status: 200, body: silentSSEStream() });
    });

    const client = createCachedClient({
      ...validConfig,
      fetch: fetchFn as unknown as typeof fetch,
    });

    // Wait for initial hydration
    await client.ready;

    // Advance time to trigger SSE reconnect + re-hydration
    await vi.advanceTimersByTimeAsync(3000);

    // At least two hydration calls: initial + one re-hydration after reconnect
    expect(hydrateCallCount).toBeGreaterThanOrEqual(2);

    client.close();
  });
});

// ────────────────────────────────────────────────────────────────────────────
// Convenience methods (bool / string)
// ────────────────────────────────────────────────────────────────────────────

describe('convenience methods', () => {
  it('bool returns true for a flag with variant "true"', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([{ key: 'dark-mode', enabled: true, variant: 'true' }]),
    });
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;

    const result = await client.bool('dark-mode', defaultContext);
    expect(result).toBe(true);

    client.close();
  });

  it('string returns the variant string', async () => {
    const fetchFn = buildFetch({
      hydrationBody: bulkResponse([{ key: 'theme', enabled: true, variant: 'ocean' }]),
    });
    const client = createCachedClient({ ...validConfig, fetch: fetchFn });
    await client.ready;

    const result = await client.string('theme', defaultContext);
    expect(result).toBe('ocean');

    client.close();
  });
});
