import { z } from 'zod';
import type { CuttlegateConfig, EvalContext } from './types.js';

/** Result of evaluating a flag. Primary result type for evaluate() and evaluateAll(). */
export interface EvalResult {
  key: string;
  enabled: boolean;
  /** @deprecated Prefer variant — null for bool flags. */
  value: string | null;
  /** Always present. "true"/"false" for bool flags, variant key for all others. */
  variant: string;
  reason: string;
  evaluatedAt: string;
}

/**
 * @deprecated Use EvalResult instead. This alias will be removed in the next major version.
 */
export type EvaluationResult = EvalResult;

/** Result of evaluating a single flag by key (deprecated shape — prefer EvalResult). */
export interface FlagResult {
  enabled: boolean;
  /** @deprecated Prefer variant — null for bool flags. */
  value: string | null;
  /** Always present. "true"/"false" for bool flags, variant key for all others. */
  variant: string;
  reason: string;
}

/** SDK client interface — implemented by both the real client and test mocks (#108). */
export interface CuttlegateClient {
  /** Evaluate a single flag by key. Returns a full EvalResult. */
  evaluate(key: string, context: EvalContext): Promise<EvalResult>;
  /** Evaluate all flags in one HTTP request. */
  evaluateAll(context: EvalContext): Promise<EvalResult[]>;
  /** Convenience: evaluate a boolean flag. Returns true if variant is "true". */
  bool(key: string, context: EvalContext): Promise<boolean>;
  /** Convenience: evaluate a string flag. Returns the variant string. */
  string(key: string, context: EvalContext): Promise<string>;
  /**
   * @deprecated Use evaluate() instead.
   */
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

const BulkEvaluateResponseSchema = z.object({
  flags: z.array(
    z.object({
      key: z.string(),
      enabled: z.boolean(),
      value: z.string().nullable(),
      value_key: z.string().default(''),
      reason: z.string(),
      type: z.string(),
    }),
  ),
  evaluated_at: z.string(),
});

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
  if (!config.project) {
    throw new CuttlegateError('invalid_config', 'project is required');
  }
  if (!config.environment) {
    throw new CuttlegateError('invalid_config', 'environment is required');
  }

  const baseUrl = config.baseUrl.replace(/\/+$/, '');
  const token = config.token;
  const project = config.project;
  const environment = config.environment;
  const timeout = config.timeout ?? DEFAULT_TIMEOUT;
  const fetchFn = config.fetch ?? globalThis.fetch;
  const defaults = config.defaults ?? {};

  const evaluateUrl = `${baseUrl}/api/v1/projects/${encodeURIComponent(project)}/environments/${encodeURIComponent(environment)}/evaluate`;

  function defaultFallbacks(): EvalResult[] {
    return Object.entries(defaults).map(([key, def]) => ({
      key,
      enabled: def.enabled,
      value: null,
      variant: def.variant,
      reason: 'default_fallback',
      evaluatedAt: '',
    }));
  }

  async function evaluateAll(context: EvalContext): Promise<EvalResult[]> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeout);

    let res: Response;
    try {
      res = await fetchFn(evaluateUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ context }),
        signal: controller.signal,
      });
    } catch (err: unknown) {
      clearTimeout(timer);
      // Fall back to defaults on network/timeout errors (if defaults are configured).
      if (Object.keys(defaults).length > 0) {
        return defaultFallbacks();
      }
      if (err instanceof DOMException && err.name === 'AbortError') {
        throw new CuttlegateError('timeout', `Request timed out after ${timeout}ms`);
      }
      throw new CuttlegateError('network_error', err instanceof Error ? err.message : 'Network request failed');
    } finally {
      clearTimeout(timer);
    }

    if (res.status === 401) {
      throw new CuttlegateError('unauthorized', 'Server returned 401 Unauthorized');
    }
    if (res.status === 403) {
      throw new CuttlegateError('forbidden', 'Server returned 403 Forbidden');
    }
    if (!res.ok) {
      // Fall back on server errors if defaults are configured.
      if (Object.keys(defaults).length > 0 && res.status >= 500) {
        return defaultFallbacks();
      }
      throw new CuttlegateError('network_error', `Server returned ${res.status}`);
    }

    let json: unknown;
    try {
      json = await res.json();
    } catch {
      throw new CuttlegateError('invalid_response', 'Response is not valid JSON');
    }

    const parsed = BulkEvaluateResponseSchema.safeParse(json);
    if (!parsed.success) {
      throw new CuttlegateError('invalid_response', 'Server returned an unexpected response shape');
    }

    const evaluatedAt = parsed.data.evaluated_at;
    return parsed.data.flags.map((flag) => ({
      key: flag.key,
      enabled: flag.enabled,
      value: flag.value,
      variant: flag.value_key || flag.value || '',
      reason: flag.reason,
      evaluatedAt,
    }));
  }

  async function evaluate(key: string, context: EvalContext): Promise<EvalResult> {
    const results = await evaluateAll(context);
    const match = results.find((r) => r.key === key);
    if (!match) {
      throw new CuttlegateError('not_found', `Flag "${key}" not found`);
    }
    return match;
  }

  async function bool(key: string, context: EvalContext): Promise<boolean> {
    const result = await evaluate(key, context);
    return result.variant === 'true';
  }

  async function string(key: string, context: EvalContext): Promise<string> {
    const result = await evaluate(key, context);
    return result.variant;
  }

  /** @deprecated Use evaluate() instead. */
  async function evaluateFlag(key: string, context: EvalContext): Promise<FlagResult> {
    const results = await evaluateAll(context);
    const match = results.find((r) => r.key === key);
    if (!match) {
      return { enabled: false, value: null, variant: '', reason: 'not_found' };
    }
    return { enabled: match.enabled, value: match.value, variant: match.variant, reason: match.reason };
  }

  return { evaluate, evaluateAll, bool, string, evaluateFlag };
}
