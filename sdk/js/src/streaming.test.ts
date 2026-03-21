import { describe, it, expect, vi } from 'vitest';
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

describe('connectStream', () => {
  it('receives flag change events with camelCase fields', async () => {
    const wireEvent = JSON.stringify({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flag_key: 'dark-mode',
      enabled: true,
      occurred_at: '2026-03-21T12:00:00Z',
    });
    const fetchFn = mockSSEFetch(`data: ${wireEvent}\n\n`);
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

  it('calls onError on connection failure', async () => {
    const fetchFn = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));
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
    expect(errors[0]).toBeInstanceOf(CuttlegateError);
    expect(errors[0].code).toBe('network_error');
    expect(errors[0].message).toBe('Failed to fetch');
  });

  it('close() disconnects and no further events are received', async () => {
    const events: FlagStateChangedEvent[] = [];
    const wireEvent = (key: string) =>
      JSON.stringify({
        type: 'flag.state_changed',
        project: 'my-project',
        environment: 'production',
        flag_key: key,
        enabled: true,
        occurred_at: '2026-03-21T12:00:00Z',
      });

    // Manually controlled stream: we enqueue the first event, wait for it to
    // be processed, call close(), then try to enqueue a second event.
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

    // Deliver first event.
    enqueue!(encoder.encode(`data: ${wireEvent('first')}\n\n`));
    await settle(30);
    expect(events).toHaveLength(1);

    // Close the connection.
    conn.close();
    await settle(10);

    // After close(), the stream controller is cancelled — enqueue throws.
    // This confirms the connection is fully torn down.
    expect(() =>
      enqueue!(encoder.encode(`data: ${wireEvent('second')}\n\n`)),
    ).toThrow();

    expect(events).toHaveLength(1);
    expect(events[0].flagKey).toBe('first');
  });

  it('converts snake_case wire fields to camelCase SDK types', async () => {
    const wireEvent = JSON.stringify({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flag_key: 'dark-mode',
      enabled: true,
      occurred_at: '2026-03-21T12:00:00Z',
    });
    const fetchFn = mockSSEFetch(`data: ${wireEvent}\n\n`);
    const events: FlagStateChangedEvent[] = [];

    const conn = connectStream(
      { ...validConfig, fetch: fetchFn },
      { onFlagChange: (e) => events.push(e) },
    );

    await settle();
    conn.close();

    expect(events[0].flagKey).toBe('dark-mode');
    expect(events[0].occurredAt).toBe('2026-03-21T12:00:00Z');
    // Verify wire format fields are NOT present on the SDK type.
    expect('flag_key' in events[0]).toBe(false);
    expect('occurred_at' in events[0]).toBe(false);
  });

  it('calls onConnected when stream opens', async () => {
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
  });

  it('skips heartbeat comments', async () => {
    const wireEvent = JSON.stringify({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flag_key: 'dark-mode',
      enabled: true,
      occurred_at: '2026-03-21T12:00:00Z',
    });
    const fetchFn = mockSSEFetch(
      `: keep-alive\n\n` + `data: ${wireEvent}\n\n`,
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

    expect(fetchFn).toHaveBeenCalledOnce();
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
    const event1 = JSON.stringify({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flag_key: 'flag-a',
      enabled: true,
      occurred_at: '2026-03-21T12:00:00Z',
    });
    const event2 = JSON.stringify({
      type: 'flag.state_changed',
      project: 'my-project',
      environment: 'production',
      flag_key: 'flag-b',
      enabled: false,
      occurred_at: '2026-03-21T12:01:00Z',
    });
    const fetchFn = mockSSEFetch(
      `data: ${event1}\n\ndata: ${event2}\n\n`,
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
