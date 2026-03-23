# Cuttlegate — UI Design Standards

This is a living document. Any design decision made in a team session that establishes a pattern or affects multiple screens must be written here before the session closes. 

---

## Visual Identity

### Design direction

Cuttlegate's visual language is **dark, precise, and alive**. Deep navy backgrounds. Teal-to-blue gradient accents on interactive elements. Elevated card surfaces with barely-visible borders. JetBrains Mono for every technical identifier. High contrast text on a surface that doesn't strain the eyes of someone who lives in this dashboard.

This is not a light app with dark mode bolted on. Dark is the product.

Reference aesthetic: the dashboard panels in the M9 design references — dense, structured, gradient accents used with restraint, no decorative chrome.

---

### Design tokens

All tokens live in `web/src/styles.css` under `@theme {}`. There is no `tailwind.config.js`. Every colour, spacing, and typography value is encoded there.

#### Colour — backgrounds and surfaces

```
--color-bg:               #0d0f1c   /* page background — deep navy */
--color-surface:          #141729   /* card / panel surface */
--color-surface-elevated: #1c1f35   /* hover state, dropdown items, active nav bg */
--color-border:           rgba(255,255,255,0.07)   /* default border — barely there */
--color-border-hover:     rgba(255,255,255,0.14)   /* hovered / focused border */
```

#### Colour — accent (gradient)

```
--color-accent-start: #00d4aa   /* teal */
--color-accent-end:   #4f7cff   /* blue-purple */
--color-accent:       #4f7cff   /* single-colour fallback for rings, icons */
```

**Gradient usage:** `background: linear-gradient(135deg, var(--color-accent-start), var(--color-accent-end))`

Use the gradient on: primary buttons, active sidebar item indicator, active tab underlines, focus glow. Use `--color-accent` (single colour) on: focus rings, icon highlights, link text.

#### Colour — text

```
--color-text-primary:   #e8eaf6   /* headings, body — warm near-white */
--color-text-secondary: #7b83a8   /* labels, descriptions — blue-gray */
--color-text-muted:     #4a5070   /* hints, timestamps, placeholders */
```

#### Colour — status

```
--color-status-enabled:  #10d9a8   /* teal-emerald — flag enabled / healthy */
--color-status-warning:  #fbbf24   /* amber — partial rollout / caution */
--color-status-error:    #f87171   /* red — error / disabled */
```

Status colours are used with a low-opacity tint background. Example for enabled:
`background: rgba(16,217,168,0.12); color: #10d9a8; border: 1px solid rgba(16,217,168,0.25)`

#### Typography scale

```
--text-xs:   0.75rem  / 1.125rem line-height
--text-sm:   0.875rem / 1.375rem
--text-base: 1rem     / 1.625rem
--text-lg:   1.125rem / 1.75rem
--text-xl:   1.25rem  / 1.875rem
--text-2xl:  1.5rem   / 2rem
```

**UI text:** system font stack — feels native, loads instantly.
**Monospace:** JetBrains Mono — loaded via `@font-face` from `web/public/fonts/`. All technical identifiers render in JetBrains Mono: flag keys, environment slugs, project slugs, client IDs, API key previews, token strings, evaluation log entries. This is non-negotiable — these values must be scannable and copyable at a glance.

#### Border radius

```
--radius-sm:  6px    /* inline elements, small badges */
--radius-md:  10px   /* buttons, inputs, small cards */
--radius-lg:  14px   /* panels, modals, large cards */
--radius-xl:  20px   /* full-bleed hero elements */
```

---

### Density

Default to showing data. Engineers can read a table. Do not hide information behind extra clicks unless there is a genuine UX reason — not aesthetics, not "cleanliness."

Information that an operator needs during an incident must be on the surface, not one drill-down away.

---

### Forbidden patterns

These patterns are banned in all new and modified code. Existing usages are tracked in #298–#302 for removal.

| Pattern | Reason | Replace with |
|---|---|---|
| `bg-white` | Light theme remnant | `bg-[var(--color-surface)]` or `bg-[var(--color-bg)]` |
| `bg-gray-*` | Light theme remnant | `bg-[var(--color-surface)]` or `bg-[var(--color-surface-elevated)]` |
| `text-gray-*` | Light theme remnant | `text-[var(--color-text-primary)]`, `text-[var(--color-text-secondary)]`, `text-[var(--color-text-muted)]` |
| `border-gray-*` | Light theme remnant | `border-[var(--color-border)]` or `border-[var(--color-border-hover)]` |
| `dark:` prefixed classes | Dark is now the default; no variant needed | Remove the `dark:` prefix; use the base class |
| `uppercase tracking-wide` on section headings | Dated pattern; removes character from the design | Title case, `text-[var(--color-text-secondary)]`, weight `font-medium` |
| Raw `<input>`, `<textarea>`, `<select>` in route files | Bypasses design system | `Input`, `Select` from `web/src/components/ui/` |

If a PR adds any of the forbidden patterns in new code, the reviewer blocks it. No exceptions.

---

### Component visual specifications

#### Button

