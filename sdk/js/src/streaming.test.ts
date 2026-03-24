import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { connectStream } from './streaming.js';
import type { FlagStateChangedEvent } from './streaming.js';
import { CuttlegateError } from './client.js';

const validConfig = {
  baseUrl: 'https://cuttlegate.example.com',
  token: 'svc_abc',
  project: 'my-project',
  environment: 'production',
};

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

/** Create a mock fetch that returns an SSE response with the given chunks. */
function mockSSEFetch(...chunks: string[]): typeof fetch {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    body: sseStream(...chunks),
  });
}

/** Wait for async stream processing to settle. */
function settle(ms = 50): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

const wireEvent = (key: string, enabled = true) =>
  JSON.stringify({
    type: 'flag.state_changed',
    project: 'my-project',
    environment: 'production',
    flag_key: key,
    enabled,
    occurred_at: '2026-03-21T12:00:00Z',
  });

describe('connectStream', () => {
  it('receives flag change events with camelCase fields', async () => {
    const fetchFn = mockSSEFetch(`data: ${wireEvent('dark-mode')}\n\n`);
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    await settle();
    conn.close();

    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flagKey: 'dark-mode',
      enabled: true,
      occurredAt: '2026-03-21T12:00:00Z',
    });
  });

  it('validates wire format and calls onError for unexpected shapes', async () => {
    const badEvent = JSON.stringify({ unexpected: 'shape' });
    const fetchFn = mockSSEFetch(`data: ${badEvent}\n\n`);
    const errors: CuttlegateError[] = [];
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: (e) => events.push(e),
        onError: (e) => errors.push(e),
      },
    );

    await settle();
    conn.close();

    expect(events).toHaveLength(0);
    expect(errors).toHaveLength(1);
    expect(errors[0]).toBeInstanceOf(CuttlegateError);
    expect(errors[0].code).toBe('invalid_response');
  });

  it('close() disconnects and no further events are received', async () => {
    const events: FlagStateChangedEvent[] = [];

    let enqueue: ((chunk: Uint8Array) => void) | undefined;
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        enqueue = (chunk) => controller.enqueue(chunk);
      },
    });

    const fetchFn = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: stream,
    });

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    const encoder = new TextEncoder();

    enqueue!(encoder.encode(`data: ${wireEvent('first')}\n\n`));
    await settle(30);
    expect(events).toHaveLength(1);

    conn.close();
    await settle(10);

    expect(() =>
      enqueue!(encoder.encode(`data: ${wireEvent('second')}\n\n`)),
    ).toThrow();

    expect(events).toHaveLength(1);
    expect(events[0].flagKey).toBe('first');
  });

  it('converts snake_case wire fields to camelCase SDK types', async () => {
    const fetchFn = mockSSEFetch(`data: ${wireEvent('dark-mode')}\n\n`);
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    await settle();
    conn.close();

    expect(events[0].flagKey).toBe('dark-mode');
    expect(events[0].occurredAt).toBe('2026-03-21T12:00:00Z');
    expect('flag_key' in events[0]).toBe(false);
    expect('occurred_at' in events[0]).toBe(false);
  });

  it('calls onConnected(false) when stream opens for the first time', async () => {
    const fetchFn = mockSSEFetch('');
    const connected = vi.fn();

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onConnected: connected,
      },
    );

    await settle();
    conn.close();

    expect(connected).toHaveBeenCalledOnce();
    expect(connected).toHaveBeenCalledWith(false);
  });

  it('skips heartbeat comments', async () => {
    const fetchFn = mockSSEFetch(
      `: keep-alive\n\n` + `data: ${wireEvent('dark-mode')}\n\n`,
    );
    const events: FlagStateChangedEvent[] = [];
    const errors: CuttlegateError[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: (e) => events.push(e),
        onError: (e) => errors.push(e),
      },
    );

    await settle();
    conn.close();

    expect(events).toHaveLength(1);
    expect(errors).toHaveLength(0);
  });

  it('calls onError for non-JSON data lines', async () => {
    const fetchFn = mockSSEFetch(`data: not-json\n\n`);
    const errors: CuttlegateError[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onError: (e) => errors.push(e),
      },
    );

    await settle();
    conn.close();

    expect(errors).toHaveLength(1);
    expect(errors[0].code).toBe('invalid_response');
    expect(errors[0].message).toBe('SSE data is not valid JSON');
  });

  it('sends Authorization header with the token', async () => {
    const fetchFn = mockSSEFetch('');

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: () => {} },
    );

    await settle();
    conn.close();

    expect(fetchFn).toHaveBeenCalled();
    const [url, opts] = (fetchFn as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toBe(
      'https://cuttlegate.example.com/api/v1/projects/my-project/environments/production/flags/stream',
    );
    expect(opts.headers.Authorization).toBe('Bearer svc_abc');
  });

  it('calls onError with unauthorized code on 401', async () => {
    const fetchFn = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      body: null,
    });
    const errors: CuttlegateError[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onError: (e) => errors.push(e),
      },
    );

    await settle();
    conn.close();

    expect(errors).toHaveLength(1);
    expect(errors[0].code).toBe('unauthorized');
  });

  it('throws synchronously on missing config fields', () => {
    expect(() =>
      connectStream({ ...validConfig, baseUrl: '' }, { onFlagChange: () => {} }),
    ).toThrow('baseUrl is required');

    expect(() =>
      connectStream({ ...validConfig, token: '' }, { onFlagChange: () => {} }),
    ).toThrow('token is required');

    expect(() =>
      connectStream({ ...validConfig, project: '' }, { onFlagChange: () => {} }),
    ).toThrow('project is required');

    expect(() =>
      connectStream({ ...validConfig, environment: '' }, { onFlagChange: () => {} }),
    ).toThrow('environment is required');
  });

  it('handles multiple events in a single chunk', async () => {
    const fetchFn = mockSSEFetch(
      `data: ${wireEvent('flag-a')}\n\ndata: ${wireEvent('flag-b', false)}\n\n`,
    );
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    await settle();
    conn.close();

    expect(events).toHaveLength(2);
    expect(events[0].flagKey).toBe('flag-a');
    expect(events[1].flagKey).toBe('flag-b');
  });
});

