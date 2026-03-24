import { describe, it, expect, afterEach } from 'vitest';
import { OpenFeature } from '@openfeature/server-sdk';
import { CuttlegateOpenFeatureProvider } from './openfeature.js';
import { createMockClient } from './testing.js';

const noopLogger = { debug: () => {}, info: () => {}, warn: () => {}, error: () => {} };

describe('CuttlegateOpenFeatureProvider', () => {
  it('exposes provider metadata', () => {
    const mock = createMockClient();
    const provider = new CuttlegateOpenFeatureProvider(mock);
    expect(provider.metadata).toEqual({ name: 'cuttlegate' });
  });

  it('implements the Provider interface accepted by OpenFeature', async () => {
    const mock = createMockClient();
    mock.enable('dark-mode');
    const provider = new CuttlegateOpenFeatureProvider(mock);

    // This is the conformance test: setProviderAndWait accepts our provider
    // and getClient().getBooleanValue resolves through it.
    await OpenFeature.setProviderAndWait('cuttlegate-test', provider);
    const client = OpenFeature.getClient('cuttlegate-test');
    const value = await client.getBooleanValue('dark-mode', false, { targetingKey: 'u1' });

    expect(value).toBe(true);
    await OpenFeature.clearProviders();
  });

  describe('resolveBooleanEvaluation', () => {
    it('returns true for an enabled bool flag', async () => {
      const mock = createMockClient();
      mock.enable('dark-mode');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('dark-mode', false, { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toBe(true);
      expect(result.variant).toBe('true');
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns false for a disabled bool flag', async () => {
      const mock = createMockClient();
      mock.disable('dark-mode');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveBooleanEvaluation('dark-mode', true, { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toBe(false);
      expect(result.variant).toBe('false');
    });
  });

  describe('resolveStringEvaluation', () => {
    it('returns variant for a string flag', async () => {
      const mock = createMockClient();
      mock.setVariant('color', 'blue');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveStringEvaluation('color', 'red', { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toBe('blue');
      expect(result.variant).toBe('blue');
      expect(result.reason).toBe('TARGETING_MATCH');
    });
  });

  describe('resolveNumberEvaluation', () => {
    it('parses numeric variant', async () => {
      const mock = createMockClient();
      mock.setVariant('rate-limit', '42');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveNumberEvaluation('rate-limit', 10, { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toBe(42);
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns default on non-numeric variant', async () => {
      const mock = createMockClient();
      mock.setVariant('rate-limit', 'not-a-number');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveNumberEvaluation('rate-limit', 10, { targetingKey: 'u1' }, noopLogger);

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

      const result = await provider.resolveObjectEvaluation('config', {}, { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toEqual({ key: 'value' });
      expect(result.reason).toBe('TARGETING_MATCH');
    });

    it('returns default on invalid JSON', async () => {
      const mock = createMockClient();
      mock.setVariant('config', 'not-json');
      const provider = new CuttlegateOpenFeatureProvider(mock);

      const result = await provider.resolveObjectEvaluation('config', { fallback: true }, { targetingKey: 'u1' }, noopLogger);

      expect(result.value).toEqual({ fallback: true });
      expect(result.reason).toBe('ERROR');
    });
  });

  describe('OpenFeature SDK conformance', () => {
    afterEach(async () => {
      await OpenFeature.clearProviders();
    });

    it('resolves boolean through OpenFeature client', async () => {
      const mock = createMockClient();
      mock.enable('feature-x');
      await OpenFeature.setProviderAndWait('conformance-bool', new CuttlegateOpenFeatureProvider(mock));
      const client = OpenFeature.getClient('conformance-bool');

      const value = await client.getBooleanValue('feature-x', false, { targetingKey: 'u1' });
      expect(value).toBe(true);
    });

    it('resolves string through OpenFeature client', async () => {
      const mock = createMockClient();
      mock.setVariant('theme', 'dark');
      await OpenFeature.setProviderAndWait('conformance-str', new CuttlegateOpenFeatureProvider(mock));
      const client = OpenFeature.getClient('conformance-str');

      const value = await client.getStringValue('theme', 'light', { targetingKey: 'u1' });
      expect(value).toBe('dark');
    });

    it('resolves number through OpenFeature client', async () => {
      const mock = createMockClient();
      mock.setVariant('limit', '100');
      await OpenFeature.setProviderAndWait('conformance-num', new CuttlegateOpenFeatureProvider(mock));
      const client = OpenFeature.getClient('conformance-num');

      const value = await client.getNumberValue('limit', 50, { targetingKey: 'u1' });
      expect(value).toBe(100);
    });

    it('resolves object through OpenFeature client', async () => {
      const mock = createMockClient();
      mock.setVariant('cfg', '{"a":1}');
      await OpenFeature.setProviderAndWait('conformance-obj', new CuttlegateOpenFeatureProvider(mock));
      const client = OpenFeature.getClient('conformance-obj');

      const value = await client.getObjectValue('cfg', {}, { targetingKey: 'u1' });
      expect(value).toEqual({ a: 1 });
    });

    it('returns default on evaluation error', async () => {
      const mock = createMockClient();
      mock.setVariant('bad-num', 'NaN');
      await OpenFeature.setProviderAndWait('conformance-err', new CuttlegateOpenFeatureProvider(mock));
      const client = OpenFeature.getClient('conformance-err');

      const value = await client.getNumberValue('bad-num', 42, { targetingKey: 'u1' });
      expect(value).toBe(42);
    });
  });
});
