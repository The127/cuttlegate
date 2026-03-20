# ADR 0006: Cuttlegate is a pure resource server — Keyline owns all auth

**Date:** 2026-03-20
**Status:** Accepted
**Supersedes:** auth approach from #21, #22, #23

## Context

During Sprint 1 we implemented three auth flows inside Cuttlegate:

- **#21** — OIDC authorization code + PKCE handler (login redirect, callback, HMAC session cookie)
- **#22** — Device Authorization Grant (RFC 8628) for SDK/CLI auth
- **#23** — Client Credentials Grant for M2M service accounts, including a credential store backed by Argon2id

We also rejected API keys in ADR 0005 in favour of these flows.

At the sprint boundary, the team established two facts that change the architecture:

1. The operator runs **Keyline** (`github.com/the127/keyline`) as the OIDC provider — a fully configurable, self-hosted IdP that supports device flow, client credentials, and custom JavaScript-based claims mapping.
2. The **Cuttlegate UI will be a JavaScript SPA** acting as an OIDC client — it handles the login flow directly with Keyline and sends Bearer tokens to the Cuttlegate API.

Given these two facts, all auth flows implemented in Cuttlegate are wrong-layer code. Cuttlegate should be a **resource server only**.

## Decision

Cuttlegate is a pure OAuth 2.0 resource server.

- It validates Bearer tokens on every protected request using Keyline's JWKS (via OIDC discovery).
- It does not redirect browsers to login pages.
- It does not handle OIDC callbacks or exchange auth codes.
- It does not manage session cookies.
- It does not store credentials of any kind.
- It does not issue tokens.

All auth flows — browser login (authorization code + PKCE), SDK/CLI login (device flow), and M2M (client credentials) — are handled by Keyline. The Cuttlegate API receives the resulting Bearer token and validates it.

Cuttlegate adds its own RBAC layer on top: a `role` claim embedded in Keyline tokens (configured via Keyline's custom JS claims mapping) determines the caller's Cuttlegate role.

## Rationale

**Single responsibility.** A resource server that also acts as an authorization server is doing two jobs with one codebase. Every auth flow we implement in Cuttlegate is a surface we own, test, and maintain forever. Keyline already implements all these flows correctly.

**Keyline is configurable enough.** Custom claims mapping means Cuttlegate roles can live in Keyline's token without any Cuttlegate-side credential store. Token lifetime is configurable. Audience claims are configurable. There is no gap that requires Cuttlegate to supplement.

**The UI model makes session cookies unnecessary.** A SPA that manages its own OIDC client session means Cuttlegate never needs to touch a cookie. All requests are Bearer-authenticated.

**Less code, smaller attack surface.** The code deleted by this decision includes: PKCE handling, state cookie management, code exchange, HMAC signing, device code store, Argon2id credential hashing, service account management. All of that attack surface goes away.

## Consequences

**Deleted from Cuttlegate:**
- `OIDCHandler` (login redirect, PKCE, callback, code exchange)
- `HMACSessionSigner` and `SessionSigner` port
- `DeviceHandler` (RFC 8628 device flow)
- `ClientCredsHandler` and `ServiceAccountService`
- `ServiceAccount` domain entity and `ServiceAccountRepository` port
- `SecretHasher` port and `Argon2SecretHasher`
- `RequireAuth` cookie middleware
- `/auth/*` routes entirely

**Added:**
- `OIDCVerifier` adapter — wraps `go-oidc` provider discovery and JWT verification; handles JWKS caching automatically
- `TokenVerifier` port — `Verify(ctx, token) (domain.User, error)`; the single auth abstraction Cuttlegate needs
- `RequireBearer` middleware — reads `Authorization: Bearer`, calls `TokenVerifier.Verify`, injects `domain.User` and `domain.AuthContext`

**Config simplified to:**
- `OIDC_ISSUER` — for provider discovery (required)
- `OIDC_AUDIENCE` — expected `aud` claim; empty skips the check (optional)
- `OIDC_ROLE_CLAIM` — claim name carrying the Cuttlegate role (default: `"role"`)

**Keyline must be configured to:**
- Embed a `role` claim (or configured name) in access tokens with a value matching one of `admin`, `editor`, `viewer`
- Include Cuttlegate's audience identifier in the `aud` claim if `OIDC_AUDIENCE` is set
- Set appropriate token lifetime per client type (browser, device, M2M)

**ADR 0005 update:** API keys are still rejected. The reasoning holds — client credentials via Keyline covers all M2M use cases. The implementation of client credentials is now Keyline's responsibility, not Cuttlegate's.
