# Cuttlegate — Project Vision

## What it is

Cuttlegate is a feature flag tool that does one thing exceptionally well. It is open source, precise, and built for anyone who has felt the pain of feature flagging. The codebase and architecture are the product — clean, maintainable, and a pleasure to contribute to.

## Who it is for

Anyone who has shipped software and wished feature flagging was simpler, more predictable, and less tied to a vendor. Developers who want to decouple deployment from release without ceremony.

## What we protect

- **One thing done really well** — any proposal that expands scope beyond feature flagging gets challenged before it gets designed. We are not a platform, a workflow tool, or an experimentation framework. We are a feature flag tool.
- **Documentation parity** — documentation is a first-class deliverable, not an afterthought. No feature is done until it is documented. Getting started must feel effortless. Doc debt is product debt.
- **Clean code and architecture** — the codebase is something contributors are proud to read, not just use. The architecture is honest: the layers mean what they say.
- **Fun** — if working on this stops being fun, something is wrong with the process, not the people.

## What "finished" looks like

A developer finds Cuttlegate, reads the getting started guide, and has a flag evaluating in their app in under ten minutes. No account, no sales call, no enterprise tier. Just a tool that works.

The codebase is something you'd show someone learning how to build a Go service — not as an example of clever, as an example of clear. The architecture is honest, the tests are real, the documentation matches the code.

It has a small, loyal user base of people who chose it because they wanted exactly what it does. The evaluation engine is fast enough that nobody ever has to think about it. A flag check that costs nothing in the path of a request.

A sharp tool that earns its place in someone's stack.

## Milestones as steering

Milestones in Hivetrack are high-level steering tools, not just issue containers. The project owner owns them and updates them to reflect the current vision. If a milestone no longer serves the vision, the project owner changes it.

| Milestone | Purpose |
|---|---|
| M1: Foundation | Core infrastructure — server, domain, auth, persistence |
| M2: Evaluation Engine | Flag evaluation — the core product capability |
| M3: Management UI | SPA for managing flags, projects, environments |
| M4: Client SDKs | SDK surfaces — the primary developer touchpoint |
| M5: Observability | Metrics, audit trail, operational visibility |
| M6: Open Source Ready | Documentation, getting started, onboarding — what makes or breaks adoption |

## Project owner's role

The project owner holds this vision between sessions and has authority to:
- Update Hivetrack milestones to reflect the vision
- Surface risks or drift without being asked
- Push back on scope that conflicts with "one thing done really well"
- Decide when to pull the owner in vs. proceed independently

The project owner does not make product decisions. They surface them early enough that the owner can make them cheaply.
