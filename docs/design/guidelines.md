# Cuttlegate Frontend Design System Guidelines

_This document is the canonical reference for all design decisions in the Cuttlegate SPA.
It supersedes and extends `web/src/design/tokens.md`, which remains the source-of-truth for
raw CSS custom property values. Read this document before implementing any new screen or
modifying an existing one._

---

## 1. Colour Palette

Raw token values and WCAG contrast ratios are defined in `web/src/design/tokens.md`.
This section specifies **how and when** to use each colour.

### Accent colour — indigo-500 (`#6366f1` / dark: `#818cf8`)

- Use for: primary action buttons, focus rings, active links, interactive hover states.
- Use sparingly — one accent per screen is the target. Do not use accent for decorative
  purposes or secondary labels.
- Never use as standalone text on white background for normal-size text. Indigo-500 on white
  is ≈4.3:1, which is borderline. Use only on `text-sm` (14 px) or larger.
- Reference via `var(--color-accent)` or Tailwind `bg-[var(--color-accent)]`. Do not hardcode
  `#6366f1` in component source.

### Status colours

| Meaning | Light token | Dark token | Usage |
|---|---|---|---|
| Enabled / healthy | `#10b981` emerald-500 | `#34d399` emerald-400 | Flag or environment is active |
| Warning / caution | `#fbbf24` amber-400 | `#fcd34d` amber-300 | Partial rollout, caution state |
| Error / disabled | `#ef4444` red-500 | `#f87171` red-400 | Flag disabled, error state |

**Rule:** status colours must never appear as standalone text on white backgrounds. They are
only used in the `StatusBadge` component (dot + text label) and as subtle tinted backgrounds
(`bg-green-50`, `bg-red-50`, `bg-amber-50`) with contrasting text.

### Neutral greys (Tailwind defaults)

| Use | Light | Dark |
|---|---|---|
| Page headings | `text-gray-900` | `text-gray-100` |
| Body copy | `text-gray-700` | `text-gray-300` |
| Secondary text / labels | `text-gray-500` | `text-gray-400` |
| Tertiary / timestamps | `text-gray-400` | `text-gray-500` |
| Card backgrounds | `bg-white` | `bg-gray-800` |
| Page backgrounds | `bg-gray-50` | `bg-gray-900` |
| Borders | `border-gray-200` | `border-gray-700` |
| Subtle dividers | `divide-gray-100` | `divide-gray-700` |

### Radix/Tailwind conflict

Radix UI components (Dialog, Select, etc.) manage their own focus styling. The canonical
resolution is to strip Radix's default outline and apply `focus-visible:ring-2 ring-[var(--color-accent)]`
via the component wrappers in `web/src/components/ui/`. Do not re-override Radix focus styles
in consuming code — override only in the `ui/` wrapper.

---

## 2. Typography Scale

All text uses the Tailwind default `font-sans` (system stack) except identifiers, which use
`font-mono`.

| Element | Tailwind | Weight | Notes |
|---|---|---|---|
| Page heading (h1) | `text-xl` | `font-semibold` | One per page/route |
| Section heading (h2) | `text-sm uppercase tracking-wide` | `font-medium` | `text-gray-500` / `text-gray-400` |
| Card/dialog heading | `text-base` | `font-semibold` | Inside modals or card headers |
| Body copy | `text-sm` | `font-normal` | Default for prose, table cells |
| Secondary / meta | `text-xs` | `font-normal` | Timestamps, helper text |
| Labels (form) | `text-xs` | `font-medium` | `text-gray-500` |
| Monospace identifiers | `text-sm font-mono` | `font-normal` | See below |
| Monospace code samples | `text-xs font-mono` | `font-normal` | Inside `<pre>` or `<code>` |

**Note on page headings:** the codebase inconsistently uses `text-lg` and `text-2xl` for h1.
The canonical size going forward is `text-xl font-semibold`. Existing screens using `text-2xl`
(settings) and `text-lg` (flag list, audit, members) are divergent — this is a known gap.

### When to use monospace

Use `font-mono` for:
- Flag keys (e.g. `my-flag-key`)
- Environment slugs (e.g. `production`)
- Project slugs
- API key prefixes (e.g. `cg_abc123…`)
- Client IDs, user IDs, OIDC subjects
- Evaluation reasons/variant keys in logs or tables
- Inline code samples

Do **not** use monospace for:
- Environment names (display names are prose, not identifiers)
- Button labels, section headings, error messages
- Member names

### Monospace rendering pattern

Identifiers that appear inline are wrapped in a subtle chip:

```tsx
<span className="font-mono text-sm text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-2 py-0.5">
  {identifier}
</span>
```

For copyable identifiers, use the `CopyableCode` component from `web/src/components/ui/`.

---

## 3. Spacing Scale

Base unit is 4 px (Tailwind default). No custom spacing tokens.

**Canonical spacing values:**

