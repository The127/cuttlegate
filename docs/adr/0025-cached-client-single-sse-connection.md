# ADR 0025: CachedClient uses a single SSE connection for all flags

**Date:** 2026-03-23
**Status:** Accepted
**Issues:** #231 (Go), #245 (JS)

## Context

`CachedClient` is an in-memory flag cache for production SDK use. It seeds via
`EvaluateAll` on `Bootstrap` and keeps the cache fresh by consuming the SSE
stream at `/flags/stream`. The implementation decision: how many SSE connections
does one `CachedClient` open?

Two alternatives were considered:

**A. One SSE connection per flag (per-flag goroutines).** Each cached flag key
gets its own goroutine calling `Subscribe(ctx, key)`. Updates are delivered
per-key with no filtering needed on the client side.

**B. One SSE connection for all flags (single goroutine).** A single goroutine
connects to `/flags/stream`, which delivers events for every flag in the
environment. The goroutine applies events only to keys already in the cache;
unknown keys are ignored.

## Decision

One SSE connection for all flags (option B). `CachedClient` starts exactly one
background goroutine in `Bootstrap`. That goroutine manages one long-lived SSE
connection and applies `flag.state_changed` events to the in-memory cache via a
`sync.RWMutex`.

## Rationale

**Connection count scales with cache size under option A.** A project with 100
flags would open 100 simultaneous SSE connections. The server's in-process fan-out
broker (see ADR-0011) handles this, but the SDK becomes a resource liability at
scale.

**Thundering herd on auth failure.** If the service token is rotated or expires,
100 goroutines all receive a terminal 401 simultaneously and all stop at once.
Under option B, one goroutine stops. The failure mode is contained and predictable.

**Reconnect backoff is shared.** Under option A, each goroutine runs its own
backoff independently. Under option B, there is one backoff sequence — simpler
to reason about, simpler to test, simpler to observe.

**The `/flags/stream` endpoint already delivers all flags.** There is no server-side
cost savings from per-flag connections. The filtering that per-flag connections
provide (routing each event to the right goroutine) is trivially replaced by a
client-side key check (`if _, ok := cache[key]; !ok { return }`).

**The `sync.RWMutex` model is sufficient.** Cache updates happen only on SSE events
(rare writes). Reads dominate. A single `RWMutex` provides correct concurrent
access without additional complexity.

## Consequences

### Go SDK

- `CachedClient.Bootstrap` starts exactly one goroutine. The goroutine stops when
  the context passed to `Bootstrap` is cancelled.
- SSE events for flag keys not present in the cache (i.e. flags added after
  `Bootstrap`) are silently ignored. `Bool`/`String` for those keys fall back to
  live HTTP.
- `CachedClient.Subscribe` delegates to the inner client. It opens a separate
  SSE connection, independent of the cache connection. This is correct: callers
  using `Subscribe` directly are opting into per-key real-time delivery; they are
  not the `CachedClient`'s cache management concern.
- If a new flag is deployed after `Bootstrap`, the only way to add it to the cache
  is to call `Bootstrap` again. A follow-up issue could introduce a TTL-based
  re-bootstrap if this becomes a production pain point.

### JS SDK (#245)

The JS `createCachedClient` follows the same single-connection principle with
these JS-specific characteristics:

- One `connectStream` call manages the single SSE connection. The connection loop
  is already built into `connectStream` with exponential backoff — `CachedClient`
  does not add its own retry logic.
- **SSE-first ordering**: `connectStream` is called before the HTTP hydration
  `evaluate()` call. SSE events received during hydration are buffered and applied
  on top of the hydration result to close the missed-event gap.
- The JS SSE event (`FlagStateChangedEvent`) carries only `flagKey` and `enabled`.
  SSE updates preserve `valueKey` and `reason` from hydration; `reason` is set to
  `"default"` after an SSE update (same behaviour as `CuttlegateProvider`).
- No HTTP fallback on cache miss: unknown keys return `reason: "not_found"`. The
  JS CachedClient is browser-first; fallback HTTP would re-expose the token path.
- `ready: Promise<void>` resolves when HTTP hydration completes. Rejects with
  `CuttlegateError` on auth failure (401/403) or timeout during hydration.
- Terminal SSE auth errors (401/403) are surfaced via `onError` in
  `CachedClientOptions`; the cache retains its last-known values.
