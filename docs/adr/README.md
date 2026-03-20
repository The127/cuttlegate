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
| [0005](0005-api-key-evaluation.md) | API key authentication — rejected | Accepted |
| [0006](0006-cuttlegate-as-pure-resource-server.md) | Cuttlegate is a pure resource server — Keyline owns all auth | Accepted |
