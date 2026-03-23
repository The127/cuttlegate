# Pre-Release Security Checklist — v1.0

_Completed: Sprint 24._

This document captures the results of the structured security review required before tagging v1.0 and publishing Docker images. It is not a penetration test — it is a checklist pass against the known trust boundaries of the system.

---

## Authentication and Authorisation

### OIDC token validation: expiry, audience, issuer checks

**Status: PASS (with deployment requirement)**

- Expiry and issuer validation are handled by the `go-oidc` library via `provider.Verifier()`. All tokens are verified against the OIDC provider's JWKS endpoint with automatic key caching.
- Audience enforcement: if `OIDC_AUDIENCE` env var is non-empty, `cfg.ClientID` is set and the `aud` claim is enforced. If `OIDC_AUDIENCE` is empty, `cfg.SkipClientIDCheck = true` — audience is not validated.
- **Deployment requirement:** The default `docker-compose.yml` does not set `OIDC_AUDIENCE` (it sets `OIDC_CLIENT_ID` which is a different variable used only to serve the SPA config). For production deployments using an OIDC provider that sets `aud` claims, operators **must** set `OIDC_AUDIENCE` to the expected audience value.
- Code: `internal/adapters/http/oidcverifier.go` `NewOIDCVerifier`, `cmd/server/config.go` L100-101.

### API key hash storage: SHA-256, plaintext never persists

**Status: PASS**

- `domain.GenerateAPIKey` hashes the plaintext with `sha256.Sum256` and stores only the hash in the returned `APIKey` struct. The plaintext is returned once to the caller and not stored.
- `PostgresAPIKeyRepository.Create` stores `key.KeyHash[:]` (the hash bytes) in the `key_hash` column. The plaintext is never written to the database.
- Code: `internal/domain/api_key.go`, `internal/adapters/db/postgres_api_key_repository.go`.

### RBAC enforced in app layer for all project-scoped endpoints

**Status: PASS**

Every write method in the app layer opens with a `requireRole` call at the method body. Verified services:

- `FlagService`: all create/update/delete/enable/disable require `RoleEditor` or higher.
- `SegmentService`, `RuleService`, `EnvironmentService`: writes require `RoleEditor`.
- `ProjectService`: create requires `RoleEditor`; update/delete require `RoleAdmin`.
- `ProjectMemberService`: mutations require `RoleAdmin`; reads require `RoleViewer`.
- `APIKeyService`: create/revoke require `RoleAdmin`; list requires `RoleViewer`.
- `PromotionService`: requires `RoleAdmin`.
- `EvaluationService`, `EvaluationAuditService`, `EvaluationStatsService`, `AuditService`: require `RoleViewer` minimum.
- No method was found without a `requireRole` call unless it is genuinely public-by-design.

RBAC enforcement is at the app service layer, not only at the HTTP adapter layer.

### Capability tier enforcement: MCP read/write/destructive boundaries

**Status: PASS**

- `toolTier()` in `internal/adapters/mcp/tools.go` maps each tool name to its required tier at compile time.
- In `handleToolsCall`, the required tier is checked against `key.CapabilityTier.Permits(requiredTier)` using the **live key** fetched from the database on every call (`liveKeyCheck`). Session tier is updated from live key — downgrades are handled automatically.
- A `TierRead` key attempting `enable_flag` or `disable_flag` receives `{"error":"insufficient_capability","required":"write","provided":"read"}`. The tool is not executed.
- The tier is sourced from the database, not from client-supplied session state.
- Code: `internal/adapters/mcp/server.go` `handleToolsCall`, `liveKeyCheck`.

---

## API Surface

### CORS: evaluation endpoint CORS posture

**Status: PASS (intentional; documentation corrected)**

No `Access-Control-*` headers are set anywhere in the codebase. No CORS middleware exists.

**Determination:** Intentional and correct. All current API consumers are same-origin SPA or server-side SDK clients. There is no browser SDK. CORS middleware is not needed.

**Action taken:** `internal/adapters/http/doc.go` corrected — the false CORS claim removed. The doc now states CORS is absent by design and documents when it would need to be added.

### Rate limiting on evaluation endpoint

**Status: PASS**

