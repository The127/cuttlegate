# ADR 0022: Evaluation event fanout to multiple independent subscribers

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #152, #202

## Context

The evaluation hot path publishes an `EvaluationEvent` after each flag evaluation. Two independent consumers now exist:

1. **Audit trail** (`PostgresEvaluationEventRepository`) — appends a raw evaluation record to `evaluation_events` for the debugging view. Introduced in #202.
2. **Evaluation stats** — upserts an aggregate row in `flag_evaluation_stats` (`last_evaluated_at`, `evaluation_count`) for the dashboard stats display. Introduced in #152.

Both consumers receive the same `EvaluationEvent` payload. They have different write patterns (append vs. upsert), different retention requirements (30-day rolling vs. indefinite), and different failure tolerances (both are best-effort per ADR 0021).

The architecture must decide: one publisher that fans out to multiple subscribers, or separate publish calls per consumer.

Three options:

1. **Single `EvaluationEventPublisher` port, one implementation that fans out** — the port interface has one method; the adapter internally dispatches to both writers. The app layer calls `publisher.Publish()` once.
2. **Two separate ports** — `EvaluationAuditPublisher` and `EvaluationStatsPublisher`; the app layer calls both explicitly.
3. **In-process event bus** — events are dispatched to registered listeners by type; app layer publishes to a bus, consumers register themselves at startup.

## Decision

**Option 1: single `EvaluationEventPublisher` port, one fire-and-forget call from the app layer.** The fanout to multiple writers is an implementation detail of the adapter, not a concern of the app layer.

The current implementation uses a single goroutine (per ADR 0021) that calls both the audit writer and the stats upsert sequentially. If the audit write fails, the stats write still proceeds (and vice versa). Errors from either are logged but not propagated.

## Rationale

Option 2 would expose infrastructure routing decisions to the app layer — `EvaluationService` would need to know that two separate systems care about evaluations, which is adapter-level knowledge. Option 3 is appropriate for larger systems with many event types but is over-engineered for two consumers.

Option 1 keeps the app layer clean: one port, one call. The adapter owns the fan-out. Adding a third consumer (e.g. a real-time analytics stream) requires only an adapter change, not an app-layer change.

## Consequences

- Adding a new evaluation event consumer requires changing the adapter implementation and wiring in `cmd/server/`, not the app service.
- Both consumers share the same goroutine per ADR 0021 — a slow consumer will delay the other. If one consumer becomes significantly slower (e.g. external HTTP call), introduce a dedicated goroutine per consumer at that point and update this ADR.
- The `EvaluationEventPublisher` port interface must remain stable — it is called from `EvaluationService` and implemented by the adapter that owns the fanout.
- Failure isolation between consumers is partial: a panic in one consumer's write function will take down the shared goroutine. Each consumer write must recover from panics or errors independently.
