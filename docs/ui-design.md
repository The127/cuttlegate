# Cuttlegate — UI Design Standards

This is a living document. Any design decision made in a team session that establishes a pattern or affects multiple screens must be written here before the session closes. 

---

## Visual Identity

### Tokens
Design tokens live in `web/src/styles.css` under `@theme {}`. There is no `tailwind.config.js`. All colour, spacing, and typography decisions are encoded there.

### Colour
- **One accent colour**, used sparingly — for primary actions and active states only. Not decoration.
- **Status colours:** enabled/active (green), disabled/inactive (grey), error/degraded (red), warning (amber). All must pass WCAG 2.1 AA colour-blind contrast checks — test against deuteranopia and protanopia.
- **Background:** near-white, not pure white. Reduces eye strain for operators who live in this dashboard.
- When in doubt: less colour, more contrast.

### Typography
- **UI text:** system font stack — feels native, loads instantly.
- **Monospace:** all technical identifiers render in monospace: flag keys, environment slugs, project slugs, client IDs, API key previews, token strings, evaluation log entries. This is non-negotiable — these values must be scannable and copyable at a glance.

### Density
- Default to showing data. Engineers can read a table. Do not hide information behind extra clicks unless there is a genuine UX reason — not aesthetics, not "cleanliness."
- Information that an operator needs during an incident must be on the surface, not one drill-down away.

---

## Empty States

Empty states are the most critical UX moment in onboarding and the most commonly neglected.

**Every empty state must have three things:**
1. A description — one sentence, active voice, specific to what is missing
2. A primary CTA — a button or link that routes to the obvious next action
3. (Optional) one line of secondary context if the action is not obvious

**Copy rules:**
- Active, not passive: *"Create your first environment"* not *"No environments yet"*
- Specific: name the thing, don't describe the absence
- Do not describe a problem without offering a solution

**Verifiable:** before any frontend issue moves to `in_review`, grep for empty state components — every one must have an action element. A text-only empty state is not done.

### Known empty states and their CTAs

| Screen | Empty condition | Required CTA |
|---|---|---|
| Project page | No environments | "Create your first environment" → environment creation form |
| Project page | No flags | "Create your first flag" → flag creation form |
| Flags list | No flags in environment | "Create a flag" → flag creation form |
| Environments list | No environments | "Create an environment" → environment creation form |
| Segments | No segments | "Create a segment" → segment creation form |
| Audit log | No events yet | No CTA needed — passive state is acceptable here (nothing to act on) |

*(Add rows as new screens ship.)*

---

## User Flow Standards

### The first-run contract
A new operator must be able to go from empty project to first flag evaluating in under three minutes — without reading documentation. Every screen in that path must answer: *what do I do next?*

The path is: **create project → create environment → create flag → copy SDK snippet**

Each step must have a clear forward action. No dead ends.

### Context-sensitive navigation
- The environment dropdown in the nav must not prompt "Select environment..." when the project has no environments. Replace with a nudge toward creating one.
- Quick links (if used) are context-sensitive — only surface actions that are currently meaningful given project state. "Compare Environments" does not appear when there is one or zero environments.

### State transitions
State transitions are design decisions, not implementation defaults. They must be stated explicitly at grooming:
- What happens to the selected environment when the user switches project?
- What does the URL look like for a project-scoped vs. environment-scoped view?
- What happens when an environment is deleted and it was the selected one?

If these are not answered at grooming, the implementer must surface them — not decide silently.

---

## Loading States

- **Initial page load:** skeleton loaders that match the layout of the content. No full-page spinners.
- **Mutations (create, update, delete):** inline spinner on the button. Disable the form while in flight.
- **Background refetch:** silent — no visible indicator unless stale data would cause confusion.

---

## Destructive Actions

Always two-step:
1. Intent: user clicks "Delete"
2. Confirm: modal names the thing being destroyed — not "Are you sure?" but "Delete environment *production*? This cannot be undone."

The confirmation must name the specific thing. Generic confirmations get clicked through without reading.

---

## Error States

### API error shape
All errors: `{ "error": "code", "message": "human readable" }` — no exceptions. Stack traces never reach the UI.

### User-actionable vs. report-only
- **User can act:** show the message and a clear action ("Try again", "Check your permissions", "Contact your admin")
- **User cannot act:** show the message and a way to report ("Something went wrong — if this keeps happening, contact support")
- Never show raw HTTP status codes to end users. The error code in the JSON is for logging; the message is for humans.

### Token expiry
Never show a raw 401. Silent renewal first; if renewal fails, redirect to login. The user is never left staring at an error they cannot act on.

---

## Copy Voice

- Active and direct: tell the user what to do, not what has happened
- Specific: name the thing, not the category
- Trust the user: they are engineers; do not oversimplify
- Error messages are written for the operator who will act on them at 2am — not for a marketing audience

---

## Design Review Gate

Design sign-off is required before any user-facing issue moves to `in_review`. This covers:
- Any screen, empty state, error message, or interaction
- Any API contract shape consumed by the SPA
- Any user flow decision (what happens after an action, what a screen shows when empty)
- Copy for any user-visible string

If something is missing soul — it feels assembled rather than designed, or it leaves the user without a clear next action — it goes back. This is not polish. It is function.

---

## Component Patterns

### TierSelector

A controlled button-group for selecting an API key capability tier. Lives at `web/src/components/ui/TierSelector.tsx`.

**Props:** `value: ToolCapabilityTier`, `onChange: (tier: ToolCapabilityTier) => void`

**Behaviour:**
- Three buttons in order: Read → Write → Destructive
- Caller must default to `"read"` — never `"destructive"` as default
- Active button uses tier-specific colour (Read: accent, Write: blue, Destructive: amber)
- **Destructive uses amber when active, not red.** Red = destructive action (Delete buttons). Amber = destructive capability (this key can delete). The distinction is intentional and must be preserved.
- Inline warning `<p>` in amber text appears only when `value === "destructive"`
- All copy via `useTranslation('projects')` — no hardcoded strings

### TierBadge

A read-only coloured pill for displaying an API key's capability tier. Lives at `web/src/components/ui/TierBadge.tsx`.

**Props:** `tier: ToolCapabilityTier`, `className?: string`

**Colour convention:**

| Tier | Classes |
|---|---|
| `read` | `bg-neutral-100 text-neutral-600 border-neutral-200` (grey) |
| `write` | `bg-blue-50 text-blue-700 border-blue-200` (blue) |
| `destructive` | `bg-amber-50 text-amber-700 border-amber-200` (amber) |

**Rule:** TierBadge is distinct from StatusBadge. Do not merge them — capability tier and flag/environment status are orthogonal concepts with different colour semantics.

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
