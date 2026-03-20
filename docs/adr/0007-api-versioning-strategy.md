# ADR 0007: API versioning strategy

**Date:** 2026-03-20
**Status:** Accepted
**Issue:** #105

## Context

Cuttlegate exposes a JSON HTTP API consumed by the SPA, SDKs (Go, TypeScript, Python), and AI agents via MCP. SDKs pin to an API version at build time — without a versioning strategy, the first breaking change becomes a production crisis for every SDK consumer simultaneously.

Three options were on the table:

1. **URL prefix** — `/api/v1/flags/...` — version is explicit in every request
2. **`Accept` header** — `Accept: application/vnd.cuttlegate.v1+json` — cleaner URLs; version is in the header
3. **No versioning** — commit to never breaking the API

## Decision

All API routes are prefixed with `/api/v1/`. This is already in place — every handler's `RegisterRoutes` method registers under `/api/v1/`.

A future incompatible API would live under `/api/v2/`. `/api/v1/` routes remain available for a documented deprecation window before removal.

Internal/admin routes (e.g. `/api/v1/config`) follow the same prefix convention.

## Rationale

- **URL prefix beats `Accept` header** for this use case: SDK authors can test routes in a browser or with `curl` without custom headers; HTTP caches key on URL by default; routing middleware is trivial.
- **No versioning is not viable**: SDKs pin to a specific API shape at release time. Without versioning, any field rename or removal is a silent breaking change with no migration path.
- **`/api/v1/` not `/v1/`**: the `/api/` prefix reserves the URL space cleanly for the SPA — anything under `/api/` is JSON, everything else serves the SPA shell.

## Breaking change policy

| Change type | Version impact | Examples |
|---|---|---|
| Add optional field to response | None — backwards compatible | Adding `description` to flag response |
| Add optional request field | None — backwards compatible | New optional filter on list endpoint |
| Remove or rename a field | **Major** — requires `/api/v2/` | Renaming `enabled` to `active` |
| Change field type | **Major** | Changing `id` from UUID string to integer |
| Remove an endpoint | **Major** | Removing `DELETE /api/v1/flags/{key}` |
| Add a required request field | **Major** | Making `environment` required on evaluate |
| Add a new endpoint | None — backwards compatible | New `POST /api/v1/.../evaluate` |
| Change error codes | **Major** | Renaming `conflict` to `duplicate` |

When a major change is required: introduce the new shape under `/api/v2/`, announce the deprecation window, and remove `/api/v1/` after the window expires. The window must be at least one SDK release cycle.

## Consequences

- All new endpoints must register under `/api/v1/` — no exceptions without an ADR
- The SPA, SDKs, and MCP server all consume `/api/v1/` — a breaking change requires coordinated updates across all three consumers
- The API contract doc (`docs/api-contract.md`) references this ADR as the versioning authority
