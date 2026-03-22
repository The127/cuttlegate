# ADR 0020: UUID vs TEXT primary key inconsistency in rules and segments

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #222

## Context

Cuttlegate's schema uses TEXT primary keys for most entities: `projects`, `environments`, `flags`, `api_keys`, `audit_events`, `evaluation_events`. The application generates these IDs as UUID strings in the Go layer via `newUUID()` and stores them as plain TEXT.

When `rules` (migration 0007) and `segments` (migration 0009) were introduced, they were created with `UUID PRIMARY KEY DEFAULT gen_random_uuid()` — native Postgres UUID columns with server-side ID generation. This diverges from the established pattern in two ways:

1. **Column type:** UUID vs TEXT
2. **ID generation:** Postgres-side (`gen_random_uuid()`) vs Go application layer (`newUUID()`)

ADR 0014 chose typed entity IDs as a long-term direction. The inconsistency creates friction if that pattern is applied uniformly across all entities — the Go type wrapping layer must handle both TEXT and UUID columns identically, which it currently does, but the difference is a latent source of confusion for contributors.

Two paths forward:

1. **Migrate rules and segments to TEXT PKs** — align with the rest of the schema; move ID generation to Go. Requires a data migration and adapter changes.
2. **Accept the inconsistency** — document it and do not migrate. The functional impact is zero.

## Decision

**Accept the inconsistency.** Do not migrate rules or segments to TEXT PKs at this time.

Both UUID columns and TEXT columns store UUID strings at the wire level. The Postgres adapter reads and writes them identically (`pgx` handles the conversion transparently). A migration would introduce risk for no user-visible benefit.

If a future migration already touches either table (e.g. adding `name` to rules — see #220), evaluate at that point whether to fold in the PK type change.

## Rationale

No concrete harm exists today. The inconsistency is aesthetic and a minor maintainability concern, not a correctness or performance problem. Migrating now is cost without benefit. The right time to fix it is when the table is already being touched for another reason.

## Consequences

- `rules.id` and `segments.id` remain `UUID` columns with Postgres-side generation.
- All other entity IDs remain `TEXT` with Go application-layer generation.
- New tables should use `TEXT PRIMARY KEY` with application-generated UUIDs to match the majority pattern.
- When any migration touches `rules` or `segments`, the author should evaluate converting the PK type and update this ADR.
- ADR 0014 (typed entity IDs) applies equally to both column types — the Go wrapper type is independent of the Postgres storage type.
