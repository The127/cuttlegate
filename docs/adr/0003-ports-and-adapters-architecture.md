# ADR 0003: Ports and adapters architecture

**Date:** 2026-03-20
**Status:** Accepted
**Issue:** #17

## Context

The project needed a structural pattern that would allow the core business logic (feature flag evaluation, project management) to be developed and tested independently of infrastructure choices (database engine, HTTP framework, auth provider). Early decisions about PostgreSQL and `net/http` should not bleed into the domain model.

Two patterns were considered:

1. **Layered / MVC** — traditional horizontal layers (model, service, controller) with no strict import enforcement
2. **Ports & adapters (hexagonal)** — domain at the centre, ports as Go interfaces, adapters as outer implementations; enforced by architecture tests

## Decision

Cuttlegate uses ports & adapters:

- `internal/domain/` — pure Go entities and value objects; stdlib only
- `internal/domain/ports/` — Go interfaces defining what the domain needs from the outside world
- `internal/app/` — use-case services that orchestrate domain objects via ports
- `internal/adapters/` — concrete implementations of ports using real infrastructure
- `cmd/` — wiring layer; constructs and injects adapters; no business logic

Import direction: `cmd → adapters → app → domain`. Enforced by `arch_test.go`.

## Rationale

**Why not layered / MVC:**

Traditional layers do not prevent upward imports — a model importing a service, or a service importing a controller, is a common failure mode as codebases grow. The pattern relies on discipline, not tooling. In a project with AI-assisted development, relying on discipline alone is insufficient; the architecture test must catch violations automatically.

**Why ports & adapters:**

The interface boundary between `app` and `adapters` makes the database and HTTP adapter swappable without touching business logic. More practically, it makes unit tests for use-case logic possible without a running database — inject a fake port implementation, test the logic. The arch test (`go test ./...`) catches any import that violates the direction rule before it merges.

## Consequences

- All framework types (`*http.Request`, `*sql.DB`, etc.) are banned from `domain` and `app`
- Every new infrastructure capability requires a new port interface before any adapter is written
- `arch_test.go` must be updated when new packages are added (see #79)
- The wiring layer (`cmd/server/`) grows as adapters are added — it is the only place where concrete types are assembled
- Unit tests for `app` services are cheap: no container, no network, no database
