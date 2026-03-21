# ADR 0012: API key as second authentication path for the evaluation endpoint

**Date:** 2026-03-21
**Status:** Accepted
**Issue:** #148
**Supersedes:** ADR 0005 (API key authentication — rejected)
**Narrows:** ADR 0006 (Cuttlegate is a pure resource server — Keyline owns all auth)

## Context

ADR 0005 rejected API keys because client credentials via Keyline covered all M2M use cases. ADR 0006 established Cuttlegate as a pure resource server that stores no credentials.

In Sprint 5 the team identified a concrete DX gap: SDK consumers must provision an OIDC client in Keyline, manage client_id/client_secret, call the token endpoint, and handle refresh — just to check a feature flag. Every competing feature flag platform offers a single API key string. This gap was identified as the #1 blocker to external adoption.

The gap ADR 0005 left open — "unless the team encounters a concrete use case that neither device flow nor client credentials can serve" — is now filled. The use case is not technical impossibility but practical adoption: no one integrates a feature flag SDK that requires an OAuth dance.

## Decision

Cuttlegate accepts **API keys as a second authentication path**, scoped to the **evaluation endpoint only**.

- Keys are scoped to a (project, environment) pair
- Key format: `cg_<base64url(32 bytes crypto/rand)>`, ~47 characters
- Storage: SHA-256 hash of the full key + a display prefix (first 8 chars of the random part) for identification
- The plaintext key is returned once at creation and never stored or retrievable
- The `Authorization: Bearer cg_...` prefix distinguishes API keys from OIDC JWTs
- Hash lookup is a database equality check (`WHERE key_hash = $1`), not a Go-side comparison — timing attacks are not viable
- Error responses are identical for invalid, revoked, and wrong-scope keys (no state leakage)

Cuttlegate does **not** become an authorization server:
- No token issuance
- No refresh flow
- No session management
- No OAuth grants

API key management (create, revoke, list) is exposed as OIDC-authenticated management endpoints with RBAC: create/revoke require `admin`, list requires `viewer`.

## Rationale

**SHA-256 over Argon2id.** Keys are 32 bytes of `crypto/rand` — not user-chosen passwords. Brute-forcing 256 bits of entropy is computationally infeasible regardless of hash speed. Argon2id would add per-request latency (evaluation is the hottest path) for zero security benefit.

**Per (project, environment) scoping.** Matches the evaluation endpoint's existing authorization model. A key grants exactly the access its scope describes — no more.

**Dual-auth middleware, not two separate middleware chains.** The evaluation endpoint accepts either credential type through a single middleware that dispatches on the `cg_` prefix. This keeps routing simple and avoids duplicating the endpoint registration.

## Consequences

- ADR 0005 is superseded — API keys are now implemented
- ADR 0006 is narrowed: Cuttlegate stores hashed API keys as access control data, but remains a resource server (no token issuance, no OAuth grants)
- New domain entity `APIKey`, port `APIKeyRepository`, service `APIKeyService`
- New migration: `api_keys` table with cascade FKs to environments (and transitively projects)
- New HTTP middleware: `RequireBearerOrAPIKey` replaces `RequireBearer` on the evaluation endpoint
- New HTTP handler: `APIKeyHandler` for create/list/revoke behind OIDC auth
- The SDK contract gains a simpler auth path: one string instead of OAuth client credentials