- `evalRateLimiter` wraps `EvaluationHandler` in `evalAuth` (cmd/server/main.go lines 156-159).
- Rate limiter is keyed per `UserID` from `AuthContext` — not per IP. Correct design for authenticated endpoints.
- Unauthenticated requests rejected with 403; rate-limited requests receive 429 with `Retry-After` header.
- Default: 600 requests per minute per user. Configurable via env vars.

### No endpoints leak internal error details

**Status: PASS**

- `WriteError` maps all unrecognised errors to `500` with `{"error":"internal_error","message":"an unexpected error occurred"}`. No stack traces, SQL errors, or internal paths in responses.
- `writeUnauthorized` returns `{"error":"unauthorized","message":"authentication required"}`.
- `writeMissingRoleClaim` deliberately excludes the token subject from the response body.
- MCP responses use opaque error codes only.

---

## Deployment

### Docker Compose default config does not ship with insecure production secrets

**Status: PASS (with comments added)**

`docker-compose.yml` contains dev default credentials (Postgres, Keyline DB). These are local dev defaults only. **Action taken:** `# dev-only defaults — do not use in production` comments added to credential blocks. `OIDC_AUDIENCE` deployment requirement commented inline.

### `deploy/keyline/config.yml` is clearly scoped to local development

**Status: PASS**

File header clearly identifies this as dev config. `passwordHash` is documented inline as the argon2id hash of `"password"`. Operator-safe for local dev; would not be confused for production config.

### AUTO_MIGRATE production warning

**Status: PASS**

`cmd/server/main.go` line 44: warning logged before any migration runs.

### No hardcoded credentials or secrets in the codebase

**Status: PASS**

- `git log --all --oneline -S 'password' -- '*.go'` returned no results.
- Go source files contain only legitimate code references to credential-related terms (config field names, token verification logic).
- `deploy/keyline/config.yml` is the only file with credential-like values; all explicitly scoped to local development.

---

## Dependencies

### govulncheck — Go module vulnerability check

**Status: FOLLOW-UP ISSUE CREATED (#315)**

`govulncheck ./...` found two standard library vulnerabilities in go1.25.7:

| ID | Package | Description | Fixed in |
|---|---|---|---|
| GO-2026-4602 | `os` | `FileInfo` escape from `Root` | go1.25.8 |
| GO-2026-4601 | `net/url` | Incorrect IPv6 host literal parsing | go1.25.8 |

Both have confirmed call paths through production code. Follow-up issue #315 created to bump Go toolchain to 1.25.8.

Zero third-party dependency vulnerabilities found.

### npm audit — web dependency check

**Status: PASS**

`npm audit` in `web/` returned: `found 0 vulnerabilities`.

---

## Summary

| Item | Status | Notes |
|---|---|---|
| OIDC expiry/issuer validation | PASS | go-oidc handles both |
| OIDC audience check | PASS | Requires `OIDC_AUDIENCE` set in production |
| API key SHA-256 hashing | PASS | Plaintext never stored |
| RBAC in app layer | PASS | All write methods verified |
| MCP capability tier enforcement | PASS | Server-side, per-call, live key check |
| CORS posture | PASS | Intentionally absent; doc.go corrected |
| Evaluation rate limiting | PASS | Per-user, 600/min default, Retry-After header |
| No internal error detail leakage | PASS | Generic 500 message throughout |
| Docker Compose insecure defaults | PASS | Dev credentials; comments added |
| Keyline config dev scope | PASS | Header clearly states dev-only |
| AUTO_MIGRATE warning | PASS | Logged before any migration runs |
| No hardcoded credentials | PASS | Git log + grep both clean |
| govulncheck | FOLLOW-UP | go1.25.7 to go1.25.8 required (issue #315) |
| npm audit | PASS | Zero vulnerabilities |

**Security sign-off:** All checklist items verified against the actual codebase. Two items required action: `doc.go` corrected (CORS claim removed); Go stdlib CVEs produce follow-up issue #315. One deployment requirement documented: `OIDC_AUDIENCE` must be set explicitly in production. Docker-compose dev credential comments added. All other items pass. This codebase is cleared for v1.0 tagging subject to the Go version bump in #315.
