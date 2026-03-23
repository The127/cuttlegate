import { createRef } from 'react'
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { Textarea } from '../Textarea'
import { FormField } from '../FormField'

describe('Textarea', () => {
  it('renders a textarea element', () => {
    render(<Textarea aria-label="Notes" />)
    expect(screen.getByRole('textbox', { name: 'Notes' })).toBeInTheDocument()
  })

  it('applies error styles when hasError is true', () => {
    render(<Textarea aria-label="Notes" hasError />)
    const el = screen.getByRole('textbox', { name: 'Notes' })
    expect(el).toHaveAttribute('aria-invalid', 'true')
  })

  it('does not set aria-invalid when hasError is false', () => {
    render(<Textarea aria-label="Notes" />)
    const el = screen.getByRole('textbox', { name: 'Notes' })
    expect(el).not.toHaveAttribute('aria-invalid')
  })

  it('respects aria-invalid="true" string without hasError', () => {
    render(<Textarea aria-label="Field" aria-invalid="true" />)
    const el = screen.getByRole('textbox', { name: 'Field' })
    expect(el).toHaveAttribute('aria-invalid', 'true')
  })

  // @edge: Textarea used outside FormField context — no auto-injected id or aria-describedby
  it('renders without auto-injected id or aria-describedby when outside FormField', () => {
    render(<Textarea aria-label="Standalone" />)
    const el = screen.getByRole('textbox', { name: 'Standalone' })
    expect(el).not.toHaveAttribute('id')
    expect(el).not.toHaveAttribute('aria-describedby')
  })

  // @edge: ref forwarding
  it('forwards ref to the underlying textarea element', () => {
    const ref = createRef<HTMLTextAreaElement>()
    render(<Textarea ref={ref} aria-label="Ref test" />)
    expect(ref.current).toBeInstanceOf(HTMLTextAreaElement)
  })

  // @edge: className merge
  it('merges custom className with design system classes', () => {
    render(<Textarea aria-label="Custom" className="font-mono" />)
    const el = screen.getByRole('textbox', { name: 'Custom' })
    expect(el.className).toContain('font-mono')
    expect(el.className).toContain('rounded')
  })

  // @happy: FormField auto-wiring
  it('reads fieldId from FormField context', () => {
    render(
      <FormField label="Description">
        <Textarea />
      </FormField>,
    )
    const el = screen.getByRole('textbox', { name: 'Description' })
    const label = screen.getByText('Description')
    expect(el).toHaveAttribute('id')
    expect(label).toHaveAttribute('for', el.id)
  })

  // @happy: FormField auto-wiring with error
  it('reads errorId from FormField context when error is present', () => {
    render(
      <FormField label="Description" error="Required">
        <Textarea />
      </FormField>,
    )
    const el = screen.getByRole('textbox', { name: 'Description' })
    expect(el).toHaveAttribute('aria-describedby')
    const errorEl = document.getElementById(el.getAttribute('aria-describedby')!)
    expect(errorEl).toHaveTextContent('Required')
  })

  // @edge: explicit id overrides FormField auto-wiring
  it('uses explicit id prop over FormField context', () => {
    render(
      <FormField label="Notes">
        <Textarea id="custom-id" />
      </FormField>,
    )
    const el = screen.getByRole('textbox')
    expect(el).toHaveAttribute('id', 'custom-id')
  })

  it('passes axe accessibility check — default', async () => {
    const { axeResults } = await renderWithAxe(
      <label>
        Notes
        <Textarea />
      </label>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — with error', async () => {
    const { axeResults } = await renderWithAxe(
      <div>
        <label htmlFor="notes-field">Notes</label>
        <Textarea id="notes-field" hasError aria-describedby="notes-error" />
        <span id="notes-error">This field is required</span>
      </div>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
