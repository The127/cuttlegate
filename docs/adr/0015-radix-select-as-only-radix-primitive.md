# ADR 0015: Radix Select as the only Radix UI primitive

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #192

## Context

Extracting shared UI primitive components (Button, Input, Label, Select, FormField) required a decision on whether to adopt a headless component library. The issue description assumed Radix UI was already installed — it was not. The team needed to decide: install Radix UI broadly, install it selectively, or use native HTML elements throughout.

Alternatives considered:
1. **No Radix** — all five components on native HTML elements + Tailwind only
2. **Full Radix** — install `@radix-ui/react-primitive`, `@radix-ui/react-label`, `@radix-ui/react-select`, and related packages
3. **Radix Select only** — install `@radix-ui/react-select` for the styled select; all other components use native elements

The Select component is the primary driver: native `<select>` cannot be reliably styled across browsers without a custom implementation. Button, Input, Label, and FormField do not require a library — native elements with Tailwind and forwarded refs are sufficient and transparent.

## Decision

Install `@radix-ui/react-select` only. All other primitive components (Button, Input, Label, FormField) are built on native HTML elements.

## Rationale

- **Select is the genuine pain point.** Native `<select>` is hard to style consistently and does not support custom option rendering. Radix Select solves this cleanly with full keyboard navigation and ARIA out of the box.
- **Other components do not need a library.** A `<button>` with Tailwind classes and a forwarded-ref `<input>` are transparent, debuggable, and carry no dependency overhead.
- **Minimal surface.** One package is easier to audit, upgrade, and replace than four. If the team decides to adopt Radix more broadly in a future sprint, that decision can be made with full context at that time — not pre-empted now.
- **Accessibility.** Radix Select handles focus management, keyboard navigation (arrow keys, Escape, Space/Enter), and ARIA roles that are non-trivial to replicate correctly with a native select.

## Consequences

- `@radix-ui/react-select` is a production dependency. Future selects in the SPA should use the shared `Select` component rather than native `<select>` (except in navigation contexts where native is explicitly acceptable, as documented in the component directory README).
- Button, Input, Label, and FormField have no Radix dependency — they can be read and modified without knowledge of Radix internals.
- If a future issue requires Radix for another primitive (e.g. Radix Dialog, Radix Checkbox), a new ADR is not required — but the change should be noted in the component directory README.
- The `--color-accent` CSS variable drives primary button background and focus ring colour. Any future primitive that requires brand-consistent colour must use this variable, not hardcoded Tailwind colour classes.
