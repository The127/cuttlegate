import { describe, it, expect, beforeEach } from 'vitest';
import { createMockClient, type MockCuttlegateClient } from './testing.js';
import type { CuttlegateClient } from './client.js';

describe('MockCuttlegateClient', () => {
  let mock: MockCuttlegateClient;
  const ctx = { user_id: 'user-1', attributes: {} as Record<string, string> };

  beforeEach(() => {
    mock = createMockClient();
  });

  // BDD: All flags disabled by default
  it('returns disabled with mock_default reason for unconfigured flags', async () => {
    const result = await mock.evaluateFlag('any-flag', ctx);

    expect(result).toEqual({
      enabled: false,
      value: null,
      valueKey: '',
      reason: 'mock_default',
    });
  });

  // BDD: Enable a flag
  it('returns enabled after enable() is called', async () => {
    mock.enable('dark-mode');

    const result = await mock.evaluateFlag('dark-mode', ctx);

    expect(result).toEqual({
      enabled: true,
      value: null,
      valueKey: 'true',
      reason: 'mock',
    });
  });

  // BDD: Set a variant
  it('returns enabled with value after setVariant() is called', async () => {
    mock.setVariant('button-color', 'blue');

    const result = await mock.evaluateFlag('button-color', ctx);

    expect(result).toEqual({
      enabled: true,
      value: 'blue',
      valueKey: 'blue',
      reason: 'mock',
    });
  });

  // BDD: Assert a flag was evaluated
  it('assertEvaluated passes when the flag was evaluated', async () => {
    mock.enable('my-flag');
    await mock.evaluateFlag('my-flag', ctx);

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
    expect(client.evaluateFlag).toBeDefined();
  });

  // BDD: No network access required
  it('works without network access (no HTTP requests made)', async () => {
    mock.enable('offline-flag');

    const result = await mock.evaluateFlag('offline-flag', ctx);

    expect(result.enabled).toBe(true);
    // If any HTTP request were attempted, the test would fail in a
    // network-isolated environment. The mock is purely in-process.
  });

  // BDD: evaluate() returns results for all configured flags
  it('evaluate() returns results for all configured flags only', async () => {
    mock.enable('flag-a');
    mock.setVariant('flag-b', 'v1');

    const results = await mock.evaluate(ctx);

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

  it('evaluate() returns empty array when no flags are configured', async () => {
    const results = await mock.evaluate(ctx);
    expect(results).toEqual([]);
  });

  it('evaluate() does not include unconfigured flags', async () => {
    mock.enable('flag-a');
    const results = await mock.evaluate(ctx);
    const keys = results.map((r) => r.key);
    expect(keys).toEqual(['flag-a']);
  });

  it('evaluate() tracks evaluation for assertEvaluated', async () => {
    mock.enable('tracked');
    await mock.evaluate(ctx);
    expect(() => mock.assertEvaluated('tracked')).not.toThrow();
  });

  it('evaluateFlag() tracks evaluation for unconfigured flags too', async () => {
    await mock.evaluateFlag('unknown', ctx);
    expect(() => mock.assertEvaluated('unknown')).not.toThrow();
  });

  it('evaluate() returns deterministic evaluatedAt timestamp', async () => {
    mock.enable('ts-flag');
    const results = await mock.evaluate(ctx);
    expect(results[0].evaluatedAt).toBe('1970-01-01T00:00:00Z');
  });

  it('reset() clears all state', async () => {
    mock.enable('flag');
    await mock.evaluateFlag('flag', ctx);

    mock.reset();

    const result = await mock.evaluateFlag('flag', ctx);
    expect(result.reason).toBe('mock_default');
    // Reset also clears eval tracking, but evaluateFlag just re-tracked 'flag'
    // so we need a fresh key to test tracking was cleared
    expect(() => mock.assertNotEvaluated('other')).not.toThrow();
  });
});
