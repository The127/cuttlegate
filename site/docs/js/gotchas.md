---
sidebar_label: Gotchas
sidebar_position: 4
---

# Gotchas & Known Behaviours

This page documents non-obvious runtime behaviours in the Cuttlegate JS/TS SDK. Each entry describes what happens, why, and whether you need to do anything about it.

## `connectStream()` abort and pending reads

**What happens:** Calling `controller.abort()` on an `AbortController` aborts the underlying `fetch` request, but it does **not** unblock a pending `reader.read()` call on the response body's `ReadableStream`. The read promise hangs indefinitely.

**Why:** This is a platform behaviour of the [Streams API](https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream), not a Cuttlegate bug. `AbortSignal` cancels the fetch, but `ReadableStreamDefaultReader.read()` has no built-in awareness of the signal.

**Does the SDK handle this?** Yes. The SDK's `connectStream()` function registers an abort listener that calls `reader.cancel()` when the signal fires. If you use `connectStream()` normally, you do not need to do anything:

```typescript
import { connectStream } from '@cuttlegate/sdk';

const conn = connectStream(config, {
  onFlagChange: (event) => console.log(event.flagKey, event.enabled),
});

// This cleanly closes the stream — no hanging reads.
conn.close();
```

**When you need to act:** If you are building custom lifecycle management and reading from a raw `ReadableStream` yourself (for example, wrapping the SSE stream in your own parser), you must explicitly cancel the reader when the abort signal fires:

```typescript
const controller = new AbortController();

const res = await fetch(streamUrl, {
  headers: { Authorization: `Bearer ${token}` },
  signal: controller.signal,
});

const reader = res.body!.getReader();

// Without this listener, reader.read() hangs after abort.
controller.signal.addEventListener('abort', () => reader.cancel());

// Now controller.abort() will unblock reader.read().
```
