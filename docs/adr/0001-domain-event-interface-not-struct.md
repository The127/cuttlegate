# ADR 0001: DomainEvent as interface, not struct

**Date:** 2026-03-20
**Status:** Accepted
**Issue:** #36

## Context

When defining the `EventPublisher` port, we needed a common type for domain events that could carry flag evaluation, flag change, and audit payloads. Two options were considered:

1. A concrete struct with a generic payload field (`Payload any`)
2. A Go interface that concrete event types implement

## Decision

`DomainEvent` is a Go interface:

```go
type DomainEvent interface {
    EventType() string
    OccurredAt() time.Time
}
```

Concrete event types (defined in Sprint 2+ alongside the use cases that produce them) implement this interface and carry their own typed fields.

## Rationale

A struct with `Payload any` requires consumers to type-assert the payload to extract typed data. Type assertions are unchecked at compile time — a consumer that handles `FlagChangedEvent` silently does nothing if a new event type arrives with a different concrete type. Bugs surface at runtime, not build time.

An interface allows each event type to carry its own strongly-typed fields. Consumers can switch on the concrete type; the compiler catches unhandled cases in exhaustive checks. New event types are added without modifying the `DomainEvent` definition.

## Consequences

- Every event type produced by the domain must implement `EventType() string` and `OccurredAt() time.Time`
- `EventPublisher` implementations accept any `DomainEvent` — they must handle unknown types gracefully (log and skip, or return an error)
- No event type constants are defined in this issue; they are the responsibility of the use-case layer in Sprint 2+
