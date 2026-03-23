# ADR 0027: Python SDK — sync-first client, httpx, PEP 544 protocol

**Date:** 2026-03-23
**Status:** Accepted
**Issues:** #65 (init & config), #66 (evaluation), #67 (SSE streaming)

## Context

Sprint 16 shipped the Python SDK foundation: `CuttlegateClient` (evaluation),
`connect_stream` (SSE), `MockCuttlegateClient` (test helper), and
`CuttlegateClientProtocol` (type contract). Three design decisions were made
that have long-term consequences and warrant recording.

**1. Sync-first vs. async-first.** Python's ecosystem has both. `httpx` supports
both sync and async interfaces. The Go and JS SDKs are single-paradigm; Python
requires an explicit choice at the SDK design level. Server-side Python (Django,
Flask, WSGI) is predominantly synchronous. Async frameworks (FastAPI, ASGI) are
growing but not yet the majority runtime. A sync-first SDK covers the larger
install base immediately without blocking an async follow-on.

**2. httpx vs. requests.** `requests` is the most widely used Python HTTP
library but does not support async. `httpx` supports both sync and async behind
the same API surface — choosing `httpx` keeps the async path open without a
library swap. `httpx>=0.27` requires Python 3.8+ and is stable for production use.
`aiohttp` was rejected because it is async-only and would require a parallel
sync implementation.

**3. PEP 544 structural protocol vs. ABC.** `CuttlegateClientProtocol` is defined
as a `typing.Protocol` (PEP 544) rather than an abstract base class. This allows
consumers to write test doubles (`MockCuttlegateClient`, or their own) without
subclassing — structural compatibility is checked statically by type checkers,
not at runtime. An ABC would force subclassing and create a coupling between the
consumer's test double and the SDK's class hierarchy.

## Decision

1. The Python SDK is **sync-first**. The public API (`CuttlegateClient`,
   `connect_stream`) is synchronous. Async support is explicitly deferred to a
   follow-on issue.

2. The HTTP layer uses **`httpx>=0.27`**. This is the only required dependency.
   `requests` is not used. The choice of httpx is permanent unless a compelling
   reason to change arises — swapping HTTP libraries in a shipped SDK is a
   breaking change for consumers who pin transitive dependencies.

3. The type contract is a **PEP 544 `Protocol`** (`CuttlegateClientProtocol`),
   not an ABC. `CuttlegateClient` and `MockCuttlegateClient` both satisfy the
   protocol structurally. Consumers SHOULD type-hint against
   `CuttlegateClientProtocol`, not the concrete class.

## Rationale

- Sync-first maximises day-one install-base coverage. WSGI apps (Django, Flask)
  can adopt the SDK without any async machinery.
- `httpx` provides a credible async migration path: the same config, error types,
  and request structure can be reused in a future `AsyncCuttlegateClient` without
  introducing a second HTTP library.
- PEP 544 protocols are the idiomatic Python approach to interface abstraction
  since 3.8. ABCs add inheritance coupling that makes test doubles harder to write
  and maintain.

## Consequences

- A future `AsyncCuttlegateClient` will use `httpx.AsyncClient`. The config,
  error types, and protocol are already compatible — only the client class itself
  needs to be added.
- `CachedClient` (in-memory cache backed by SSE, analogous to Go and JS
  CachedClient) is deferred. It requires the async client or a threading model
  equivalent to the Go `sync.RWMutex` approach. A follow-on issue will track this.
- Consumers who pin `httpx` explicitly and pin to a version below 0.27 will have
  a dependency conflict. This is the expected cost of specifying a minimum version.
- The `CuttlegateClientProtocol` is the v1 contract for consumers. Adding methods
  to it is a breaking change for any consumer who has implemented the protocol.
  New methods must be added with a default implementation or versioned separately.
