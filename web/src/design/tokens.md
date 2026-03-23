# Cuttlegate Design Tokens

This document is the source-of-truth for all design tokens used in the Cuttlegate UI.
Tokens are implemented as CSS custom properties in `src/styles.css` under `@theme {}` and
are available as Tailwind utilities (e.g. `bg-accent`, `text-status-enabled`, `font-mono`).

---

## Colour Palette

### Accent

| Token | Value | Tailwind utility |
|---|---|---|
| `--color-accent` | `#6366f1` (indigo-500) | `bg-accent`, `text-accent`, `border-accent` |

Use the accent colour for primary actions (buttons, focus rings, links). Use sparingly — one
accent per screen is the target.

**WCAG AA note:** indigo-500 on white is approximately 4.3:1 — borderline for normal text.
Always use accent colour text at `text-sm` (14px) or larger, and on consistent backgrounds.
Focus rings and borders do not require 4.5:1 (only 3:1 needed for UI components).

### Status colours

| Token | Value | Tailwind utility | Meaning |
|---|---|---|---|
| `--color-status-enabled` | `#10b981` (emerald-500) | `text-status-enabled`, `bg-status-enabled` | Flag enabled / healthy |
| `--color-status-warning` | `#fbbf24` (amber-400) | `text-status-warning`, `bg-status-warning` | Partial rollout / caution |
| `--color-status-error` | `#ef4444` (red-500) | `text-status-error`, `bg-status-error` | Flag disabled / error |

#### WCAG 2.1 AA usage rule

Status colours are designed for use on dark backgrounds (`gray-900`/`gray-800`). On white
backgrounds, emerald-500 and amber-400 do not achieve 4.5:1 for normal text and must not
be used as standalone text colour on white.

**Rule:** status colour dots/badges must always include a text label. Standalone coloured text
on white is not permitted for status communication. On dark backgrounds all four tokens pass
AA for normal text.

Approximate contrast on `#111827` (gray-900, luminance ≈ 0.018):

| Colour | Contrast on gray-900 | AA normal text |
|---|---|---|
| indigo-500 `#6366f1` | ~6:1 | Pass |
| emerald-500 `#10b981` | ~5:1 | Pass |
| amber-400 `#fbbf24` | ~10:1 | Pass |
| red-500 `#ef4444` | ~7:1 | Pass |

---

## Typography

### Monospace font

| Token | Value | Tailwind utility |
|---|---|---|
| `--font-mono` | `'JetBrains Mono', ui-monospace, monospace` | `font-mono` |

Self-hosted from `web/public/fonts/JetBrainsMono[wght].woff2` via `@font-face` in `src/styles.css`.
Variable font, Latin subset, weight range 100–800. No CDN dependency. Used for:
- Flag keys and slugs
- Environment IDs and slugs
- API key previews
- Code samples

All other UI copy uses the system sans-serif stack (Tailwind default `font-sans`).

### Type scale

No custom type scale — Tailwind's default scale applies (`text-xs` through `text-4xl`).
Font sizes for common elements:

| Element | Size | Weight |
|---|---|---|
| Page heading | `text-xl` | `font-semibold` |
| Section heading | `text-base` | `font-semibold` |
| Body copy | `text-sm` | `font-normal` |
| Monospace identifiers | `text-sm` | `font-normal` |
| Labels / captions | `text-xs` | `font-medium` |

---

## Spacing

Base unit: **4px** (Tailwind default). No custom spacing scale.

Use Tailwind's standard scale: `p-1` (4px), `p-2` (8px), `p-4` (16px), `p-6` (24px), `p-8` (32px).

---

## Motion

**Principle: functional only.** No decorative animation.

| Use case | Duration | Property |
|---|---|---|
| Status colour change (enabled ↔ disabled) | `150ms` | `color`, `background-color` |
| Button hover/focus state | `150ms` | `background-color`, `border-color` |
| All other transitions | None | — |

Apply via Tailwind: `transition-colors duration-150`.

No transforms, no opacity fades, no enter/exit animations — unless a future sprint explicitly
adds them with an accessibility review (`prefers-reduced-motion` must be honoured).
