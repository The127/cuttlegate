# ADR 0023: In-house UI component library over third-party component frameworks

**Date:** 2026-03-22
**Status:** Accepted
**Issue:** #201, #192

## Context

The Cuttlegate SPA needs a consistent set of UI primitives: buttons, badges, data tables, copy-able code blocks, form inputs, and labels. Two broad paths exist:

1. **Third-party component framework** — shadcn/ui, MUI, Ant Design, Chakra UI, or similar. Pre-built components, pre-designed styles, large ecosystems. Cuttlegate's design tokens and branding must be adapted to fit the framework's theming model.
2. **In-house primitives** — thin components built from scratch (or from minimal unstyled primitives like Radix UI) styled with Tailwind. Full control over markup, styling, and accessibility. No dependency on a third-party design system.

ADR 0015 already established that Radix Select is used for the select primitive. This ADR generalises that decision to the full component library.

## Decision

**Build and maintain an in-house component library** (`web/src/components/ui/`). Components are built from scratch using Tailwind CSS, with Radix UI primitives used selectively for complex interaction patterns (dropdowns, modals, tooltips) where accessibility-correct keyboard handling is non-trivial to implement correctly.

The library currently includes: `Button`, `StatusBadge`, `DataTable`, `CopyableCode`, `Input`, `Label`, `Select` (Radix), `FormField`.

## Rationale

- **Design token ownership.** Cuttlegate has a defined design token spec (#200). Third-party frameworks impose their own token model; adapting it to ours creates a two-layer theming system that is harder to maintain than direct Tailwind classes.
- **Dependency surface.** A full component framework is a large, frequently-updated dependency. Cuttlegate's UI surface is small enough that the maintenance burden of a framework outweighs its benefits.
- **Bundle size.** In-house components include only what is used. Tree-shaking a large framework is less reliable than not importing it in the first place.
- **Precedent.** ADR 0015 already established the pattern: use Radix for complex accessibility primitives, build everything else. This ADR formalises what was already practice.

## Consequences

- New UI primitives are written in-house. Before writing a new component, check `web/src/components/ui/` — the primitive may already exist.
- For components with non-trivial keyboard/focus behaviour (combobox, dialog, tooltip, popover), prefer a Radix primitive as the unstyled base rather than implementing ARIA patterns from scratch.
- No shadcn/ui, MUI, Ant Design, or Chakra UI. If a PR introduces one of these as a dependency, it requires a new ADR.
- Design tokens (`web/src/styles.css` — the `@theme {}` block) are the single source of truth for colour, spacing, and typography. Components must use tokens, not hardcoded values.
- Accessibility (WCAG 2.1 AA) is the component author's responsibility. The axe-core integration in the frontend test harness (#137) is the gate.
