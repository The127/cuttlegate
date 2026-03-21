# ADR 0013: Transactional unit of work for multi-repository operations

**Date:** 2026-03-21
**Status:** Accepted
**Issue:** #146

## Context

Cuttlegate's app-layer services currently receive repository ports as independent constructor arguments. Each Postgres adapter wraps a `*sql.DB` and executes queries against the shared connection pool — but there is no mechanism for two repositories to share a single database transaction.

This is fine today: most use cases write to a single repository. But upcoming work will require atomic multi-repository writes:

- **Audit trail** (#151): creating or updating a flag must atomically write an audit event — `FlagRepository.Update` + `AuditRepository.Record` in one transaction.
- **RBAC permission changes**: revoking a member's role while simultaneously updating dependent state.
- **Environment lifecycle**: deleting an environment must cascade to flag-environment state and rules atomically.

Without a transaction-sharing mechanism, these operations either silently lose atomicity (each repo commits independently) or push transaction management into the adapter layer where the app layer cannot control it.

Three patterns were evaluated:

1. **Context-scoped transaction** — carry a `*sql.Tx` in `context.Context`; repositories detect it and use it instead of `*sql.DB`.
2. **Unit of work port** — define a `UnitOfWork` interface in `domain/ports`; the app layer explicitly begins a transactional scope and obtains transaction-scoped repositories from it.
3. **Application-layer compensation** — no shared transaction; if a second write fails, the app layer explicitly reverses the first write.

## Decision

Adopt **pattern 2: unit of work port**.

Define a `UnitOfWork` interface in `domain/ports`:

```go
// UnitOfWork represents a transactional scope that provides
// repository access within a single atomic operation.
type UnitOfWork interface {
    FlagRepository() FlagRepository
    AuditRepository() AuditRepository
    // Add repository accessors as needed.
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}

// UnitOfWorkFactory begins a new transactional scope.
type UnitOfWorkFactory interface {
    Begin(ctx context.Context) (UnitOfWork, error)
}
```

The Postgres adapter implements `UnitOfWorkFactory` by calling `db.BeginTx`, then constructing repository instances that share the resulting `*sql.Tx`. App-layer services receive a `UnitOfWorkFactory` and call `Begin` when they need atomicity. Non-transactional use cases continue to use directly-injected repositories — no change required.

## Rationale

### Why not context-scoped transaction (pattern 1)?

Context-scoped transactions are idiomatic Go and widely used. The `*sql.Tx` is placed in the context (typically via a typed key), and repositories extract it with a helper like `txFromContext(ctx)`, falling back to `*sql.DB` if absent.

**Advantages:**
- Zero new interfaces — repositories keep the same signatures.
- Familiar pattern in the Go ecosystem.
- A typed context key (not raw `context.Value`) keeps adapter code clean.

**Why rejected:**
- **Implicit from the app layer.** The app service has no typed signal that it is inside a transaction. Whether writes are atomic depends on whether the caller remembered to wrap the context — a silent, invisible contract. Forgetting it produces no compile error and no runtime error; writes simply go non-atomic.
- **Port interfaces lie.** The repository port signatures accept `context.Context` but their transactional behaviour depends on hidden context values. A developer reading the port interface gets no indication that atomicity is possible or expected.
- **Testing is harder.** Faking transactional behaviour in test doubles requires the test to also set up the context convention. The unit of work pattern makes the transactional boundary an explicit argument, testable by construction.

### Why not application-layer compensation (pattern 3)?

Compensation (saga pattern) is designed for distributed systems where a shared transaction is impossible — separate databases, separate services, network boundaries.

**Why rejected:**
- Cuttlegate has one database. A single Postgres transaction is simpler, faster, and more reliable than compensating writes.
- Compensation logic in the app layer is complex, error-prone, and must handle partial failures during the compensation itself.
- The pattern is over-engineered for the problem we have.

### Why unit of work (pattern 2)?

- **Explicit in the app layer.** The service calls `uow.Begin(ctx)`, gets transaction-scoped repositories, and calls `uow.Commit(ctx)` or `uow.Rollback(ctx)`. The transactional boundary is visible in the code, not hidden in context values.
- **Port interfaces are honest.** `UnitOfWork` is a named interface in `domain/ports`. A developer reading the ports knows that transactional scopes exist and how to use them.
- **Testable by construction.** A fake `UnitOfWork` in tests records which repositories were accessed and whether `Commit` or `Rollback` was called. No context setup required.
- **Opt-in.** Services that don't need transactions keep their existing repository injection. The unit of work is additive — it does not force a rewrite of every service constructor.
- **Hexagonal-clean.** The interface lives in `domain/ports`, the implementation in `adapters/db`. The app layer depends only on the interface. No infrastructure types leak inward.

**Tradeoff acknowledged:** the unit of work adds a new interface, a factory, and a Postgres implementation. This is more ceremony than the context approach. We accept this cost because the explicitness prevents a class of silent atomicity bugs that would be difficult to diagnose in production.

## Consequences

- A `UnitOfWork` interface and `UnitOfWorkFactory` interface will be added to `domain/ports`.
- A `PostgresUnitOfWork` adapter will be added to `adapters/db`, wrapping `*sql.Tx` and constructing transaction-scoped repository instances.
- App services that need atomicity (starting with audit trail, #151) will receive a `UnitOfWorkFactory` and call `Begin`/`Commit`/`Rollback` explicitly.
- App services that do not need atomicity are unchanged — no forced migration.
- `arch_test.go` must be updated if the unit of work introduces new packages (unlikely — it lives in existing packages).
- The `UnitOfWork` interface will grow accessors as more repositories need transactional participation. Each new accessor is a small, additive change.
- `cmd/server/` wiring grows by one constructor call to build the `PostgresUnitOfWork` factory and inject it into services that need it.
