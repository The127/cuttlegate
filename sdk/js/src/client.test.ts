import { describe, it, expect } from 'vitest';
import { createClient, CuttlegateError } from './client.js';
import type { CuttlegateClient } from './client.js';

const validConfig = {
  baseUrl: 'https://cuttlegate.example.com',
  token: 'svc_abc',
  environment: 'production',
};

describe('createClient', () => {
  it('returns a CuttlegateClient with valid config', () => {
    const client = createClient(validConfig);
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

  it('throws synchronously when environment is missing', () => {
    expect(() =>
      createClient({ ...validConfig, environment: '' }),
    ).toThrow('environment is required');
  });

  it('uses default timeout and native fetch when not provided', () => {
    // Should not throw — defaults are applied internally
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
    // TypeScript compile-time check: a function accepting the interface
    function useClient(c: CuttlegateClient): boolean {
      return typeof c.evaluate === 'function' && typeof c.evaluateFlag === 'function';
    }
    const client = createClient(validConfig);
    expect(useClient(client)).toBe(true);
  });

  it('does not expose the token on the returned client object', () => {
    const client = createClient(validConfig);
    const serialised = JSON.stringify(client);
    expect(serialised).not.toContain('svc_abc');

    // Also check that no enumerable property contains the token
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
    // Provide a valid token but missing baseUrl to trigger an error
    try {
      createClient({ baseUrl: '', token: sensitiveToken, environment: 'prod' });
      expect.fail('should have thrown');
    } catch (err) {
      const message = (err as Error).message;
      expect(message).not.toContain(sensitiveToken);
      expect(String(err)).not.toContain(sensitiveToken);
    }
  });
});
