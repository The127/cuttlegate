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
 * const result = await mock.evaluateFlag('dark-mode', { user_id: 'u1', attributes: {} });
 * // { enabled: true, value: null, reason: 'mock' }
 *
 * mock.assertEvaluated('dark-mode');
 * ```
 */

import type { EvalContext } from './types.js';
import type {
  CuttlegateClient,
  EvaluationResult,
  FlagResult,
} from './client.js';

const MOCK_EVALUATED_AT = '1970-01-01T00:00:00Z';

interface FlagConfig {
  enabled: boolean;
  value: string | null;
  valueKey: string;
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
    async evaluate(_context: EvalContext): Promise<EvaluationResult[]> {
      const results: EvaluationResult[] = [];
      for (const [key, config] of flags) {
        evaluated.add(key);
        results.push({
          key,
          enabled: config.enabled,
          value: config.value,
          valueKey: config.valueKey,
          reason: 'mock',
          evaluatedAt: MOCK_EVALUATED_AT,
        });
      }
      return results;
    },

    async evaluateFlag(key: string, _context: EvalContext): Promise<FlagResult> {
      evaluated.add(key);
      const config = flags.get(key);
      if (!config) {
        return { enabled: false, value: null, valueKey: '', reason: 'mock_default' };
      }
      return {
        enabled: config.enabled,
        value: config.value,
        valueKey: config.valueKey,
        reason: 'mock',
      };
    },

    enable(key: string): void {
      flags.set(key, { enabled: true, value: null, valueKey: 'true' });
    },

    disable(key: string): void {
      flags.delete(key);
    },

    setVariant(key: string, value: string): void {
      flags.set(key, { enabled: true, value, valueKey: value });
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
