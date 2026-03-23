import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { FormField } from '../FormField'
import { Input } from '../Input'
import { Textarea } from '../Textarea'

describe('FormField', () => {
  // @happy: auto-wire label to Input without explicit ids
  it('auto-wires label to Input without manual htmlFor or id', () => {
    render(
      <FormField label="Name">
        <Input />
      </FormField>,
    )
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
  })

  // @happy: auto-wire aria-describedby on Input when error is present
  it('auto-wires aria-describedby on Input when error prop is set', () => {
    render(
      <FormField label="Email" error="Invalid format">
        <Input />
      </FormField>,
    )
    const input = screen.getByLabelText('Email')
    const errorEl = screen.getByRole('alert')
    expect(errorEl).toHaveTextContent('Invalid format')
    expect(input).toHaveAttribute('aria-describedby', errorEl.id)
  })

  // @happy: explicit htmlFor overrides auto-generated id
  it('uses explicit htmlFor when provided, caller-supplied id wins', () => {
    render(
      <FormField label="Slug" htmlFor="custom-id">
        <Input id="custom-id" />
      </FormField>,
    )
    const label = screen.getByText('Slug').closest('label')
    expect(label).toHaveAttribute('for', 'custom-id')
    const input = screen.getByLabelText('Slug')
    expect(input).toHaveAttribute('id', 'custom-id')
  })

  it('renders error message when error prop is set', () => {
    render(
      <FormField label="Slug" error="Invalid slug format">
        <Input />
      </FormField>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('Invalid slug format')
  })

  it('does not render error element when no error', () => {
    render(
      <FormField label="Name">
        <Input />
      </FormField>,
    )
    expect(screen.queryByRole('alert')).toBeNull()
  })

  // @happy: axe passes with auto-wired Input (no manual ids)
  it('passes axe check — auto-wired Input, no error', async () => {
    const { axeResults } = await renderWithAxe(
      <FormField label="Name">
        <Input />
      </FormField>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  // @happy: axe passes with auto-wired Input and error
  it('passes axe check — auto-wired Input with error', async () => {
    const { axeResults } = await renderWithAxe(
      <FormField label="Email" error="Invalid format">
        <Input hasError />
      </FormField>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  // @happy: FormField wrapping Textarea — auto-wiring works the same as Input
  it('passes axe check — Textarea auto-wired via FormField context', async () => {
    const { axeResults } = await renderWithAxe(
      <FormField label="Notes">
        <Textarea />
      </FormField>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
