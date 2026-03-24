/**
 * @cuttlegate/sdk/testing — in-process mock for testing flag integrations.
 *
 * Provides a `MockCuttlegateClient` that implements `CuttlegateClient` without
 * network access. Consumers set flag state directly and assert evaluations.
 *
 * @example
 * ```ts
 * import { createMockClient } from '@cuttlegate/sdk/testing';
 *
 * const mock = createMockClient();
 * mock.enable('dark-mode');
 *
 * const result = await mock.evaluate('dark-mode', { user_id: 'u1', attributes: {} });
 * // { key: 'dark-mode', enabled: true, variant: 'true', reason: 'mock', ... }
 *
 * mock.assertEvaluated('dark-mode');
 * ```
 */

import type { EvalContext } from './types.js';
import type {
  CuttlegateClient,
  EvalResult,
  FlagResult,
} from './client.js';

const MOCK_EVALUATED_AT = '1970-01-01T00:00:00Z';

interface FlagConfig {
  enabled: boolean;
  value: string | null;
  variant: string;
}

export interface MockCuttlegateClient extends CuttlegateClient {
  enable(key: string): void;
  disable(key: string): void;
  setVariant(key: string, value: string): void;
  assertEvaluated(key: string): void;
  assertNotEvaluated(key: string): void;
  reset(): void;
}

export function createMockClient(): MockCuttlegateClient {
  const flags = new Map<string, FlagConfig>();
  const evaluated = new Set<string>();

  return {
    async evaluate(key: string, _context: EvalContext): Promise<EvalResult> {
      evaluated.add(key);
      const config = flags.get(key);
      if (!config) {
        return {
          key,
          enabled: false,
          value: null,
          variant: '',
          reason: 'mock_default',
          evaluatedAt: MOCK_EVALUATED_AT,
        };
      }
      return {
        key,
        enabled: config.enabled,
        value: config.value,
        variant: config.variant,
        reason: 'mock',
        evaluatedAt: MOCK_EVALUATED_AT,
      };
    },

    async evaluateAll(_context: EvalContext): Promise<EvalResult[]> {
      const results: EvalResult[] = [];
      for (const [key, config] of flags) {
        evaluated.add(key);
        results.push({
          key,
          enabled: config.enabled,
          value: config.value,
          variant: config.variant,
          reason: 'mock',
          evaluatedAt: MOCK_EVALUATED_AT,
        });
      }
      return results;
    },

    async bool(key: string, context: EvalContext): Promise<boolean> {
      const result = await this.evaluate(key, context);
      return result.variant === 'true';
    },

    async string(key: string, context: EvalContext): Promise<string> {
      const result = await this.evaluate(key, context);
      return result.variant;
    },

    /** @deprecated Use evaluate() instead. */
    async evaluateFlag(key: string, _context: EvalContext): Promise<FlagResult> {
      evaluated.add(key);
      const config = flags.get(key);
      if (!config) {
        return { enabled: false, value: null, variant: '', reason: 'mock_default' };
      }
      return {
        enabled: config.enabled,
        value: config.value,
        variant: config.variant,
        reason: 'mock',
      };
    },

    enable(key: string): void {
      flags.set(key, { enabled: true, value: null, variant: 'true' });
    },

    disable(key: string): void {
      flags.set(key, { enabled: false, value: null, variant: 'false' });
    },

    setVariant(key: string, value: string): void {
      flags.set(key, { enabled: true, value, variant: value });
    },

    assertEvaluated(key: string): void {
      if (!evaluated.has(key)) {
        throw new Error(
          `Expected flag "${key}" to have been evaluated, but it was not`,
        );
      }
    },

    assertNotEvaluated(key: string): void {
      if (evaluated.has(key)) {
        throw new Error(
          `Expected flag "${key}" to NOT have been evaluated, but it was`,
        );
      }
    },

    reset(): void {
      flags.clear();
      evaluated.clear();
    },
  };
}
