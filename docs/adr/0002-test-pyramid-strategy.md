# ADR 0002: Test pyramid strategy

**Date:** 2026-03-20
**Status:** Accepted
**Context:** Sprint 1 foundation session

## Context

With real adapter code now in place, the team needed to agree on a test strategy before integration tests are written. The key decision was how to handle tests that require a live PostgreSQL instance — and whether to use docker-compose or a programmatic approach.

## Decision

Three-tier test pyramid:

| Tier | Scope | Command | DB required |
|---|---|---|---|
| Unit | Domain logic, HTTP handlers, pure functions | `go test ./...` | No |
| Integration | DB adapter implementations against real Postgres | `go test -tags=integration ./...` | Yes |
| E2E | Full HTTP stack | Deferred to Sprint 2 | Yes |

Integration tests are gated behind the `//go:build integration` build tag so that `go test ./...` remains fast and dependency-free.

**For integration tests: testcontainers-go over docker-compose.**

## Rationale

**Why testcontainers-go over docker-compose:**

A separate `docker-compose.yml` for tests is a second thing to keep in sync with the application's schema and configuration. When the compose file drifts from what tests expect, failures are confusing and hard to attribute. testcontainers-go spins up a Postgres container programmatically inside the test itself — the test owns its database lifecycle, seeds its own schema, and tears down after. This makes tests self-contained and eliminates the compose-drift failure mode.

The cost is ~3s container startup per integration test package. For a foundation-stage project with few integration tests, this is acceptable.

**Why build tags over a separate directory:**

Integration tests live alongside the code they test (e.g. `internal/adapters/db/flag_repository_integration_test.go`). The build tag keeps them co-located with the unit tests they complement, which makes it easier to see coverage gaps. A separate `test/integration/` directory would require mirroring the package structure.

## Makefile targets

```
make test                 # go test ./...
make test-integration     # go test -tags=integration ./...
```

## Consequences

- All integration tests must carry `//go:build integration` at the top of the file
- Each integration test is responsible for starting and stopping its own container
- `go test ./...` in CI runs only unit tests — a separate CI step is needed to run integration tests with Docker socket access
- **CI Docker access (resolved):** GitHub Actions `ubuntu-latest` runners ship with Docker installed and the daemon running. testcontainers-go connects via the Docker socket automatically — no DinD, no socket mounts, no paid service required. Integration tests run identically in CI and local dev. The only requirement: `make test-integration` must run on a runner with Docker available (document this for self-hosted runner users).
- E2E tests deferred until Sprint 2 when there is domain behaviour to test end-to-end
