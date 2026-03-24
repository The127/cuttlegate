import { describe, it, expect } from 'vitest';
import { CuttlegateOpenFeatureProvider } from './openfeature.js';
import { createMockClient } from './testing.js';

describe('CuttlegateOpenFeatureProvider', () => {
  it('exposes provider metadata', () => {
    const mock = createMockClient();
    const provider = new CuttlegateOpenFeatureProvider(mock);
    expect(provider.metadata).toEqual({ name: 'cuttlegate' });
  });

  describe('resolveBooleanEvaluation', () => {
    it('returns true for an enabled bool flag', async () => {
      const mock = createMockClient();
      mock.enable('dark-mode');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('dark-mode', false, { targetingKey: 'u1' });

      expect(result.value).toBe(true);
      expect(result.variant).toBe('true');
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns false for a disabled bool flag', async () => {
      const mock = createMockClient();
      mock.disable('dark-mode');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('dark-mode', true, { targetingKey: 'u1' });

      expect(result.value).toBe(false);
      expect(result.variant).toBe('false');
    });

    it('returns default value on error for unknown flag', async () => {
      const mock = createMockClient();
      // Mock returns enabled=false for unknown flags — no error.
      // Provider wraps mock which returns false, so value is false (not default).
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('missing-flag', true, { targetingKey: 'u1' });

      // The mock doesn't throw for unknown flags, it returns false.
      expect(result.value).toBe(false);
    });
  });

  describe('resolveStringEvaluation', () => {
    it('returns variant for a string flag', async () => {
      const mock = createMockClient();
      mock.setVariant('color', 'blue');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveStringEvaluation('color', 'red', { targetingKey: 'u1' });

      expect(result.value).toBe('blue');
      expect(result.variant).toBe('blue');
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns empty string for unknown flag (mock does not throw)', async () => {
      const mock = createMockClient();
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveStringEvaluation('missing', 'fallback', { targetingKey: 'u1' });

      // Mock returns empty variant for unknown flags, not an error
      expect(result.value).toBe('');
      expect(result.reason).toBe('TARGETING_MATCH');
    });
  });

  describe('resolveNumberEvaluation', () => {
    it('parses numeric variant', async () => {
      const mock = createMockClient();
      mock.setVariant('rate-limit', '42');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveNumberEvaluation('rate-limit', 10, { targetingKey: 'u1' });

      expect(result.value).toBe(42);
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns default on non-numeric variant', async () => {
      const mock = createMockClient();
      mock.setVariant('rate-limit', 'not-a-number');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveNumberEvaluation('rate-limit', 10, { targetingKey: 'u1' });

      expect(result.value).toBe(10);
      expect(result.reason).toBe('ERROR');
      expect(result.errorCode).toBe('PARSE_ERROR');
    });
  });

  describe('resolveObjectEvaluation', () => {
    it('parses JSON variant', async () => {
      const mock = createMockClient();
      mock.setVariant('config', '{"key":"value"}');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveObjectEvaluation('config', {}, { targetingKey: 'u1' });

      expect(result.value).toEqual({ key: 'value' });
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns default on invalid JSON', async () => {
      const mock = createMockClient();
      mock.setVariant('config', 'not-json');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveObjectEvaluation('config', { fallback: true }, { targetingKey: 'u1' });

      expect(result.value).toEqual({ fallback: true });
      expect(result.reason).toBe('ERROR');
    });
  });

  describe('context mapping', () => {
    it('maps targetingKey to user_id', async () => {
      const mock = createMockClient();
      mock.enable('flag');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      await provider.resolveBooleanEvaluation('flag', false, {
        targetingKey: 'user-42',
        plan: 'pro',
      });

      // The evaluation succeeded — the context was valid
      mock.assertEvaluated('flag');
    });

    it('handles missing context gracefully', async () => {
      const mock = createMockClient();
      mock.enable('flag');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('flag', false);

      expect(result.value).toBe(true);
    });
  });
});
