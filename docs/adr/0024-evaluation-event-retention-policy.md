# ADR 0024: Evaluation event retention policy — 30-day default with configurable cleanup

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #202, #233

## Context

The `evaluation_events` table accumulates one row per flag evaluation. At any non-trivial traffic volume this table grows unbounded. A retention policy is required to prevent unbounded storage growth in production.

Two cleanup mechanisms were considered:

1. **pg_cron** — a PostgreSQL extension that schedules SQL jobs inside the database engine. Clean, no application code required. Dependency: pg_cron must be installed and enabled in the Postgres instance.
2. **Application-level cleanup goroutine** — a background goroutine in the server process that runs a DELETE on a configurable interval. No database extension required; retention config lives in server config alongside other settings.

The retention period must be configurable from day one. Hardcoding 30 days would make it impossible to tune without a code change.

## Decision

**Application-level cleanup goroutine** with a configurable retention period (default: 30 days).

The goroutine runs at server startup on a configurable interval (default: 24 hours) and deletes `evaluation_events` rows older than the retention threshold. The retention period and cleanup interval are both exposed as server config keys.

## Rationale

pg_cron requires an extension that may not be available in all Postgres hosting environments (managed Postgres services vary in extension support). The application-level goroutine has no external dependencies and fits naturally into the existing server lifecycle (startup/shutdown). The cleanup logic is simple enough that the added complexity of a background goroutine is justified by the elimination of the pg_cron dependency.

## Consequences

- `evaluation_events` rows older than the retention period are deleted automatically. Operators who need longer retention must set the config key before data is purged — there is no recovery after deletion.
- The retention config key must be documented in deployment docs and the getting-started guide.
- The cleanup goroutine runs in the same process as the server. A server restart interrupts the cleanup cycle but does not cause data loss — the next cycle catches up.
- Adding new event tables with their own retention requirements will need their own cleanup path; this goroutine is not a general-purpose retention framework.
- Integration test coverage: insert rows older than threshold, run cleanup, assert deletion; insert recent rows, assert retention.
