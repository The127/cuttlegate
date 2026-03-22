# ADR 0019: arch_test.go does not cover external _test packages

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #221

## Context

`arch_test.go` uses `golang.org/x/tools/go/packages` to load all packages under the module and check import rules. The `packages.Config` does not set `Tests: true`, which means the loader excludes external test packages (those declared `package foo_test`). Only non-test packages and internal test packages (`package foo`) are checked.

Cuttlegate uses external test packages in several places — for example, `internal/app/fake_segment_repository_test.go` is declared `package app_test`. A layer violation introduced inside an external `_test` package (e.g. an `app_test` file importing an adapter package directly) would not be caught by the current arch test. Test helpers have historically been a place where shortcuts are taken under the assumption that "it's just tests."

Two options:

1. **Set `Tests: true`** in `packages.Config` so external test packages are loaded and their imports checked against the same rules.
2. **Accept the gap** and document it, relying on code review to catch violations in test files.

## Decision

**Accept the gap for now.** Do not set `Tests: true` yet.

Setting `Tests: true` causes `packages.Load` to synthesise additional packages (test variants of each package) and can produce false positives — specifically, test framework imports and `_test` package cross-imports that are valid in test context but look like violations to a naive checker. Handling these correctly requires distinguishing test-only imports from production imports, which adds non-trivial complexity to `checkRule`.

A follow-up issue (#221) tracks the correct fix: extend `arch_test.go` to set `Tests: true` with appropriate handling for test-only import patterns.

## Rationale

The immediate cost of the gap is low: external test packages are reviewed by humans and the codebase currently has no violations. The cost of a broken or false-positive arch test is high — developers learn to ignore it, which eliminates its value entirely. A partial fix is worse than a clear documented gap.

## Consequences

- External `_test` packages are not covered by import rule checks.
- Developers must not introduce adapter imports in external test packages without a code review comment acknowledging this gap.
- Issue #221 tracks the fix. Until it is closed, this ADR is the canonical record of the known blind spot.
- When #221 is implemented, update this ADR status to reflect the resolved state.
