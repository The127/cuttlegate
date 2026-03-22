# ADR 0019: arch_test.go does not cover external _test packages

**Date:** 2026-03-22
**Status:** Resolved — gap closed by issue #221
**Issue:** #221

## Context

`arch_test.go` uses `golang.org/x/tools/go/packages` to load all packages under the module and check import rules. The `packages.Config` does not set `Tests: true`, which means the loader excludes external test packages (those declared `package foo_test`). Only non-test packages and internal test packages (`package foo`) are checked.

Cuttlegate uses external test packages in several places — for example, `internal/app/fake_segment_repository_test.go` is declared `package app_test`. A layer violation introduced inside an external `_test` package (e.g. an `app_test` file importing an adapter package directly) would not be caught by the current arch test. Test helpers have historically been a place where shortcuts are taken under the assumption that "it's just tests."

Two options:

1. **Set `Tests: true`** in `packages.Config` so external test packages are loaded and their imports checked against the same rules.
2. **Accept the gap** and document it, relying on code review to catch violations in test files.

## Decision

**Accept the gap for now.** Do not set `Tests: true` yet. *(Original decision — see Resolution below.)*

Setting `Tests: true` causes `packages.Load` to synthesise additional packages (test variants of each package) and can produce false positives — specifically, test framework imports and `_test` package cross-imports that are valid in test context but look like violations to a naive checker. Handling these correctly requires distinguishing test-only imports from production imports, which adds non-trivial complexity to `checkRule`.

A follow-up issue (#221) tracks the correct fix: extend `arch_test.go` to set `Tests: true` with appropriate handling for test-only import patterns.

## Rationale

The immediate cost of the gap is low: external test packages are reviewed by humans and the codebase currently has no violations. The cost of a broken or false-positive arch test is high — developers learn to ignore it, which eliminates its value entirely. A partial fix is worse than a clear documented gap.

## Resolution (issue #221)

The gap is closed. `arch_test.go` now sets `Tests: true` in both `TestImportRules` and `TestCompositionRootExclusivity`.

The key implementation insight: `packages.Load` with `Tests: true` produces three categories of test-related package variants in addition to production packages:
- External test packages (PkgPath ends with `_test`, e.g. `…/adapters/http_test`)
- Test binary packages (PkgPath ends with `.test`, e.g. `…/adapters/http.test`)
- Synthesised test variants (PkgPath contains `[`, e.g. `… [….test]`)

These are identified by the `isTestPackage` predicate. Production purity rules (Rules 1, 3, 4) do not fire for test packages — test files legitimately import test frameworks and helpers. Rule 2 (no cross-adapter imports) is enforced for test packages via `checkTestImportRule`, which is self-guarding and called from both test function loops.

The test binary's import of its own external test package (e.g. `http.test` importing `http_test`) is not flagged as a cross-adapter violation because `isCrossAdapterImport` skips imports that are themselves test packages.

## Consequences

- External `_test` packages are now covered by Rule 2 (no cross-adapter imports).
- Production purity rules (domain stdlib-only, app layer domain-only) are not applied to test packages — test files may import test frameworks and helpers freely.
- The arch test is slightly slower due to loading more packages; this is acceptable.
