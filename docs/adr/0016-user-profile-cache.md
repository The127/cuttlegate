# ADR 0016: User profile cache — upsert from OIDC token claims

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #197

## Context

The members list API returns `user_id`, `role`, and `created_at`. The members UI (#59) shipped displaying raw user IDs because no name or email data was available at the time. To show human-readable identities, Cuttlegate needs access to each member's name and email.

User identity is owned by Keyline (ADR 0006). Cuttlegate is a pure resource server — it does not query Keyline for user profiles on demand. Instead, it caches the name and email from the verified OIDC token claims that arrive on every authenticated request.

## Decision

Maintain a local `users` table that caches each user's OIDC profile (sub, name, email). The `RequireBearer` middleware upserts the authenticated user's profile on every authenticated request, using the claims from the already-verified token. The upsert is best-effort: if it fails, the request proceeds and the error is logged.

## Rationale

- **No additional round-trips.** The OIDC token is already verified by the middleware. The name and email claims are already in memory. The upsert is a single index-hit SQL statement — no external calls required.
- **Keyline remains the source of truth.** If a user's name or email changes in the IdP, the next authenticated request updates the cache. Staleness is bounded by session duration.
- **Best-effort is appropriate for a display cache.** A failed upsert leaves stale or missing profile data — a cosmetic issue, not a security or correctness failure. Blocking the request for a cache write would be the wrong trade-off.

## Trust boundary

The cached name and email values originate from a verified OIDC token — the signature, expiry, and audience have been checked before the upsert runs. These values are trusted to the same degree as any other claim in the token.

**Critical constraint:** the `users` table is a display cache only. It must never be used for authorization decisions. Roles come from `project_members` (persisted at membership grant time) and the OIDC role claim (verified per-request). A query that reads role from `users` would be a security defect.

## Consequences

- A `users` table exists in the database with `id TEXT PRIMARY KEY` (OIDC sub), `name TEXT NOT NULL DEFAULT ''`, `email TEXT NOT NULL DEFAULT ''`, `updated_at TIMESTAMPTZ NOT NULL`.
- `RequireBearer` middleware gains a `ports.UserRepository` dependency. The upsert runs synchronously on the hot path but does not block or fail the request on error.
- Members who have never authenticated have no row in `users`. `GetByID` on a missing sub returns `(nil, nil)`. The service maps nil to empty strings in the response — never null or omitted.
- Future optimisation: `GetByIDs` (batch lookup) to eliminate the N+1 in `ListMembers`. Acceptable for current scale; tracked as a backlog item.
