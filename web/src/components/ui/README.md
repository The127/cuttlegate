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