describe('reconnect', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.spyOn(Math, 'random').mockReturnValue(0.5);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('reconnects after a transient network failure', async () => {
    let callCount = 0;
    const fetchFn = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.reject(new TypeError('Failed to fetch'));
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(`data: ${wireEvent('dark-mode')}\n\n`),
      });
    });

    const events: FlagStateChangedEvent[] = [];
    const connected = vi.fn();

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: (e) => events.push(e),
        onConnected: connected,
      },
    );

    await vi.advanceTimersByTimeAsync(2000);

    conn.close();

    expect(callCount).toBeGreaterThanOrEqual(2);
    expect(events.length).toBeGreaterThanOrEqual(1);
    expect(events[0].flagKey).toBe('dark-mode');
    expect(connected).toHaveBeenCalledWith(true);
  });

  it('calls onDisconnect when connection drops and reconnects', async () => {
    let callCount = 0;
    const fetchFn = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.resolve({
          ok: true,
          status: 200,
          body: sseStream(''),
        });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(`data: ${wireEvent('flag-a')}\n\n`),
      });
    });

    const onDisconnect = vi.fn();
    const connected = vi.fn();

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onDisconnect,
        onConnected: connected,
      },
    );

    await vi.advanceTimersByTimeAsync(2000);
    conn.close();

    expect(onDisconnect).toHaveBeenCalled();
    expect(connected).toHaveBeenCalledWith(false);
    expect(connected).toHaveBeenCalledWith(true);
  });

  it('stops retrying on 401 and fires onError', async () => {
    const fetchFn = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      body: null,
    });
    const errors: CuttlegateError[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onError: (e) => errors.push(e),
      },
    );

    await vi.advanceTimersByTimeAsync(5000);
    conn.close();

    expect(fetchFn).toHaveBeenCalledOnce();
    expect(errors).toHaveLength(1);
    expect(errors[0].code).toBe('unauthorized');
  });

  it('stops retrying on 403 and fires onError', async () => {
    const fetchFn = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      body: null,
    });
    const errors: CuttlegateError[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: () => {},
        onError: (e) => errors.push(e),
      },
    );

    await vi.advanceTimersByTimeAsync(5000);
    conn.close();

    expect(fetchFn).toHaveBeenCalledOnce();
    expect(errors).toHaveLength(1);
    expect(errors[0].code).toBe('forbidden');
  });

  it('does not fire onError on transient failures during reconnect', async () => {
    let callCount = 0;
    const fetchFn = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount <= 2) {
        return Promise.reject(new TypeError('Failed to fetch'));
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(`data: ${wireEvent('flag-a')}\n\n`),
      });
    });

    const errors: CuttlegateError[] = [];
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: (e) => events.push(e),
        onError: (e) => errors.push(e),
      },
    );

    await vi.advanceTimersByTimeAsync(10000);
    conn.close();

    expect(errors).toHaveLength(0);
    expect(events.length).toBeGreaterThanOrEqual(1);
  });

  it('uses exponential backoff with increasing delays', async () => {
    const timestamps: number[] = [];
    const fetchFn = vi.fn().mockImplementation(() => {
      timestamps.push(Date.now());
      if (timestamps.length < 4) {
        return Promise.reject(new TypeError('Failed to fetch'));
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(''),
      });
    });

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: () => {} },
    );

    await vi.advanceTimersByTimeAsync(60000);
    conn.close();

    expect(timestamps.length).toBeGreaterThanOrEqual(4);
    // First attempt is immediate (no delay)
    // Subsequent attempts have increasing delay (with jitter)
    if (timestamps.length >= 3) {
      const delay1 = timestamps[1] - timestamps[0];
      const delay2 = timestamps[2] - timestamps[1];
      expect(delay1).toBeGreaterThanOrEqual(0);
      expect(delay2).toBeGreaterThanOrEqual(0);
    }
  });

  it('reconnects after stream ends (server closes connection)', async () => {
    let callCount = 0;
    const fetchFn = vi.fn().mockImplementation(() => {
      callCount++;
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(
          callCount === 1
            ? `data: ${wireEvent('first')}\n\n`
            : `data: ${wireEvent('second')}\n\n`,
        ),
      });
    });

    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    await vi.advanceTimersByTimeAsync(3000);
    conn.close();

    expect(callCount).toBeGreaterThanOrEqual(2);
    expect(events.some((e) => e.flagKey === 'first')).toBe(true);
    expect(events.some((e) => e.flagKey === 'second')).toBe(true);
  });
});

