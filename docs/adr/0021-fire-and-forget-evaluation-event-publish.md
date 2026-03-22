# ADR 0021: Fire-and-forget goroutine for evaluation event publish

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #202

## Context

The evaluation service publishes an `EvaluationEvent` after each flag evaluation to populate the audit trail. The publish operation is a database write via `PostgresEvaluationEventRepository`. Two approaches were considered:

1. **Synchronous publish** — call `publisher.Publish()` in the request path before returning the evaluation result. The HTTP response is held until the publish completes. If the publish fails, the caller receives an error even though the evaluation itself succeeded.

2. **Fire-and-forget goroutine** — call `publisher.Publish()` in a goroutine after building the result. Return the evaluation response immediately. Log publish errors but never propagate them to the caller.

## Decision

**Fire-and-forget goroutine.** Evaluation event publishing is best-effort. The implementation in `EvaluationService.publishEvent` serialises `evalCtx.Attributes` (a map) in the calling goroutine before spawning to avoid a data race, then spawns a single goroutine for the IO. Errors are logged via `slog.Error`.

## Rationale

- **Evaluation latency is on the critical path; auditing is not.** SDK clients call the evaluation endpoint in the hot path of application requests. Adding a synchronous DB write to that path increases latency and creates a hard dependency on audit DB availability.
- **Partial audit data is better than evaluation errors.** If the audit write fails, the flag still evaluated correctly. A missed audit record is recoverable in context; an evaluation error propagated to the SDK caller is not.
- **Reliable delivery (outbox, WAL tailing) is out of scope at this stage.** A transactional outbox would eliminate the gap but is significant infrastructure overhead for a feature-flag tool in its current phase. The gap is documented and accepted.

## Consequences

- Evaluation events are best-effort. Under database pressure, events may be silently dropped without user-visible error.
- The `EvaluationEventPublisher` interface contract is implicitly "best-effort" — implementations must not assume that publish errors reach the caller.
- Goroutine lifetime is bounded: one goroutine per `publishEvent` call, exits after one DB write. No retry loop; no leak risk beyond request volume.
- The goroutine uses `context.Background()`, not the request context — it continues even if the HTTP request is cancelled.
- If audit completeness becomes a product requirement (e.g. compliance use cases), supersede this ADR with one that introduces reliable delivery.
