import type { CuttlegateConfig, EvalContext } from './types.js';

/** Result of evaluating all flags for a context. */
export interface EvaluationResult {
  key: string;
  enabled: boolean;
  value: string | null;
  reason: string;
  evaluatedAt: string;
}

/** Result of evaluating a single flag by key. */
export interface FlagResult {
  enabled: boolean;
  value: string | null;
  reason: string;
}

/** SDK client interface — implemented by both the real client and test mocks (#108). */
export interface CuttlegateClient {
  evaluate(context: EvalContext): Promise<EvaluationResult[]>;
  evaluateFlag(key: string, context: EvalContext): Promise<FlagResult>;
}

/** Structured error thrown by the SDK. */
export class CuttlegateError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.name = 'CuttlegateError';
    this.code = code;
  }
}

const DEFAULT_TIMEOUT = 5000;

/**
 * Create a Cuttlegate SDK client.
 *
 * Validates config synchronously — throws on missing required fields.
 * The token is closure-scoped and never exposed on the returned object.
 */
export function createClient(config: CuttlegateConfig): CuttlegateClient {
  if (!config.baseUrl) {
    throw new CuttlegateError('invalid_config', 'baseUrl is required');
  }
  if (!config.token) {
    throw new CuttlegateError('invalid_config', 'token is required');
  }
  if (!config.environment) {
    throw new CuttlegateError('invalid_config', 'environment is required');
  }

  const _baseUrl = config.baseUrl.replace(/\/+$/, '');
  const _token = config.token;
  const _environment = config.environment;
  const _timeout = config.timeout ?? DEFAULT_TIMEOUT;
  const _fetch = config.fetch ?? globalThis.fetch;

  // Suppress unused-variable lint until #62 implements these.
  void _baseUrl;
  void _token;
  void _environment;
  void _timeout;
  void _fetch;

  return {
    evaluate(_context: EvalContext): Promise<EvaluationResult[]> {
      throw new CuttlegateError(
        'not_implemented',
        'evaluate() is not yet implemented — see #62',
      );
    },

    evaluateFlag(_key: string, _context: EvalContext): Promise<FlagResult> {
      throw new CuttlegateError(
        'not_implemented',
        'evaluateFlag() is not yet implemented — see #62',
      );
    },
  };
}
