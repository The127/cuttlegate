# Engineering Principles

_What this project believes — and what it explicitly does not do._

This document is for human contributors joining Cuttlegate for the first time. It explains the values behind the codebase so you understand _why_ things are the way they are before you write your first line of code.

For individual architectural decisions and their full reasoning, see `docs/adr/`. For per-package responsibilities, see the `doc.go` file in each package.

---

## Architecture

Cuttlegate uses **ports and adapters** (hexagonal architecture). The domain model is pure Go — no framework types, no infrastructure imports, stdlib only. The application layer orchestrates domain objects through port interfaces. Adapters implement those interfaces with real infrastructure and live at the outer ring.

The import direction is strict and one-way:

```
cmd → adapters → app → domain/ports → domain
```

This is not a guideline. It is enforced by `arch_test.go`, which runs on every `go test ./...` invocation. If you add an import that violates the direction, the build fails before you can merge. **We enforce architecture with tests, not discipline.**

Why this matters: the domain and app layers can be tested without a database, an HTTP server, or any running infrastructure. A fake port implementation is all you need. This is the single biggest advantage of the pattern — cheap, fast, reliable unit tests for business logic.

See [ADR-0003](adr/0003-ports-and-adapters-architecture.md) for the full decision record.

## Testability

We use a three-tier test pyramid:

| Tier | What it tests | Requires infrastructure |
|---|---|---|
| **Unit** | Domain logic, app services, HTTP handlers | No |
| **Integration** | DB adapters against real Postgres | Yes (testcontainers) |
| **E2E** | Full HTTP stack, end-to-end | Yes |

Unit tests are the foundation. They are fast, run without Docker, and cover the domain and app layers thoroughly. Integration tests use **testcontainers-go** to spin up a real Postgres instance programmatically — no `docker-compose.yml` to drift out of sync. Each test owns its database lifecycle.

Architecture tests (`arch_test.go`) are first-class citizens, not a nice-to-have. They guard layer boundaries, import direction, and naming conventions. Breaking an arch test is the same as breaking a unit test — you fix it before you merge.

RBAC is tested in the app layer, not the HTTP layer, because authorization is a business rule — not a transport concern. Every app-layer service method calls `requireRole` before accessing data. See [ADR-0008](adr/0008-rbac-in-app-layer.md).

See [ADR-0002](adr/0002-test-pyramid-strategy.md) for the full test strategy decision.

## Maintainability

**Naming is load-bearing.** Suffixes tell you what a type does and where it lives:

- `Repository` — persistence port interface (in `domain/ports`)
- `Service` — use-case orchestrator (in `app`)
- `Handler` — HTTP handler (in `adapters/http`)
- `Postgres*Repository` — SQL implementation of a port (in `adapters/db`)

If you're unsure where something goes, the name tells you. If the name doesn't tell you, the name is wrong.

**Every package has a `doc.go`.** It answers three questions: what this package owns, what it deliberately does not own, and what to look at first. Twenty lines max, no implementation code. New packages must include one — this is not optional.

**No magic.** Code should be readable without knowing the framework, the ORM, or the build system. We use `net/http` directly, write SQL by hand, and wire dependencies explicitly in `cmd/server/`. If you need to understand how a request gets from the router to the database, you can follow the call chain without jumping through reflection, code generation, or annotation processing.

**Small, focused files.** A file that does one thing is easier to find, easier to review, and easier to delete. If a file is growing beyond its original responsibility, split it.

## What we don't do

These are approaches we considered and rejected. Each has a rationale — if you find yourself reaching for one of these, read the reasoning first.

**No framework types in domain or app.**
`*http.Request`, `http.ResponseWriter`, `*sql.DB`, and all other infrastructure types are banned from `domain/` and `app/`. Port interfaces use only domain types, `context.Context`, `time.Time`, and `error`. This keeps business logic testable without infrastructure and prevents the domain from coupling to transport or storage choices. See [ADR-0003](adr/0003-ports-and-adapters-architecture.md).

**No ORM.**
We write SQL by hand. ORMs obscure what queries actually execute, make it harder to optimize, and introduce a mapping layer that drifts from the schema over time. Hand-written SQL in adapter files is explicit, reviewable, and easy to profile.

**No mocks for integration tests.**
Integration tests hit a real Postgres instance via testcontainers. Mocks for database tests create a false sense of confidence — they test your mock, not your queries. We use mocks only in unit tests for port interfaces, where the goal is to test orchestration logic in isolation. See [ADR-0002](adr/0002-test-pyramid-strategy.md).

**No layered/MVC architecture.**
Traditional layers don't prevent upward imports. The pattern relies on discipline, which fails silently in a growing codebase. Ports and adapters with `arch_test.go` enforcement catches violations at build time. See [ADR-0003](adr/0003-ports-and-adapters-architecture.md).

**No HTTP-layer authorization.**
RBAC is enforced in the app layer, not in HTTP middleware or handlers. Authorization is a business rule, not a transport concern. Enforcing it only at the HTTP layer would leave future transports (gRPC, MCP, CLI) unprotected. See [ADR-0008](adr/0008-rbac-in-app-layer.md).

**No `Payload any` for domain events.**
Domain events are a Go interface, not a struct with a generic payload field. Type assertions on `any` are unchecked at compile time. An interface lets each event carry strongly-typed fields and lets the compiler catch unhandled cases. See [ADR-0001](adr/0001-domain-event-interface-not-struct.md).

**No docker-compose for tests.**
A separate compose file for tests is a second thing that drifts from the application's schema. testcontainers-go makes each test self-contained — it owns its database lifecycle, seeds its own schema, and tears down after. See [ADR-0002](adr/0002-test-pyramid-strategy.md).

---

_This document complements but does not duplicate `docs/adr/` (individual decisions). If you want the full reasoning behind any principle listed here, follow the ADR link._
