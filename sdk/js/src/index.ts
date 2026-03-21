/**
 * @cuttlegate/sdk — Node.js / ESM entry point.
 *
 * Feature client implementation lives in #61 (init & config) and #62 (evaluation).
 * This file establishes the public type surface for those issues to build on.
 */
export type {
  ApiError,
  CuttlegateConfig,
  EvalContext,
  EvalReason,
  EvalRequest,
  EvalResponse,
  FlagType,
} from './types.js';
