# Changelog

All notable changes to the Cuttlegate Go SDK are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `EvalResult.Variant` — the primary field for the variant key, mapped from the
  server-side `value_key` field. For bool flags the value is `"true"` or
  `"false"`; for all other flag types it is the variant key string.
  Always use `Variant` in preference to `Value` for new code.
- `FlagResult.Variant` — same as `EvalResult.Variant`, present on the result
  type returned by `EvaluateFlag`.

### Deprecated

- `EvalResult.Value` — this field is empty for bool flags. Migrate to
  `EvalResult.Variant` for the raw variant key, `Bool()` for boolean evaluation,
  or `String()` for string flags.
- `FlagResult.Value` — same as `EvalResult.Value`. Migrate to
  `FlagResult.Variant`.

### Notes

- `String(ctx, key, evalCtx)` returns `result.Value`, which is empty for bool
  flags. Do not use `String()` to evaluate bool flags — use `Bool()` instead.
  For the raw variant key (including `"true"`/`"false"` for bool flags), read
  `result.Variant` directly.
- Migration: replace `result.Value` with `result.Variant` in all evaluation
  result consumers. For bool flags, replace `result.Value == "true"` with a
  `Bool()` call.