| Context | Value |
|---|---|
| Page content padding | `p-6` (24 px) |
| Card/panel internal padding | `p-4` (16 px) |
| List row padding | `px-4 py-3` |
| Form field vertical gap | `space-y-4` |
| Inline element gap | `gap-3` |
| Section gap on a dashboard page | `mt-8` |
| Heading-to-content gap | `mb-6` |

**Inconsistency note:** The projects list home page uses `p-8` while all other screens use
`p-6`. The canonical value going forward is `p-6`. The home page is a known divergence.

**Max-width constraints:**

| Screen type | Max width |
|---|---|
| Narrow (settings, rules editor) | `max-w-2xl` |
| Wide (flag list, members, audit) | `max-w-4xl` or `max-w-5xl` |
| Dashboard overview | `max-w-5xl` |
| Full-bleed tables | No max-width cap |

---

## 4. Component Behaviour Rules

### 4.1 Primary action buttons

**Canonical component:** `<Button>` from `web/src/components/ui/Button.tsx`.

Never use raw `<button className="bg-blue-600 ...">` for primary actions. Always use the
`Button` component with an explicit `variant` prop.

| Action type | Variant | Example |
|---|---|---|
| Primary CTA (create, save) | `variant="primary"` | "New Flag", "Save" |
| Secondary / cancel | `variant="secondary"` | "Cancel" |
| Destructive (delete, revoke) | `variant="destructive"` | "Delete Project" |
| Outline destructive (intent step) | `variant="danger-outline"` | "Revoke", trash icon buttons |
| Text link action | `variant="ghost"` | "Retry", inline text actions |

**Known gap:** segments, environment settings, API keys, and members screens use raw
`<button className="bg-blue-600...">` for primary actions. These must be migrated to
`<Button variant="primary">` in Wave 2.

### 4.2 Destructive action confirmation pattern

Two-step: **intent → confirm**.

Step 1 — intent: user clicks a danger-outline or destructive-outline button. This does not
execute the action.

Step 2 — confirm: a modal dialog appears with:
- A clear heading naming the action ("Delete flag `my-flag-key`?")
- A body explaining irreversibility
- For high-consequence actions (project delete): a typed-name confirmation field
- A cancel button (autofocused by default)
- A destructive confirm button

**Modal component:** use `Dialog` from `web/src/components/ui/Dialog.tsx` (Radix-backed).
Do **not** roll a new `role="dialog"` div with manual `useEscapeKey`. The direct-div pattern
exists in several screens (flags, segments, members, API keys, environment settings) — it is
non-canonical legacy and must not be used in new code.

### 4.3 Loading state pattern

| Situation | Pattern |
|---|---|
| Initial page data load | Skeleton (`animate-pulse` placeholder blocks matching layout) |
| Mutation in progress | `<Button loading={true}>` (inline spinner, button disabled) |
| Background refetch | No visible indicator — TanStack Query handles silently |
| Partial section refresh | Skeleton only if the section had no prior data; otherwise leave stale data visible |

**Never use:** a full-page spinner for any state. Never use a bare `<p>Loading…</p>` — always
use a skeleton that matches the shape of the content.

Skeleton pattern:

```tsx
<div className="h-4 w-32 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
```

Skeletons use `bg-gray-100 dark:bg-gray-700` and `animate-pulse`. Dimensions should approximate
the real content dimensions.

### 4.4 Error message pattern

Page-level errors (query failed to load):

```tsx
<div className="p-6">
  <span className="text-sm text-red-600 dark:text-red-400">{errorMessage} </span>
  <button
    onClick={onRetry}
    className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
  >
    {t('actions.retry', { ns: 'common' })}
  </button>
</div>
```

Inline form field errors (validation):
```tsx
<p id="field-error" className="mt-1 text-xs text-red-600 dark:text-red-400">
  {errorMessage}
</p>
```

Mutation errors that do not affect a specific field go above the submit button, as `text-xs
text-red-600 dark:text-red-400`.

Rules:
- Always show a retry action on page-level fetch errors.
- Never show a raw HTTP status code or stack trace to the user.
- Server errors that the user cannot act on: "Something went wrong. Try again."
- Validation errors the user can fix: state exactly what is wrong and what the valid format is.

### 4.5 Empty state pattern

Empty states must earn trust by explaining what to do next, not just what's absent.

Structure:
```tsx
<div className="text-center py-16 px-6">
  <p className="text-sm text-gray-500 dark:text-gray-400">
    {/* One sentence: what's absent */}
  </p>
  <Button onClick={onCreate} size="lg" className="mt-4">
    {/* CTA: create the first item */}
  </Button>
</div>
```

For screens where no creation is possible from this view (e.g. members empty state when
viewer-role): omit the CTA. Show only the explanatory sentence.

For rule and condition empty states: use a dashed border variant:
```tsx
<div className="border border-dashed border-gray-200 dark:border-gray-600 rounded-lg px-6 py-10 text-center">
```

---

## 5. Status Badge Rules

**Component:** `StatusBadge` from `web/src/components/ui/StatusBadge.tsx`.

### What gets a badge

