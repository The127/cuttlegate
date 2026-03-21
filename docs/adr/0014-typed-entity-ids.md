# ADR 0014: Typed entity IDs to replace bare string IDs

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #174

## Context

Every entity ID in the domain layer is a bare `string`. There are at least six distinct ID concepts — ProjectID, FlagID, EnvironmentID, RuleID, SegmentID, and UserID (OIDC `Sub`) — all sharing the same Go type. This is a textbook case of Primitive Obsession.

The practical risk: port interface signatures like `ListByFlagEnvironment(ctx, flagID, environmentID string)` and `UpdateRole(ctx, projectID, userID string, role)` accept any string in any position. Swapping two ID arguments compiles and passes type checks but produces silent data corruption or not-found errors at runtime. As the codebase grows, this class of bug becomes increasingly likely and harder to diagnose.

Three options were evaluated:

**Option A — Accept: migrate all entities at once.** Define typed IDs for all entities and update every port, service, adapter, and handler in a single pass. This guarantees consistency but produces a large, hard-to-review changeset touching 40+ files.

**Option B — Accept: migrate incrementally, one entity at a time.** Define typed IDs and migrate entity-by-entity, starting with the most cross-cutting ID. Each migration is a self-contained, reviewable unit. The codebase is temporarily inconsistent (some IDs typed, some bare) but the inconsistency is mechanical and resolvable by inspection.

**Option C — Defer with conditions.** Keep bare strings and revisit when a bug attributable to ID misuse occurs or when the codebase crosses a complexity threshold. This avoids churn now but leaves the compiler unable to help.

## Decision

**Accept Option B — introduce typed entity IDs and migrate incrementally.**

Define each typed ID as a named string type in `internal/domain/`:

```go
type ProjectID string
type FlagID string
type EnvironmentID string
type RuleID string
type SegmentID string
```

UserID (OIDC `Sub`) follows the same pattern but is already semantically distinct — it comes from the identity provider, not from our ID generation. It should be typed as `type UserID string` for the same compile-time safety.

Migration order: start with `ProjectID` since it appears in the most entities and port signatures (Project, Flag, Environment, ProjectMember, Segment, and their repositories). This establishes the pattern and flushes out any ergonomic issues before applying it to the remaining types.

Each entity migration is one issue, one branch, one commit.

**Before/after at the port boundary:**

```go
// Before — bare strings, any ID accepted in any position
type FlagRepository interface {
    GetByKey(ctx context.Context, projectID, key string) (*domain.Flag, error)
    Delete(ctx context.Context, id string) error
}

// After — compiler rejects mismatched ID types
type FlagRepository interface {
    GetByKey(ctx context.Context, projectID domain.ProjectID, key string) (*domain.Flag, error)
    Delete(ctx context.Context, id domain.FlagID) error
}
```

## Rationale

**Why typed IDs in Go work well:**

- `type ProjectID string` has zero runtime overhead — it is the same size and layout as `string`.
- JSON marshalling is transparent: `json.Marshal(ProjectID("abc"))` produces `"abc"`. No custom marshaller needed. Locked SDK contracts (eval endpoint response, SSE events) are unaffected.
- SQL scanning requires a trivial `sql.Scanner` implementation (or use `(*string)(&id)` in `Scan` calls). The `database/sql` package handles this naturally.
- The pattern is idiomatic Go — the standard library uses it extensively (`http.Method`, `os.FileMode`).

**Why incremental over all-at-once:**

- Each migration touches one entity's struct, its port interface, its adapters, its service, and its handlers — a bounded, reviewable unit.
- Temporary inconsistency (some IDs typed, some bare) is visible and mechanical. A developer can see which entities have been migrated by checking the type of the ID field.
- If an unexpected ergonomic issue surfaces (e.g., a third-party library that doesn't handle named types well), it is caught early with minimal sunk cost.

**Why not defer:**

- The codebase already has 40+ files importing domain types. The blast radius grows with every new entity or port method. Deferring makes the eventual migration larger, not smaller.
- The risk is not hypothetical: `string`-typed ID parameters in port methods are a latent bug surface that the compiler cannot guard against today.

## Consequences

- **Every port interface signature changes** as each entity is migrated. Adapters and services must update in lockstep. This is mechanical but unavoidable.
- **Constructor and factory functions** that accept IDs must accept the typed version. Callers at the adapter boundary (HTTP handlers, SQL scan targets) perform the conversion.
- **Test code** that constructs domain objects with string literals must use the typed ID: `domain.ProjectID("test-id")`. This is slightly more verbose but makes test intent clearer.
- **Temporary inconsistency** during the migration window: some IDs will be typed, others bare. This is acceptable and self-documenting — the struct field type is the source of truth.
- **No wire format changes.** JSON and SQL representations are identical. No migration scripts, no API version bump, no SDK changes.