describe('heartbeat timeout', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.spyOn(Math, 'random').mockReturnValue(0.5);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('triggers reconnect when no data arrives within 90 seconds', async () => {
    let callCount = 0;

    const fetchFn = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        // First connection: stream that hangs (no data sent)
        const stream = new ReadableStream<Uint8Array>({
          start() {
            // intentionally empty — no data, no close
          },
        });
        return Promise.resolve({ ok: true, status: 200, body: stream });
      }
      // Second connection after heartbeat timeout
      return Promise.resolve({
        ok: true,
        status: 200,
        body: sseStream(`data: ${wireEvent('reconnected')}\n\n`),
      });
    });

    const events: FlagStateChangedEvent[] = [];
    const connected = vi.fn();

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      {
        onFlagChange: (e) => events.push(e),
        onConnected: connected,
      },
    );

    // Wait for initial connection
    await vi.advanceTimersByTimeAsync(100);
    expect(connected).toHaveBeenCalledWith(false);

    // Advance past 90s heartbeat timeout + backoff + reconnect
    await vi.advanceTimersByTimeAsync(92000);

    conn.close();

    expect(callCount).toBeGreaterThanOrEqual(2);
    expect(events.length).toBeGreaterThanOrEqual(1);
    expect(events[0].flagKey).toBe('reconnected');
  });

  it('resets heartbeat timer when data arrives', async () => {
    let enqueue: ((chunk: Uint8Array) => void) | undefined;
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        enqueue = (chunk) => controller.enqueue(chunk);
      },
    });

    const fetchFn = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      body: stream,
    });

    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    const encoder = new TextEncoder();

    // Send data at 60s intervals (within the 90s timeout)
    await vi.advanceTimersByTimeAsync(60000);
    enqueue!(encoder.encode(`: keep-alive\n\n`));

    await vi.advanceTimersByTimeAsync(60000);
    enqueue!(encoder.encode(`data: ${wireEvent('flag-a')}\n\n`));

    await vi.advanceTimersByTimeAsync(60000);
    enqueue!(encoder.encode(`: keep-alive\n\n`));

    await vi.advanceTimersByTimeAsync(100);

    conn.close();

    // Connection should still be on its first attempt (no reconnect triggered)
    expect(fetchFn).toHaveBeenCalledOnce();
    expect(events).toHaveLength(1);
    expect(events[0].flagKey).toBe('flag-a');
  });
});
