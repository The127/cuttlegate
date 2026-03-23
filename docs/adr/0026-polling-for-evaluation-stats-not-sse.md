# ADR 0026: Polling for evaluation stats refresh, not SSE

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #243

## Context

The evaluation stats panel in the management UI (#243) needs to stay reasonably fresh so operators see recent flag evaluation counts and ratios without a manual page reload. Two approaches were available:

1. **Polling** — the SPA calls the stats REST endpoint on a fixed interval (e.g. every 30 seconds).
2. **SSE subscription** — add a new SSE event type for stats updates and push them server-side.

The codebase already has an SSE infrastructure (`Broker`, `SSEHandler`, `EvaluationEventPublisher`) used for real-time flag state change delivery to SDK clients.

## Decision

Use periodic client-side polling for the evaluation stats panel. Do not add an SSE event type for stats.

## Rationale

- **Stats are not real-time critical.** Flag state changes (enable/disable, variant switch) need sub-second delivery to SDK clients — that is what the SSE infrastructure is for. Evaluation counts becoming stale by 30–60 seconds has no operational consequence.
- **Polling avoids a new server-side concern.** An SSE stats event would require the server to decide when to emit it (after every evaluation? batched at intervals?), which is a non-trivial design question. Polling pushes that concern to the client, where it belongs for non-critical data.
- **Adding SSE event types is an SDK contract concern.** The `flag.state_changed` event shape is a locked v1 SDK contract (ADR 0018, project memory). Adding stats events to the same SSE stream risks coupling the SDK contract to an internal UI concern. Keeping the SSE stream SDK-only is a cleaner boundary.
- **Simplicity.** Polling is a `setInterval` + fetch call. It requires no server changes, no new event types, and no broker modifications.

## Consequences

- The stats panel will lag the server by the polling interval. This is accepted.
- Future developers who see polling next to SSE infrastructure should not "fix" it — the choice is intentional.
- If a use case arises where sub-second stats freshness is genuinely needed (e.g. a live operations dashboard), revisit this decision and consider a dedicated WebSocket or a separate SSE endpoint that does not share the SDK stream.
- Any new UI feature that requires live data should default to polling first, and escalate to SSE only if the latency requirement genuinely demands it.
