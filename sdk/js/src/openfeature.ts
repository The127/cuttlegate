/**
 * OpenFeature Provider for Cuttlegate.
 *
 * Implements the `Provider` interface from `@openfeature/server-sdk`.
 *
 * @example
 * ```ts
 * import { OpenFeature } from '@openfeature/server-sdk';
 * import { createClient } from '@cuttlegate/sdk';
 * import { CuttlegateProvider } from '@cuttlegate/sdk/openfeature';
 *
 * const client = createClient({ ... });
 * await OpenFeature.setProviderAndWait(new CuttlegateProvider(client));
 *
 * const ofClient = OpenFeature.getClient();
 * const value = await ofClient.getBooleanValue('dark-mode', false);
 * ```
 */

import type {
  EvaluationContext,
  JsonValue,
  Logger,
  Provider,
  ResolutionDetails,
} from '@openfeature/server-sdk';
import { ErrorCode, ServerProviderStatus } from '@openfeature/server-sdk';
import type { CuttlegateClient } from './client.js';
import type { EvalContext } from './types.js';

/**
 * CuttlegateOpenFeatureProvider implements the OpenFeature `Provider`
 * interface backed by a CuttlegateClient.
 */
export class CuttlegateOpenFeatureProvider implements Provider {
  readonly metadata = { name: 'cuttlegate' } as const;
  status: ServerProviderStatus = ServerProviderStatus.READY;
  readonly hooks = [];

  private client: CuttlegateClient;

  constructor(client: CuttlegateClient) {
    this.client = client;
  }

  private toEvalContext(ctx: EvaluationContext): EvalContext {
    const attributes: Record<string, string> = {};
    for (const [k, v] of Object.entries(ctx)) {
      if (k !== 'targetingKey' && typeof v === 'string') {
        attributes[k] = v;
      }
    }
    return {
      user_id: ctx.targetingKey ?? '',
      attributes,
    };
  }

  async resolveBooleanEvaluation(
    flagKey: string,
    defaultValue: boolean,
    context: EvaluationContext,
    _logger: Logger,
  ): Promise<ResolutionDetails<boolean>> {
    try {
      const result = await this.client.bool(flagKey, this.toEvalContext(context));
      return { value: result, variant: result ? 'true' : 'false', reason: 'TARGETING_MATCH' };
    } catch (err) {
      return {
        value: defaultValue,
        reason: 'ERROR',
        errorCode: ErrorCode.GENERAL,
        errorMessage: err instanceof Error ? err.message : 'Unknown error',
      };
    }
  }

  async resolveStringEvaluation(
    flagKey: string,
    defaultValue: string,
    context: EvaluationContext,
    _logger: Logger,
  ): Promise<ResolutionDetails<string>> {
    try {
      const result = await this.client.string(flagKey, this.toEvalContext(context));
      return { value: result, variant: result, reason: 'TARGETING_MATCH' };
    } catch (err) {
      return {
        value: defaultValue,
        reason: 'ERROR',
        errorCode: ErrorCode.GENERAL,
        errorMessage: err instanceof Error ? err.message : 'Unknown error',
      };
    }
  }

  async resolveNumberEvaluation(
    flagKey: string,
    defaultValue: number,
    context: EvaluationContext,
    _logger: Logger,
  ): Promise<ResolutionDetails<number>> {
    try {
      const result = await this.client.string(flagKey, this.toEvalContext(context));
      const num = Number(result);
      if (Number.isNaN(num)) {
        return { value: defaultValue, reason: 'ERROR', errorCode: ErrorCode.PARSE_ERROR, errorMessage: `Cannot parse "${result}" as number` };
      }
      return { value: num, variant: result, reason: 'TARGETING_MATCH' };
    } catch (err) {
      return {
        value: defaultValue,
        reason: 'ERROR',
        errorCode: ErrorCode.GENERAL,
        errorMessage: err instanceof Error ? err.message : 'Unknown error',
      };
    }
  }

  async resolveObjectEvaluation<T extends JsonValue>(
    flagKey: string,
    defaultValue: T,
    context: EvaluationContext,
    _logger: Logger,
  ): Promise<ResolutionDetails<T>> {
    try {
      const result = await this.client.string(flagKey, this.toEvalContext(context));
      const parsed = JSON.parse(result) as T;
      return { value: parsed, variant: result, reason: 'TARGETING_MATCH' };
    } catch (err) {
      return {
        value: defaultValue,
        reason: 'ERROR',
        errorCode: ErrorCode.GENERAL,
        errorMessage: err instanceof Error ? err.message : 'Unknown error',
      };
    }
  }
}
