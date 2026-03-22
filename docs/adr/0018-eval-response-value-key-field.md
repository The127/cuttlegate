# ADR 0018: Evaluation response — add `value_key` field, clarify v1 contract lock scope

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #211

## Context

The v1 evaluation response type (`EvalView`) uses `Value *string` to return the evaluated variant key. For bool flags `Value` is `nil` — consumers must check `Type` before deciding whether to dereference it. For all other flag types `Value` is non-nil and contains the variant key.

This nil-encodes type information in an optionality signal. Every consumer (SPA, JS SDK, Go SDK) must special-case bool flags, which produces conditional branches in Zod schemas, SDK wrapper code, and integration tests.

The Sprint 3 contract lock ("the evaluation endpoint response shape is the v1 SDK contract — locked") was understood to mean the shape cannot change. A strict reading blocks any fix. A precise reading distinguishes:

- **Removing or renaming existing fields** — breaking; blocked by the lock
- **Adding new fields** — non-breaking for any well-behaved consumer that ignores unknown JSON keys (Go `encoding/json` and JS `JSON.parse` both satisfy this)

Two options were debated:

- **Option A:** Add `value_key string` to the v1 response — always present, always the variant key as a string (`"true"`/`"false"` for bool flags). Deprecate `Value *string` in docs.
- **Option B:** Introduce a `/api/v2/` evaluation endpoint with a clean, consistent shape. Freeze v1.

## Decision

**Option A.** Add `value_key string` to `EvalView`. It is present for all flag types. For bool flags it is `"true"` or `"false"`. For all other types it is the variant key string.

`Value *string` is not removed. It remains in v1 with its existing nil-for-bool semantics, deprecated in documentation and SDK changelogs. It will be removed when v2 is introduced.

The Sprint 3 contract lock is clarified to mean: **existing fields are frozen** (no removal, no rename, no semantic change). Additive fields are permitted within v1 under standard backwards-compatibility rules.

v2 is deferred until a second breaking-level change justifies the migration cost. "Use v2" is not an acceptable response to a single wart.

## Rationale

Option A is non-breaking and solves the consumer problem now. Option B is architecturally cleaner but expensive: it requires versioning Go types, updating all SDKs, writing a migration guide, and maintaining two endpoints indefinitely. One nil-pointer wart does not justify that cost. The threshold for v2 is two or more breaking-level changes, not one.

Naming `value_key` (not `raw_value`) is deliberate: it names what the field contains (the variant key), not how it differs from the old field. SDK docs should lead with `value_key`; `value` should be marked deprecated on first mention.

## Consequences

- `EvalView` gains `ValueKey string` (Go) / `value_key` (JSON) — always non-nil, always the variant key
- `Value *string` remains but is deprecated; SDK changelogs must note this
- SPA Zod schema can drop its `if/then` branch on `type`; use `value_key` unconditionally
- Go SDK and JS SDK must expose `value_key` and mark `value` as deprecated in their public APIs
- Future maintainers: do not introduce v2 for a single breaking change — accumulate them
- ADR 0007 (URL versioning strategy) is not affected; this is an additive field change within v1
