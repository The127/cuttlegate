# 0009 — JS/TS SDK: single package with dual entry points

**Status:** Accepted
**Date:** 2026-03-21

## Context

Sprint 5 introduces the first JS/TS SDK for Cuttlegate. Two packaging options were considered:

1. **Two separate packages** — `@cuttlegate/browser` and `@cuttlegate/node`
2. **Single package** — `@cuttlegate/sdk` with browser and node entry points via the `package.json` `exports` map

## Decision

Ship a **single `@cuttlegate/sdk` package** with dual entry points declared in the `exports` map:

```json
"exports": {
  ".": {
    "browser": "./dist/browser/index.js",
    "import": "./dist/index.js",
    "require": "./dist/index.cjs"
  }
}
```

The browser build is produced with tsup's `platform: 'browser'` option, excluding Node built-ins. The ESM and CJS builds target Node consumers. Build tool is **tsup**.

TypeScript types are defined in the SDK as a **consumer-side contract** — they are not imported from the server codebase. If the server wire format changes, the SDK takes a breaking change and a semver-major bump. That is the correct relationship.

## Rationale

- A single package means one version number, one publish pipeline, and one set of release notes for consumers to track.
- The `exports` map with `browser`, `import`, and `require` conditions is well-supported by all major bundlers (Vite, webpack, esbuild, Rollup) and Node.js ≥ 12.
- The browser and node builds do not have radically different dependency graphs at this stage — the primary difference is platform (`fetch` vs Node HTTP), which the `exports` condition handles cleanly.
- If the two entry points diverge significantly in the future (e.g. a native node addon, a Service Worker layer in the browser), splitting into separate packages is straightforward — SDK consumers update one import, not two.

## Alternatives rejected

**Two packages (`@cuttlegate/browser` + `@cuttlegate/node`)**
- Two version numbers to keep in sync — a consumer using both must manage two independent semver constraints.
- Two publish pipelines — doubles CI and release overhead.
- No concrete technical benefit at the current scope.

## Consequences

- Both entry points must be built and type-checked independently in CI (separate tsup entries, separate `dts` output).
- Any breaking change to a shared wire-format type affects both consumers simultaneously — which is correct.
- The `exports` map condition order matters: `browser` must come before `import`/`require` to be selected by bundlers that evaluate conditions in declaration order.
- The SDK CI runs as a **separate job** from Go CI — different runtime, different dependencies, failures are independent.
