import { describe, it, expect, vi } from 'vitest';
import { createClient, CuttlegateError } from './client.js';
import type { CuttlegateClient } from './client.js';

const validConfig = {
  baseUrl: 'https://cuttlegate.example.com',
  token: 'svc_abc',
  project: 'my-project',
  environment: 'production',
};

function mockFetchResponse(body: unknown, status = 200): typeof fetch {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

const bulkResponse = {
  flags: [
    { key: 'dark-mode', enabled: true, value: null, value_key: 'true', reason: 'rule_match', type: 'bool' },
    { key: 'banner-text', enabled: true, value: 'holiday', value_key: 'holiday', reason: 'default', type: 'string' },
  ],
  evaluated_at: '2026-03-21T10:00:00Z',
};

describe('createClient', () => {
  it('returns a CuttlegateClient with valid config', () => {
    const client = createClient({ ...validConfig, fetch: mockFetchResponse({}) });
    expect(client).toBeDefined();
    expect(typeof client.evaluate).toBe('function');
    expect(typeof client.evaluateFlag).toBe('function');
  });

  it('throws synchronously when token is missing', () => {
    expect(() =>
      createClient({ ...validConfig, token: '' }),
    ).toThrow('token is required');
  });

  it('throws synchronously when baseUrl is missing', () => {
    expect(() =>
      createClient({ ...validConfig, baseUrl: '' }),
    ).toThrow('baseUrl is required');
  });

  it('throws synchronously when project is missing', () => {
    expect(() =>
      createClient({ ...validConfig, project: '' }),
    ).toThrow('project is required');
  });

  it('throws synchronously when environment is missing', () => {
    expect(() =>
      createClient({ ...validConfig, environment: '' }),
    ).toThrow('environment is required');
  });

  it('uses default timeout and native fetch when not provided', () => {
    const client = createClient(validConfig);
    expect(client).toBeDefined();
  });

  it('accepts optional timeout and custom fetch', () => {
    const customFetch = (() => {}) as unknown as typeof fetch;
    const client = createClient({
      ...validConfig,
      timeout: 10000,
      fetch: customFetch,
    });
    expect(client).toBeDefined();
  });

  it('is usable as a CuttlegateClient type', () => {
    function useClient(c: CuttlegateClient): boolean {
      return typeof c.evaluate === 'function' && typeof c.evaluateFlag === 'function';
    }
    const client = createClient({ ...validConfig, fetch: mockFetchResponse({}) });
    expect(useClient(client)).toBe(true);
  });

  it('does not expose the token on the returned client object', () => {
    const client = createClient({ ...validConfig, fetch: mockFetchResponse({}) });
    const serialised = JSON.stringify(client);
    expect(serialised).not.toContain('svc_abc');

    for (const value of Object.values(client)) {
      if (typeof value === 'string') {
        expect(value).not.toContain('svc_abc');
      }
    }
  });

  it('throws CuttlegateError with code field on validation failure', () => {
    try {
      createClient({ ...validConfig, baseUrl: '' });
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(CuttlegateError);
      expect((err as CuttlegateError).code).toBe('invalid_config');
    }
  });

  it('error messages do not leak the token value', () => {
    const sensitiveToken = 'super_secret_token_12345';
    try {
      createClient({ baseUrl: '', token: sensitiveToken, project: 'p', environment: 'prod' });
      expect.fail('should have thrown');
    } catch (err) {
      const message = (err as Error).message;
      expect(message).not.toContain(sensitiveToken);
      expect(String(err)).not.toContain(sensitiveToken);
    }
  });
});

describe('evaluate', () => {
  it('returns all flags for a user context', async () => {
    const fetchFn = mockFetchResponse(bulkResponse);
    const client = createClient({ ...validConfig, fetch: fetchFn });

    const results = await client.evaluate({ user_id: 'u1', attributes: { plan: 'pro' } });

    expect(fetchFn).toHaveBeenCalledOnce();
    const [url, opts] = (fetchFn as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toBe('https://cuttlegate.example.com/api/v1/projects/my-project/environments/production/evaluate');
    expect(opts.method).toBe('POST');
    expect(opts.headers['Authorization']).toBe('Bearer svc_abc');
    expect(JSON.parse(opts.body)).toEqual({ context: { user_id: 'u1', attributes: { plan: 'pro' } } });

    expect(results).toHaveLength(2);
    expect(results[0]).toEqual({
      key: 'dark-mode',
      enabled: true,
      value: null,
      valueKey: 'true',
      reason: 'rule_match',
      evaluatedAt: '2026-03-21T10:00:00Z',
    });
    expect(results[1]).toEqual({
      key: 'banner-text',
      enabled: true,
      value: 'holiday',
      valueKey: 'holiday',
      reason: 'default',
      evaluatedAt: '2026-03-21T10:00:00Z',
    });
  });

  it('throws CuttlegateError with code="network_error" when fetch fails', async () => {
    const fetchFn = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));
    const client = createClient({ ...validConfig, fetch: fetchFn });

    await expect(client.evaluate({ user_id: 'u1', attributes: {} })).rejects.toSatisfy(
      (err: unknown) => err instanceof CuttlegateError && err.code === 'network_error' && err.message === 'Failed to fetch',
    );
  });

  it('throws CuttlegateError with code="timeout" when request times out', async () => {
    const fetchFn = vi.fn().mockImplementation((_url: string, opts: RequestInit) => {
      return new Promise((_resolve, reject) => {
        opts.signal?.addEventListener('abort', () => {
          reject(new DOMException('The operation was aborted.', 'AbortError'));
        });
      });
    });
    const client = createClient({ ...validConfig, timeout: 50, fetch: fetchFn });

    await expect(client.evaluate({ user_id: 'u1', attributes: {} })).rejects.toSatisfy(
      (err: unknown) => err instanceof CuttlegateError && err.code === 'timeout',
    );
  });

  it('throws CuttlegateError with code="unauthorized" on 401', async () => {
    const fetchFn = mockFetchResponse({}, 401);
    const client = createClient({ ...validConfig, fetch: fetchFn });

    await expect(client.evaluate({ user_id: 'u1', attributes: {} })).rejects.toSatisfy(
      (err: unknown) => err instanceof CuttlegateError && err.code === 'unauthorized',
    );
  });

  it('throws CuttlegateError with code="forbidden" on 403', async () => {
    const fetchFn = mockFetchResponse({}, 403);
    const client = createClient({ ...validConfig, fetch: fetchFn });

    await expect(client.evaluate({ user_id: 'u1', attributes: {} })).rejects.toSatisfy(
      (err: unknown) => err instanceof CuttlegateError && err.code === 'forbidden',
    );
  });

  it('throws CuttlegateError with code="invalid_response" on malformed JSON shape', async () => {
    const fetchFn = mockFetchResponse({ unexpected: 'shape' });
    const client = createClient({ ...validConfig, fetch: fetchFn });

    await expect(client.evaluate({ user_id: 'u1', attributes: {} })).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof CuttlegateError &&
        err.code === 'invalid_response' &&
        err.message === 'Server returned an unexpected response shape',
    );
  });
});

describe('evaluateFlag', () => {
  it('returns a single flag result from bulk response', async () => {
    const client = createClient({ ...validConfig, fetch: mockFetchResponse(bulkResponse) });

    const result = await client.evaluateFlag('dark-mode', { user_id: 'u1', attributes: {} });
    // @happy: valueKey is "true" for bool flag; value remains null (v1 compat)
    expect(result).toEqual({ enabled: true, value: null, valueKey: 'true', reason: 'rule_match' });
  });

  it('returns not_found for unknown key without throwing', async () => {
    const client = createClient({ ...validConfig, fetch: mockFetchResponse(bulkResponse) });

    const result = await client.evaluateFlag('beta-feature', { user_id: 'u1', attributes: {} });
    expect(result).toEqual({ enabled: false, value: null, valueKey: '', reason: 'not_found' });
  });

  it('makes only one HTTP call (wraps evaluate)', async () => {
    const fetchFn = mockFetchResponse(bulkResponse);
    const client = createClient({ ...validConfig, fetch: fetchFn });

    await client.evaluateFlag('dark-mode', { user_id: 'u1', attributes: {} });
    expect(fetchFn).toHaveBeenCalledOnce();
  });
});
