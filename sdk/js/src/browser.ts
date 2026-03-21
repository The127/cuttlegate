/**
 * @cuttlegate/sdk — browser entry point.
 *
 * This entry point is selected by bundlers that honour the `browser` condition
 * in the package.json `exports` map (Vite, webpack, esbuild, Rollup).
 * It must not import any Node built-ins (fs, crypto, path, etc.).
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
