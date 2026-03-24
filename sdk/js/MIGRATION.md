# Migrating to JS SDK v2

This guide covers the breaking changes in `@cuttlegate/sdk` v2.0.

## Method renames

| v1 | v2 | Notes |
|---|---|---|
| `evaluateFlag(key, context)` | `evaluate(key, context)` | Single-flag evaluation |
| `evaluate(context)` | `evaluateAll(context)` | Bulk evaluation (all flags) |

`evaluateFlag` still works as a deprecated alias but will be removed in v3.

```ts
// v1
const result = await client.evaluateFlag('dark-mode', ctx);
const all = await client.evaluate(ctx);

// v2
const result = await client.evaluate('dark-mode', ctx);
const all = await client.evaluateAll(ctx);
```

## New convenience methods

```ts
const enabled = await client.bool('dark-mode', ctx);   // boolean
const variant = await client.string('banner-text', ctx); // string
```

## Result type rename

| v1 | v2 |
|---|---|
| `EvaluationResult` | `EvalResult` |

`EvaluationResult` still exists as a deprecated type alias.

## Field rename: `valueKey` → `variant`

The primary field on `EvalResult` is now `variant`:

```ts
// v1
console.log(result.valueKey);

// v2
console.log(result.variant);
```

The deprecated `value` field (which was `null` for bool flags) is still present but will be removed in v3.

## React hook: `useFlagVariant`

The return shape changed from `{ value, loading }` to `{ variant, loading }`:

```tsx
// v1
const { value, loading } = useFlagVariant('banner-text');

// v2
const { variant, loading } = useFlagVariant('banner-text');
```

## Cross-SDK consistency

These renames align the JS SDK with the Go and Python SDKs:

| Concept | Go | Python | JS (v2) |
|---|---|---|---|
| Single eval | `Evaluate` | `evaluate` | `evaluate` |
| Bulk eval | `EvaluateAll` | `evaluate_all` | `evaluateAll` |
| Result type | `EvalResult` | `EvalResult` | `EvalResult` |
| Variant field | `Variant` | `variant` | `variant` |
| Bool helper | `Bool` | `bool` | `bool` |
| String helper | `String` | `string` | `string` |
