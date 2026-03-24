import { describe, it, expect, beforeEach } from 'vitest';
import { createMockClient, type MockCuttlegateClient } from './testing.js';
import type { CuttlegateClient } from './client.js';

describe('MockCuttlegateClient', () => {
  let mock: MockCuttlegateClient;
  const ctx = { user_id: 'user-1', attributes: {} as Record<string, string> };

  beforeEach(() => {
    mock = createMockClient();
  });

  // BDD: evaluate returns mock_default for unconfigured flags
  it('returns mock_default reason for unconfigured flags via evaluate', async () => {
    const result = await mock.evaluate('any-flag', ctx);

    expect(result).toEqual({
      key: 'any-flag',
      enabled: false,
      value: null,
      variant: '',
      reason: 'mock_default',
      evaluatedAt: '1970-01-01T00:00:00Z',
    });
  });

  // BDD: deprecated evaluateFlag still works
  it('evaluateFlag returns disabled with mock_default reason for unconfigured flags', async () => {
    const result = await mock.evaluateFlag('any-flag', ctx);

    expect(result).toEqual({
      enabled: false,
      value: null,
      variant: '',
      reason: 'mock_default',
    });
  });

  // BDD: Enable a flag
  it('returns enabled after enable() is called', async () => {
    mock.enable('dark-mode');

    const result = await mock.evaluate('dark-mode', ctx);

    expect(result.enabled).toBe(true);
    expect(result.variant).toBe('true');
    expect(result.reason).toBe('mock');
  });

  // BDD: Set a variant
  it('returns enabled with variant after setVariant() is called', async () => {
    mock.setVariant('button-color', 'blue');

    const result = await mock.evaluate('button-color', ctx);

    expect(result.enabled).toBe(true);
    expect(result.value).toBe('blue');
    expect(result.variant).toBe('blue');
    expect(result.reason).toBe('mock');
  });

  // BDD: Assert a flag was evaluated
  it('assertEvaluated passes when the flag was evaluated', async () => {
    mock.enable('my-flag');
    await mock.evaluate('my-flag', ctx);

    expect(() => mock.assertEvaluated('my-flag')).not.toThrow();
  });

  // BDD: Assert a flag was NOT evaluated
  it('assertNotEvaluated passes when no evaluation was made', () => {
    expect(() => mock.assertNotEvaluated('any-flag')).not.toThrow();
  });

  it('assertEvaluated throws when the flag was not evaluated', () => {
    expect(() => mock.assertEvaluated('any-flag')).toThrow(
      'Expected flag "any-flag" to have been evaluated',
    );
  });

  // BDD: Mock satisfies CuttlegateClient interface
  it('satisfies CuttlegateClient interface at compile time', () => {
    const client: CuttlegateClient = mock;
    expect(client.evaluate).toBeDefined();
    expect(client.evaluateAll).toBeDefined();
    expect(client.bool).toBeDefined();
    expect(client.string).toBeDefined();
    expect(client.evaluateFlag).toBeDefined();
  });

  // BDD: No network access required
  it('works without network access (no HTTP requests made)', async () => {
    mock.enable('offline-flag');

    const result = await mock.evaluate('offline-flag', ctx);

    expect(result.enabled).toBe(true);
  });

  // BDD: evaluateAll() returns results for all configured flags
  it('evaluateAll() returns results for all configured flags only', async () => {
    mock.enable('flag-a');
    mock.setVariant('flag-b', 'v1');

    const results = await mock.evaluateAll(ctx);

    expect(results).toHaveLength(2);
    expect(results).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'flag-a',
          enabled: true,
          value: null,
          reason: 'mock',
        }),
        expect.objectContaining({
          key: 'flag-b',
          enabled: true,
          value: 'v1',
          reason: 'mock',
        }),
      ]),
    );
  });

  it('evaluateAll() returns empty array when no flags are configured', async () => {
    const results = await mock.evaluateAll(ctx);
    expect(results).toEqual([]);
  });

  it('evaluateAll() does not include unconfigured flags', async () => {
    mock.enable('flag-a');
    const results = await mock.evaluateAll(ctx);
    const keys = results.map((r) => r.key);
    expect(keys).toEqual(['flag-a']);
  });

  it('evaluateAll() tracks evaluation for assertEvaluated', async () => {
    mock.enable('tracked');
    await mock.evaluateAll(ctx);
    expect(() => mock.assertEvaluated('tracked')).not.toThrow();
  });

  it('evaluate() tracks evaluation for unconfigured flags too', async () => {
    await mock.evaluate('unknown', ctx);
    expect(() => mock.assertEvaluated('unknown')).not.toThrow();
  });

  it('evaluateAll() returns deterministic evaluatedAt timestamp', async () => {
    mock.enable('ts-flag');
    const results = await mock.evaluateAll(ctx);
    expect(results[0].evaluatedAt).toBe('1970-01-01T00:00:00Z');
  });

  it('reset() clears all state', async () => {
    mock.enable('flag');
    await mock.evaluate('flag', ctx);

    mock.reset();

    const result = await mock.evaluate('flag', ctx);
    expect(result.reason).toBe('mock_default');
    expect(() => mock.assertNotEvaluated('other')).not.toThrow();
  });

  // BDD: bool convenience method
  it('bool() returns true for enabled bool flag', async () => {
    mock.enable('dark-mode');
    const result = await mock.bool('dark-mode', ctx);
    expect(result).toBe(true);
  });

  it('bool() returns false for non-bool variant', async () => {
    mock.setVariant('theme', 'ocean');
    const result = await mock.bool('theme', ctx);
    expect(result).toBe(false);
  });

  // BDD: string convenience method
  it('string() returns the variant string', async () => {
    mock.setVariant('theme', 'ocean');
    const result = await mock.string('theme', ctx);
    expect(result).toBe('ocean');
  });
});
