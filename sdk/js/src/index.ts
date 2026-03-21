/**
 * @cuttlegate/sdk — Node.js / ESM entry point.
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
