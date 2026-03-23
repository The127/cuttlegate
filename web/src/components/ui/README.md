# Shared UI Primitives

Shared interactive element components for the Cuttlegate SPA.

## Rule

| Situation | Use |
|---|---|
| Button, submit, or action element | `<Button>` |
| Text, number, or similar input | `<Input>` |
| Form field label | `<Label>` |
| Dropdown / option selector (non-navigation) | `<Select>` + `<SelectItem>` |
| Label + input + optional error as a unit | `<FormField>` |
| One-off layout (padding, flex, grid, spacing) | Inline Tailwind is fine |

**Do not** add raw `<button>`, `<input>`, `<label>`, or `<select>` elements in feature code except:
- Navigation context selects (e.g. `ProjectSwitcher` — native `<select>` is acceptable there)
- Textarea elements (not extracted — use native `<textarea>` with Tailwind directly)
- Read-only display elements that happen to use a tag name (e.g. `<span>`)
- Inline text-link style triggers that are not form action buttons (e.g. `+ Add condition` in rule editors — these are micro-interactions, not submit/save/delete actions)

## FormField + Input auto-wiring

When `<Input>` is a direct or nested child of `<FormField>`, the label-to-input association is
handled automatically — no `htmlFor` or `id` props are needed:

```tsx
// Correct — no manual id wiring required
<FormField label="Project name">
  <Input value={name} onChange={...} />
</FormField>

// Also correct — explicit id overrides auto-generated one
<FormField label="Slug" htmlFor="project-slug">
  <Input id="project-slug" ref={slugRef} />
</FormField>
```

`FormField` passes a generated `fieldId` and `errorId` via React context. `Input` reads the
context and applies `id` and `aria-describedby` only when not already supplied by the caller.

For non-`Input` children (e.g. `<textarea>`), manual wiring is still required:

```tsx
<FormField label="Notes" htmlFor="notes">
  <textarea id="notes" />
</FormField>
```

## Accent colour

The `Button` (primary variant) and all focus rings use `--color-accent` (default: `#2563eb`).
This is set in `src/styles.css` and overridden at runtime by the brandability configuration.
Any new component that needs brand-consistent colour must use `var(--color-accent)`, not a hardcoded Tailwind colour class.

## Navigation selects (ProjectSwitcher exception)

`ProjectSwitcher` uses native `<select>` elements for the project and environment switchers.
This is intentional: these are navigation controls embedded in a toolbar, not form controls.
Native select is acceptable and preferred in that context.

## ADR

See `docs/adr/0015-radix-select-as-only-radix-primitive.md` for the decision record on why
`@radix-ui/react-select` is the only Radix package used.

| Modal / dialog | `<Dialog>` + `<DialogContent>` + `<DialogTitle>` (and optional sub-parts) |

`Dialog` is backed by `@radix-ui/react-dialog` (added Sprint 15, #242) — it provides focus trapping, Escape-to-close, overlay-click-to-close, and correct ARIA roles without custom event listeners. `PromoteDialog` and `CreateProjectDialog` use it internally. ADR 0015 anticipated this addition.