| Entity | Gets badge? | Status values |
|---|---|---|
| Flag in environment | Yes | `enabled`, `disabled`, `warning` (toggle error) |
| Environment health | Not yet — future work |
| Member role | Role badge, not status badge — use `RoleBadge` in `$slug.members.tsx` |
| API key | Status not surfaced currently |
| Evaluation reason | Inline pill, not `StatusBadge` — uses `REASON_BADGE_CLASS` in evaluations.tsx |

### Status badge copy rules

| Status | Default label | When to override |
|---|---|---|
| `enabled` | "Enabled" | When toggle just failed: pass `label={t('toggle.failed')}` with `status="warning"` |
| `disabled` | "Disabled" | Never — use the default |
| `warning` | "Warning" | When toggle failed; pass translated failure message |
| `unknown` | "Unknown" | Fallback when state is unavailable |

Always pass a translated `label` prop; never rely on the component's hardcoded default label
in production code (the default is English-only).

### Role badges

Member roles use a separate badge pattern (not `StatusBadge`):

| Role | Background | Text |
|---|---|---|
| `admin` | `bg-blue-50 dark:bg-blue-950` | `text-blue-700 dark:text-blue-300` |
| `editor` | `bg-green-50 dark:bg-green-950` | `text-green-700 dark:text-green-300` |
| `viewer` | `bg-gray-50 dark:bg-gray-700` | `text-gray-600 dark:text-gray-300` |

---

## 6. Navigation and Layout

### App shell

The authenticated layout (`_authenticated.tsx`) provides the app shell. Within it, each
route renders its own page content including its own heading and padding.

### Breadcrumbs

`Breadcrumbs` component (`web/src/components/Breadcrumbs.tsx`) provides path-based
breadcrumb navigation. All routes nested below the project level should include breadcrumbs.

### Project switcher

`ProjectSwitcher` component (`web/src/components/ProjectSwitcher.tsx`) in the nav header.
Do not implement secondary project-switching in page content.

---

## 7. Analytics Colour Usage

The `FlagAnalyticsPanel` SVG chart uses its own colour set:

| Use | Value | Source |
|---|---|---|
| Boolean flag: `true` segment | `#22c55e` green-500 | Hardcoded in component |
| Boolean flag: `false` segment | `#f87171` red-400 | Hardcoded in component |
| Multivariate: palette[0] | `#6366f1` indigo-500 | Matches accent token |
| Multivariate: palette[1–4] | amber/cyan/pink/lime | Hardcoded palette |

**Known gap:** the chart colours do not reference CSS custom properties. They will diverge
from the token system if the accent colour changes. A future refactor should map chart colours
to tokens. For now, the colours in the component are canonical for the chart only.

---

## 8. Radix UI Integration Notes

The SPA uses Radix UI primitives via `web/src/components/ui/` wrappers:

| Component | Wrapper file | Notes |
|---|---|---|
| `Dialog` / modal | `ui/Dialog.tsx` | Radix `Dialog.Root` — canonical modal; use this |
| `Select` / dropdown | `ui/Select.tsx` | Radix `Select.Root` — canonical select |
| `Button` | `ui/Button.tsx` | Not Radix — custom `<button>` with variants |
| `Input` | `ui/Input.tsx` | Not Radix — styled `<input>` |
| `StatusBadge` | `ui/StatusBadge.tsx` | Not Radix — pure presentational |

**Radix Dialog backdrop:** Radix `Dialog.Overlay` applies `position: fixed; inset: 0` but
no background tint by default. Our `Dialog.tsx` wrapper applies `bg-black/30` to the overlay.
Do not add a second backdrop in consuming code.

**Radix Dialog focus trap:** Radix Dialog automatically traps focus inside the dialog. Do not
implement `useEscapeKey` or `document.addEventListener('keydown', ...)` in dialogs that use
the Radix wrapper — escape is handled automatically.

**Radix Dialog `aria-hidden`:** Radix sets `aria-hidden="true"` on the rest of the document
(not `aria-modal`) for NVDA/JAWS compatibility. Tests using axe-core must query
`document.body`, not the container, to correctly evaluate dialog accessibility.

---

## 9. i18n Rules

All user-visible strings must use `react-i18next`. No hardcoded English strings in JSX.

- Call `const { t } = useTranslation('<namespace>')` at the top of each component.
- Namespaces: `common` (shared actions/states), `projects`, `flags`, `segments`, `rules`.
- Cross-namespace: `t('actions.retry', { ns: 'common' })`.
- Interpolated JSX (inline HTML): use `<Trans>` with `components` mapping.
- Add new locale keys to `web/public/locales/en/<namespace>.json` before using in code.

---

## 10. Relationship to tokens.md

`web/src/design/tokens.md` remains the authoritative source for:
- CSS custom property names and values
- WCAG contrast ratio tables
- Motion/animation rules

This document (`docs/design/guidelines.md`) is the authoritative source for:
- When and how to use each token
- Component behaviour rules (loading, error, empty state, confirmation)
- Which components are canonical vs. legacy
- Screen-level layout conventions

When the two documents conflict, this document takes precedence on usage rules;
tokens.md takes precedence on raw values.
