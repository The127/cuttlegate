# Changelog

All notable changes to Cuttlegate are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## Upgrading

### From pre-release builds to v1.0.0

v1.0.0 is the initial public release. There is no prior stable version to
upgrade from. If you were running a pre-release build from main, apply any
pending database migrations with `just migrate-up` (or the Docker Compose
`migrate` service) before starting the new server binary.

---

## [1.0.0] - 2026-03-23

### Added

#### Feature Flags

- Feature flag CRUD with four flag types: `bool`, `string`, `number`, and `json`
- Per-environment flag state toggle — flags are independently enabled or
  disabled in each environment within a project
- Multivariate flag variants — each flag carries one or more named variants
  with typed values
- Flag promotion — copy flag state and variants across environments via the
  management UI or API

#### Targeting Rules

- Targeting rules with attribute-based and segment-based conditions — rules
  evaluate conditions against evaluation context attributes and resolve to a
  variant when matched
- Segment CRUD — create segments, manage member lists, and reference them in
  targeting rules via `in_segment` / `not_in_segment` conditions
- Rule evaluation engine with condition operators: `eq`, `neq`, `contains`,
  `starts_with`, `ends_with`, `in`, `not_in`, `in_segment`, `not_in_segment`
- Rule name field carried through evaluation results and audit events

#### Authentication

- OIDC authentication with PKCE flow — the server validates JWTs from any
  OIDC-compliant provider; the bundled Docker Compose configuration uses
  [Keyline](https://github.com/the127/keyline) as the default provider
- Role-based access control enforced at the application layer — three roles:
  `admin`, `editor`, `viewer`; role is read from a configurable JWT claim
- API key authentication for SDK clients — keys are scoped to a project and
  carry a capability tier (`read`, `write`, or `destructive`)
- Project membership model — users are granted access to specific projects

#### Client SDKs

- **Go SDK** (`sdk/go`) — `CuttlegateClient` with `Bool`, `String`, `Evaluate`,
  and `EvaluateAll` methods; `CachedClient` with in-memory flag cache backed by
  a persistent SSE connection and automatic reconnect; `MockCuttlegateClient`
  for in-process testing; `Subscribe` for streaming flag state updates
- **JavaScript/TypeScript SDK** (`sdk/js`) — `CuttlegateClient` with typed
  evaluation methods; `CachedClient` with SSE-backed cache; `useCachedFlag`
  React hook for subscribe-based reactivity; full TypeScript types
- **Python SDK** (`sdk/python`) — `CuttlegateClient` with `bool_flag`,
  `string_flag`, `evaluate`, and `evaluate_all` methods; `CachedClient` with
  SSE-backed cache and daemon thread; `MockCuttlegateClient` for in-process
  testing; supports Python 3.11+

#### MCP Server

- MCP server (`internal/adapters/mcp`) with HTTP+SSE transport — exposes
  Cuttlegate management operations as MCP tools for AI agent integration
- Three capability tiers enforced per API key:
  - `read` — flag evaluation and project/environment inspection
  - `write` — flag and segment creation and modification
  - `destructive` — flag deletion and irreversible operations
- E2E test coverage for capability tier enforcement

#### Observability

- Evaluation audit trail — every flag evaluation is recorded with flag key,
  environment, variant resolved, targeting rule matched (by name), and
  timestamp; queryable via cursor-paginated API and management UI
- Evaluation analytics — time-bucketed evaluation stats endpoint; bar chart
  panel in the management UI
- Evaluation event retention — configurable via environment variables
- Audit log for management operations — flag create/update/delete,
  segment create/delete, environment changes; filterable and paginated

#### Management SPA

- Single-page application (`web/`) built with React and Vite
- Dark theme with deep navy palette — design tokens defined in CSS `@theme`
  block under Tailwind v4
- JetBrains Mono for monospace content (flag keys, variant values, code
  snippets); self-hosted via `@font-face`, no external font CDN dependency
- App shell with dark sidebar, gradient active state, and top bar
- Flag list, flag detail, and flag edit views with per-environment toggle
- Segment list and segment detail views
- Audit log UI with filter and pagination
- Analytics dashboard panel with SVG bar chart
- Promote-flag UI for cross-environment flag state comparison and promotion
- First-run SDK prompt shown after flag creation
- Component library: `Button`, `StatusBadge`, `DataTable`, `CopyableCode`,
  `Dialog` (Radix UI)
- i18n: English (default), Simplified Chinese (`zh-CN`), and German (`de`)
  translations

#### Deployment

- Docker Compose configuration for self-hosted deployment — services:
  `db` (Postgres 17), `migrate`, `server`, `keyline`, `keyline-ui`
- `/health` endpoint for deployment readiness checks (used by Docker Compose
  health check and load balancers)
- Multi-stage Dockerfile producing a minimal runtime image
- `just` task runner with targets: `build`, `test`, `test-integration`,
  `lint`, `ci`, `migrate-up`, `migrate-down`, `dev` (hot reload)

#### Documentation

- Getting-started guide covering project setup, flag creation, targeting rules,
  and SDK integration
- MCP server getting-started guide
- Go SDK README with API reference
- JavaScript/TypeScript SDK README with API reference
- Python SDK README with API reference
- Architecture Decision Records in `docs/adr/` (ADRs 0001–0030)

### Infrastructure

- Ports and adapters (hexagonal) architecture — `domain`, `app`, `adapters`,
  and `cmd` layers with enforced import direction via `arch_test.go`
- PostgreSQL persistence with `golang-migrate` SQL migrations
- `golangci-lint` configured with import ordering and vet checks
- CI pipeline covering Go lint, unit tests, integration tests (Testcontainers),
  and SDK tests for all three languages
- Git commit-msg hook enforcing Conventional Commits format with issue reference
