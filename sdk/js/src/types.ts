/**
 * Wire format types for the Cuttlegate v1 evaluation API.
 *
 * These types are defined here as a consumer-side contract — they are not
 * imported from the server. If the server changes a field name, the SDK takes
 * a breaking change and a major version bump. That is the correct relationship.
 *
 * Locked at Sprint 3 close. Do not change without a semver-major bump.
 */

/** Reason a particular variant was returned for a flag evaluation. */
export type EvalReason = 'disabled' | 'default' | 'rule_match' | 'rollout';

/** Type of the feature flag, determining the shape of `value`. */
export type FlagType = 'bool' | 'string' | 'number' | 'json';

/**
 * Response from `POST /api/v1/projects/:slug/environments/:env/flags/:key/evaluate`.
 *
 * `value` is `null` for `bool` flags; a string representing the variant key for
 * all other flag types. Deprecated — prefer `value_key`.
 *
 * `value_key` is always present for all flag types. For `bool` flags it is
 * `"true"` or `"false"`; for all other types it equals `value`.
 */
export interface EvalResponse {
  key: string;
  enabled: boolean;
  value: string | null; // deprecated — prefer value_key
  value_key: string;
  reason: EvalReason;
  type: FlagType;
}

/** User context sent with every evaluation request. */
export interface EvalContext {
  user_id: string;
  attributes: Record<string, string>;
}

/** Request body for the single-flag evaluation endpoint. */
export interface EvalRequest {
  context: EvalContext;
}

/** SDK configuration options. */
export interface CuttlegateConfig {
  /** Base URL of the Cuttlegate server, e.g. https://flags.example.com */
  baseUrl: string;
  /** Service account token for authentication. */
  token: string;
  /** Project slug, e.g. "my-project". */
  project: string;
  /** Environment slug to evaluate against, e.g. "production". */
  environment: string;
  /** Request timeout in milliseconds. Defaults to 5000. */
  timeout?: number;
  /** Custom fetch implementation. Defaults to the global `fetch`. */
  fetch?: typeof fetch;
  /**
   * Default flag values to use when the server is unreachable.
   * Keys are flag keys; values specify the fallback state.
   *
   * When a network error or timeout occurs, the SDK returns these defaults
   * with `reason: 'default_fallback'` instead of throwing.
   * Auth errors (401/403) still throw — defaults are not applied for auth failures.
   */
  defaults?: Record<string, { enabled: boolean; variant: string }>;
}

/** Standard error shape returned by the Cuttlegate API. */
export interface ApiError {
  error: string;
  message: string;
}

/**
 * Persistence interface for the CachedClient's flag cache.
 * Implementations control where flag state is stored between sessions.
 * The SDK ships with `noopFlagStore` (the default).
 */
export interface FlagStore {
  save(flags: FlagStoreEntry[]): Promise<void>;
  load(): Promise<FlagStoreEntry[]>;
}

/** A single flag entry as persisted by FlagStore. */
export interface FlagStoreEntry {
  key: string;
  enabled: boolean;
  value: string | null;
  variant: string;
  reason: string;
  evaluatedAt: string;
}

/** A FlagStore that does nothing. Save is a no-op; load returns an empty array. */
export const noopFlagStore: FlagStore = {
  save: () => Promise.resolve(),
  load: () => Promise.resolve([]),
};
