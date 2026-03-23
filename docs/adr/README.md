# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) — short documents that capture significant technical decisions made in this project.

## When to write an ADR

Write an ADR whenever a decision:

- Is hard to reverse (architectural commitments, data model choices, auth flows)
- Affects layer boundaries or import rules
- Chooses between two real alternatives with non-obvious tradeoffs
- Will confuse a future developer who reads the code without context

Do **not** write an ADR for implementation details, library configuration, or anything reversible without significant cost.

## Naming convention

```
NNNN-short-title-in-kebab-case.md
```

Numbering is zero-padded to four digits: `0001`, `0002`, `0003`, …

## Format

See `template.md`. Status values: `Proposed`, `Accepted`, `Deprecated`, `Superseded by ADR NNNN`.

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-domain-event-interface-not-struct.md) | DomainEvent as interface, not struct | Accepted |
| [0002](0002-test-pyramid-strategy.md) | Test pyramid strategy | Accepted |
| [0003](0003-ports-and-adapters-architecture.md) | Ports and adapters architecture | Accepted |
| [0004](0004-ai-development-tooling-graphiti-axon.md) | AI development tooling evaluation — Graphiti and Axon | Accepted |
| [0005](0005-api-key-evaluation.md) | API key authentication — rejected | Superseded by ADR 0012 |
| [0006](0006-cuttlegate-as-pure-resource-server.md) | Cuttlegate is a pure resource server — Keyline owns all auth | Accepted |
| [0007](0007-api-versioning-strategy.md) | API versioning strategy — URL prefix `/api/v1/` | Accepted |
| [0008](0008-rbac-in-app-layer.md) | RBAC enforced in the app layer, not the HTTP adapter | Accepted |
| [0009](0009-js-sdk-single-package.md) | JS/TS SDK: single package with dual entry points | Accepted |
| [0011](0011-in-process-event-fanout.md) | In-process event fanout for SSE delivery | Accepted |
| [0012](0012-api-key-second-auth-path.md) | API key as second authentication path for the evaluation endpoint | Accepted |
| [0013](0013-transactional-unit-of-work.md) | Transactional unit of work for multi-repository operations | Accepted |
| [0014](0014-typed-entity-ids.md) | Typed entity IDs to replace bare string IDs | Accepted |
| [0015](0015-radix-select-as-only-radix-primitive.md) | Radix Select as the only Radix UI primitive | Accepted |
| [0016](0016-user-profile-cache.md) | User profile cache — upsert from OIDC token claims | Accepted |
| [0017](0017-oidc-missing-role-policy.md) | OIDC missing role claim — reject by default, viewer opt-in | Accepted |
| [0010](0010-environment-existence-oracle.md) | Environment existence oracle — always 403 for unknown environments | Accepted |
| [0018](0018-eval-response-value-key-field.md) | Evaluation response — add `value_key` field, clarify v1 contract lock scope | Accepted |
| [0019](0019-arch-test-external-test-package-coverage.md) | arch_test.go does not cover external _test packages | Accepted |
| [0020](0020-uuid-vs-text-pk-inconsistency.md) | UUID vs TEXT primary key inconsistency in rules and segments | Accepted |
| [0021](0021-fire-and-forget-evaluation-event-publish.md) | Fire-and-forget goroutine for evaluation event publish | Accepted |
| [0022](0022-evaluation-event-multi-subscriber-fanout.md) | Evaluation event fanout to multiple independent subscribers | Accepted |
| [0023](0023-in-house-component-library.md) | In-house UI component library over third-party component frameworks | Accepted |
| [0024](0024-evaluation-event-retention-policy.md) | Evaluation event retention policy — 30-day default with configurable cleanup | Accepted |
| [0025](0025-cached-client-single-sse-connection.md) | CachedClient uses a single SSE connection for all flags | Accepted |
| [0026](0026-polling-for-evaluation-stats-not-sse.md) | Polling for evaluation stats refresh, not SSE | Accepted |
| [0027](0027-python-sdk-sync-first-httpx-pep544.md) | Python SDK — sync-first client, httpx, PEP 544 protocol | Accepted |
