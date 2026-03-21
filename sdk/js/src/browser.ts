/**
 * @cuttlegate/sdk — browser entry point.
 *
 * Selected by bundlers that honour the `browser` condition in the package.json
 * `exports` map (Vite, webpack, esbuild, Rollup).
 * Must not import any Node built-ins (fs, crypto, path, etc.).
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

export { createClient, CuttlegateError } from './client.js';
export type { CuttlegateClient, EvaluationResult, FlagResult } from './client.js';

export { connectStream } from './streaming.js';
export type { StreamOptions, StreamConnection, FlagStateChangedEvent } from './streaming.js';
