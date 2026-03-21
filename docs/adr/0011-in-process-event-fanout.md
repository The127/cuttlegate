# ADR 0011: In-process event fanout for SSE delivery

**Date:** 2026-03-21
**Status:** Accepted
**Issue:** #46

## Context

Cuttlegate needs to deliver flag state change events to connected clients in real time via Server-Sent Events. This requires a fanout mechanism that accepts published domain events and distributes them to all active SSE subscribers.

Three options were considered:

1. **In-process broker** — a Go struct with channels and a mutex, running inside the server process.
2. **Redis Pub/Sub** — external message broker, supports multiple server instances.
3. **NATS** — dedicated messaging system, supports multiple server instances with more features than Redis.

The current deployment target is a single server instance.

## Decision

Use an in-process broker (`Broker` struct in `internal/adapters/http/`) that implements `ports.EventPublisher`. The broker fans out events to subscribers via buffered Go channels with non-blocking send semantics.

### SSE wire format contract

Events are delivered as SSE `data:` lines with JSON payloads. Field names use `snake_case` to match the REST API convention:

```
data: {"type":"flag.state_changed","project":"my-project","environment":"production","flag_key":"dark-mode","enabled":true,"occurred_at":"2026-03-21T12:00:00Z"}\n\n
```

The `type` field is required — SDK consumers use it for event dispatch.

### Design choices

- **Non-blocking publish:** if a subscriber's channel buffer is full, the event is dropped for that subscriber. This prevents one slow client from blocking event delivery to all others.
- **No replay:** the broker does not persist events or support `Last-Event-ID` replay. SSE clients that reconnect receive only new events — they must re-evaluate flag state on reconnect.
- **Shutdown ordering:** the HTTP server must drain in-flight requests (`srv.Shutdown()`) before the broker is shut down (`broker.Shutdown()`). Reversing this order breaks in-flight SSE connections with closed channels.

## Rationale

An in-process broker is the simplest solution that works for single-instance deployment. It requires no external dependencies, no configuration, and no network round-trips. Redis Pub/Sub or NATS would add operational complexity with no benefit until horizontal scaling is needed.

The `EventPublisher` port interface decouples the app layer from the delivery mechanism. Swapping to Redis Pub/Sub later requires only a new adapter that implements the same interface — no app or domain changes.

## Consequences

- **Easier:** no external infrastructure to deploy, configure, or monitor. Zero additional dependencies.
- **Harder:** horizontal scaling requires replacing this adapter with a distributed alternative (Redis Pub/Sub, NATS, or similar). This is a deliberate tradeoff — the interface boundary makes the swap mechanical.
- **Keep in sync:** shutdown ordering in `cmd/server/main.go` must respect the contract: HTTP server first, then broker.
- **Event filtering:** the broker broadcasts all events. Per-project/environment filtering is the SSE handler's responsibility (#47), not the broker's.
