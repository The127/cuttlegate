# ADR 0017: OIDC missing role claim — reject by default, viewer opt-in

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #185
**Related:** ADR 0006 (Cuttlegate as pure resource server)

## Context

ADR 0006 established that Cuttlegate validates Bearer tokens via OIDC and extracts a role claim to determine the caller's permission level. When an operator misconfigures Keyline's custom claims mapping, tokens may arrive without the role claim.

Prior to this decision, `OIDCVerifier` silently defaulted any token with a missing role claim to `viewer`. Sprint 9 added a log warning for this case (#167), but the behaviour remained permissive. Security review raised that silent defaulting is a privilege decision made without visibility — a token with no role claim may represent a misconfiguration, and granting access (even as viewer) is the wrong default.

## Decision

**Reject tokens with a missing role claim by default.**

An `OIDC_MISSING_ROLE_POLICY` environment variable provides an explicit operator opt-in to the permissive fallback:

| Value | Behaviour |
|---|---|
| `reject` (default, also when unset) | 401 with `{"error":"missing_role_claim","message":"..."}` — token subject logged server-side only |
| `viewer` | viewer role granted — WARN log includes token subject and policy value |
| any other value (including `""`) | startup error naming the bad value and listing valid options |

The token subject is deliberately excluded from the 401 response body to avoid information disclosure. It appears only in the structured log.

## Rationale

**Missing claim = misconfiguration, not a known state.** If the IdP is configured correctly, every token has a role claim. Absence means the configuration is wrong. The correct response to a misconfiguration is a hard error, not a permissive guess.

**Operator intent must be explicit.** An operator who wants permissive fallback must set `OIDC_MISSING_ROLE_POLICY=viewer` deliberately. This makes the decision auditable: if something breaks, the configuration explains why.

**Fail-fast on bad config.** An unrecognised policy value (including empty string set explicitly) causes a startup error, not a silent default. The operator finds out immediately, not when the first user with a missing claim hits production.

**Subject in logs, not in response.** Returning the token subject in a 401 response body would disclose whether an account exists. The subject goes to structured logs only, where it is useful for operator debugging without being visible to the caller.

## Consequences

- `OIDCVerifier` now takes a `MissingRolePolicy` constructor parameter; the policy is resolved once at startup and injected.
- `Config.OIDCMissingRolePolicy` is populated by `Load()` from `OIDC_MISSING_ROLE_POLICY`; invalid values fail startup.
- `RequireBearer` and `RequireBearerOrAPIKey` middleware detect `errMissingRoleClaim` and write `{"error":"missing_role_claim"}` rather than the generic `{"error":"unauthorized"}`.
- Operators running with an IdP that does not embed role claims in every token must set `OIDC_MISSING_ROLE_POLICY=viewer` to restore previous behaviour.
- `docker-compose.yml` and README document the env var.
- ADR 0006 is unchanged: the resource-server model is unaffected; this decision narrows the default behaviour of role extraction within that model.