- **Primary:** gradient fill (`--color-accent-start` → `--color-accent-end`), white text, `border-radius: var(--radius-md)`. Hover: subtle glow `box-shadow: 0 0 16px rgba(0,212,170,0.25)`.
- **Secondary:** `--color-surface-elevated` background, `--color-text-primary` text, `--color-border` border. Hover: `--color-border-hover`.
- **Destructive:** solid `#f87171` background, white text. This is a serious action — no gradient, no subtlety.
- **Danger-outline:** `--color-border` base border shifts to `#f87171` on hover, `#f87171` text.
- Loading state: spinner or shimmer — the gradient button should not go flat while loading.

#### Input / FormField

- Background: `--color-surface-elevated`
- Border: `--color-border` at rest; glows accent on focus: `box-shadow: 0 0 0 2px rgba(79,124,255,0.35)`
- Text: `--color-text-primary`; placeholder: `--color-text-muted`
- Error state: `#f87171` border + red glow
- Label: title case, `--color-text-secondary`, `font-medium text-sm`

#### StatusBadge

Pill shape with a coloured dot on the left.
- Background: 12% opacity tint of the status colour
- Border: 25% opacity tint of the status colour
- Text: the status colour at full opacity
- Example (enabled): `background: rgba(16,217,168,0.12); border: 1px solid rgba(16,217,168,0.25); color: #10d9a8`

#### TierSelector

A controlled button-group for selecting an API key capability tier. Lives at `web/src/components/ui/TierSelector.tsx`.

**Props:** `value: ToolCapabilityTier`, `onChange: (tier: ToolCapabilityTier) => void`

- Three buttons in order: Read → Write → Destructive
- Caller must default to `"read"` — never `"destructive"` as default
- Active button: gradient fill (same as primary button)
- **Destructive uses amber when active, not red.** Red = destructive action (Delete buttons). Amber = destructive capability (this key can delete). The distinction is intentional and must be preserved.
- Inactive buttons: secondary button treatment
- Inline warning `<p>` in amber text appears only when `value === "destructive"`
- All copy via `useTranslation('projects')` — no hardcoded strings

#### TierBadge

A read-only coloured pill for displaying an API key's capability tier. Lives at `web/src/components/ui/TierBadge.tsx`.

**Props:** `tier: ToolCapabilityTier`, `className?: string`

Dark-theme colour convention:

| Tier | Treatment |
|---|---|
| `read` | `rgba(255,255,255,0.06)` bg, `--color-text-secondary` text, `--color-border` border |
| `write` | `rgba(79,124,255,0.15)` bg, `#818cf8` text, `rgba(79,124,255,0.3)` border |
| `destructive` | `rgba(251,191,36,0.12)` bg, `#fbbf24` text, `rgba(251,191,36,0.25)` border |

**Rule:** TierBadge is distinct from StatusBadge. Do not merge them — capability tier and flag/environment status are orthogonal concepts.

#### Dialog

- Background: `--color-surface`, `backdrop-filter: blur(12px)`
- Border: `--color-border-hover`
- Overlay: `rgba(0,0,0,0.65)`
- `border-radius: var(--radius-lg)`
- Shadow: `0 24px 48px rgba(0,0,0,0.5)`

#### DataTable

- Header: `--color-surface-elevated`, `--color-text-secondary`, title case
- Row base: `--color-surface`; hover: `--color-surface-elevated`
- Row dividers: `--color-border`
- No outer shadow on the table container

#### Card pattern

Used for project cards, environment cards, API key cards, and any entity displayed as a tile.

```
background:    var(--color-surface)
border:        1px solid var(--color-border)
border-radius: var(--radius-lg)
```

Hover state: `border-color: var(--color-border-hover)` + `box-shadow: 0 4px 20px rgba(0,0,0,0.3)`

Do not add inner padding inconsistently. Standard card padding: `p-4` (16px).

#### Sidebar navigation

- Background: `--color-surface`, right border `--color-border`
- Nav item rest: `--color-text-secondary`, `px-3 py-2`, `border-radius: var(--radius-sm)`
- Nav item hover: `--color-surface-elevated` background
- **Active nav item:** 3px gradient left border (`--color-accent-start` → `--color-accent-end`) + `--color-surface-elevated` background + `--color-text-primary` text
- Nav item icon: 16px Lucide icon, left of label, same colour as text

---

## Empty States

Empty states are the most critical UX moment in onboarding and the most commonly neglected.

**Every empty state must have three things:**
1. A description — one sentence, active voice, specific to what is missing
2. A primary CTA — gradient primary button that routes to the obvious next action
3. (Optional) one line of secondary context if the action is not obvious

**Copy rules:**
- Active, not passive: *"Create your first environment"* not *"No environments yet"*
- Specific: name the thing, don't describe the absence
- Do not describe a problem without offering a solution

**Verifiable:** before any frontend issue moves to `in_review`, grep for empty state components — every one must have an action element (a gradient primary button or a link styled as one). A text-only empty state is not done.

### Known empty states and their CTAs

