# Changelog

All notable changes to the Cuttlegate JavaScript/TypeScript SDK are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `EvaluationResult.valueKey` — the primary field for the variant key, mapped
  from the server-side `value_key` field. For bool flags the value is `"true"`
  or `"false"`; for all other flag types it is the variant key string.
  Always use `valueKey` in preference to `value` for new code.
- `FlagResult.valueKey` — same as `EvaluationResult.valueKey`, present on the
  result type returned by `evaluateFlag()`.

### Deprecated

- `EvaluationResult.value` — this field is `null` for bool flags. Migrate to
  `valueKey` for the raw variant key. The `@deprecated` JSDoc annotation is
  already present in the TypeScript types.
- `FlagResult.value` — same as `EvaluationResult.value`. Migrate to
  `FlagResult.valueKey`.

### Notes

- Migration: replace `result.value` with `result.valueKey` in all evaluation
  result consumers. For bool flags, replace `result.value === 'true'` with
  `result.valueKey === 'true'` or simply use `result.enabled`.
- `value` is `null` (not an empty string) for bool flags in the JS SDK.
  `valueKey` is always a non-null string — `"true"` or `"false"` for bool
  flags, the variant key string for all other types.
