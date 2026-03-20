# ADR 0005: API key authentication — rejected

**Date:** 2026-03-20
**Status:** Accepted
**Issue:** #24

## Context

Cuttlegate supports two OAuth 2.0 auth flows for non-browser clients:

- **Device Authorization Grant (RFC 8628)** — for SDK and CLI tools that need interactive user approval.
- **Client Credentials Grant** — for service accounts and automated pipelines that authenticate without user interaction.

Before closing Sprint 1, the team evaluated whether a third mechanism — **API keys** — was still necessary given both flows are implemented.

The primary motivating scenario: a backend script or developer tooling that needs to call the Cuttlegate API programmatically without managing the OAuth client credentials flow.

## Decision

**Reject API keys.** No API key implementation will be added to Cuttlegate.

## Rationale

**Client credentials covers all M2M use cases.**
A service account is provisioned once (admin step), its `client_id` and `client_secret` are stored by the caller, and a token is obtained with a single POST to `/auth/token`. For a one-off script, this is one extra HTTP call. For a CI pipeline, a service account is the right identity model regardless — it ties usage to a named identity with an explicit role, and the resulting token is short-lived.

**API keys would not be simpler in practice.**
A secure API key implementation requires: scopes, TTL, revocation, Argon2id hashing of stored keys (the same requirement applied to service account secrets). A minimal "API key" that lacks these is less secure than client credentials, not more convenient. We would be adding implementation complexity to produce a weaker alternative.

**No remaining use case is uncovered.**
The scenarios evaluated:

| Scenario | Covered by |
|---|---|
| SDK auth in interactive environments (CLI, dev tools) | Device flow (#22) |
| M2M: persistent automated services and pipelines | Client credentials (#23) |
| M2M: serverless functions (no persistent state between invocations) | Client credentials — one `POST /auth/token` per invocation is cheap and correct |
| Environments without OIDC | Out of scope — Cuttlegate requires an OIDC provider |

**Device flow covers SDK interactive auth.**
SDKs embedded in user-facing tools obtain a token via device flow, which is standard OAuth 2.0 and widely supported. API keys would not improve this scenario.

## Consequences

- No API key implementation will be built in any sprint. This decision is permanent unless the team encounters a concrete use case that neither device flow nor client credentials can serve.
- Future requests to add API keys must reference this ADR and identify a specific gap. The gap named here (simple script access) is not sufficient justification on its own.
- Service account client credentials are the recommended path for all programmatic Cuttlegate access. The admin tooling for service account management (`POST /service-accounts`, rotate, deactivate) is built in Sprint 1 (#23).