| Screen | Empty condition | Required CTA |
|---|---|---|
| Home | No projects | "Create your first project" → CreateProjectDialog |
| Project dashboard | No environments | "Create your first environment" → CreateEnvironmentDialog |
| Project dashboard | No flags | "Create your first flag" → flag creation form |
| Flags list | No flags in environment | "Create a flag" → flag creation form |
| Environments list | No environments | "Create an environment" → CreateEnvironmentDialog |
| Segments | No segments | "Create a segment" → segment creation form |
| Audit log | No events yet | No CTA needed — passive state is acceptable here |

*(Add rows as new screens ship.)*

---

## User Flow Standards

### The first-run contract

A new operator must be able to go from empty project to first flag evaluating in under three minutes — without reading documentation. Every screen in that path must answer: *what do I do next?*

The path is: **create project → create environment → create flag → copy SDK snippet**

Each step must have a clear forward action. No dead ends. Environment creation must be accessible directly from the project dashboard (not buried in Settings).

### Context-sensitive navigation

- The environment switcher in the top bar must not show "Select environment..." when the project has no environments. Replace with a nudge: "No environments — Create one →" linking to environment creation.
- Navigation items are context-sensitive — only surface actions meaningful to the current project state. "Compare Environments" does not appear when there is one or zero environments.

### State transitions

State transitions are design decisions, not implementation defaults. They must be stated explicitly at grooming:
- What happens to the selected environment when the user switches project?
- What does the URL look like for a project-scoped vs. environment-scoped view?
- What happens when an environment is deleted and it was the selected one?

If these are not answered at grooming, the implementer must surface them — not decide silently.

---

## Loading States

- **Initial page load:** skeleton loaders that match the layout of the content. Skeleton backgrounds use `--color-surface-elevated` with a pulse animation. No full-page spinners.
- **Mutations (create, update, delete):** inline spinner on the button. Disable the form while in flight. The gradient button should shimmer or show a spinner — not go flat.
- **Background refetch:** silent — no visible indicator unless stale data would cause confusion.

---

## Destructive Actions

Always two-step:
1. Intent: user clicks "Delete"
2. Confirm: modal names the thing being destroyed — not "Are you sure?" but "Delete environment *production*? This cannot be undone."

The confirmation must name the specific thing. Generic confirmations get clicked through without reading.

Destructive confirm button: solid `#f87171` (red), not gradient. Red is reserved for destructive actions. Amber is reserved for destructive capability (TierSelector). These are distinct.

---

## Error States

### API error shape

All errors: `{ "error": "code", "message": "human readable" }` — no exceptions. Stack traces never reach the UI.

### User-actionable vs. report-only

- **User can act:** show the message and a clear action ("Try again", "Check your permissions", "Contact your admin")
- **User cannot act:** show the message and a way to report ("Something went wrong — if this keeps happening, contact support")
- Never show raw HTTP status codes to end users.

### Token expiry

Never show a raw 401. Silent renewal first; if renewal fails, redirect to login. The user is never left staring at an error they cannot act on.

---

## Copy Voice

- Active and direct: tell the user what to do, not what has happened
- Specific: name the thing, not the category
- Trust the user: they are engineers; do not oversimplify
- Error messages are written for the operator who will act on them at 2am — not for a marketing audience
- Section headings: title case. Not `UPPERCASE TRACKING-WIDE`. Not sentence case for labels either — title case for headings, sentence case for body copy and descriptions.

---

## Design Review Gate

Design sign-off is required before any user-facing issue moves to `in_review`. This covers:
- Any screen, empty state, error message, or interaction
- Any API contract shape consumed by the SPA
- Any user flow decision (what happens after an action, what a screen shows when empty)
- Copy for any user-visible string

**The forbidden-patterns check is part of every review.** If a PR introduces `bg-white`, `bg-gray-*`, `dark:` classes, or `uppercase tracking-wide` headings in new code, it goes back. No exceptions.

If something is missing soul — it feels assembled rather than designed, or it leaves the user without a clear next action — it goes back. This is not polish. It is function.

---

## Changelog

| Date | Decision | Session context |
|---|---|---|
| 2026-03-23 | Empty states must have CTAs; passive empty states are not done | Design review session with owner |
| 2026-03-23 | Context-sensitive nav: no "Select environment..." when none exist | Design review session with owner |
| 2026-03-23 | First-run contract: project → environment → flag in under 3 minutes, no docs | Design review session with owner |
| 2026-03-23 | "Beautiful, not just functional" — visual craft is a product requirement, not polish | Owner directive |
| 2026-03-23 | TierSelector: amber for destructive capability (not red); inline warning on destructive | Sprint 21 #278 |
| 2026-03-23 | TierBadge: grey/blue/amber for read/write/destructive — separate from StatusBadge | Sprint 21 #278 |
| 2026-03-23 | Full visual overhaul: dark-by-default, deep navy + teal-blue gradient accent language | M9 design session with owner — replaces all previous light-theme token decisions |
| 2026-03-23 | Forbidden patterns list established: bg-white, bg-gray-*, dark: classes, uppercase tracking-wide banned | M9 design session |
| 2026-03-23 | Section headings: title case only — uppercase tracking-wide pattern retired project-wide | M9 design session |
| 2026-03-23 | Destructive confirm button: solid red only, not gradient — red reserved for destructive actions | M9 design session |
