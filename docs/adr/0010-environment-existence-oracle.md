# ADR 0010: Environment existence oracle — always 403 for unknown environments

**Date:** 2026-03-21
**Status:** Accepted
**Issue:** #106

## Context

The flag evaluation endpoint must handle requests for environments that do not exist. Two behaviours were considered:

1. **Always 403** — all callers receive `403 Forbidden` for unknown environments, regardless of project membership. This is the current implementation, established in #125 and maintained in #106.
2. **Differentiated** — non-members receive `403 Forbidden`; authenticated project members receive `404 Not Found`, revealing that the environment does not exist.

During Sprint 5 grooming, BDD scenarios for #106 specified option 2 (members get 404). During implementation, the team followed option 1 (the established pattern). The retro surfaced this as an unresolved product decision.

## Decision

**Always return 403 for unknown environments**, regardless of the caller's project membership.

## Rationale

- **Security default:** a 404 for members creates an environment enumeration surface. An attacker who compromises a member's credentials can probe for environment names. The information gained is low-value, but the cost of preventing it (always 403) is zero.
- **Simplicity:** a single code path for unknown environments is easier to reason about, test, and explain than membership-conditional responses.
- **Consistency:** the evaluation endpoint already returns 403 for unknown projects. Returning 403 for unknown environments maintains a uniform error posture across all resource resolution failures.

The trade-off is that authenticated members cannot distinguish "this environment doesn't exist" from "I don't have access to this environment" — they must check the environment list separately. This is acceptable for an SDK-consumed endpoint where the caller typically knows which environments exist.

## Consequences

- All evaluation endpoint handlers return 403 for any unresolvable project or environment — no exceptions.
- BDD scenarios for evaluation endpoints must use 403 (not 404) for unknown environments.
- If a future product decision reverses this (e.g., to improve the developer experience for dashboard users), a new ADR should supersede this one and the change should be applied consistently across all endpoints.
