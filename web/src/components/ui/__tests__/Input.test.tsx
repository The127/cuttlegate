import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { Input } from '../Input'

describe('Input', () => {
  it('renders an input element', () => {
    render(<Input aria-label="Username" />)
    expect(screen.getByRole('textbox', { name: 'Username' })).toBeInTheDocument()
  })

  it('applies error styles when hasError is true', () => {
    render(<Input aria-label="Slug" hasError />)
    const input = screen.getByRole('textbox', { name: 'Slug' })
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('does not set aria-invalid when hasError is false', () => {
    render(<Input aria-label="Name" />)
    const input = screen.getByRole('textbox', { name: 'Name' })
    expect(input).not.toHaveAttribute('aria-invalid')
  })

  it('respects aria-invalid="true" string without hasError', () => {
    render(<Input aria-label="Field" aria-invalid="true" />)
    const input = screen.getByRole('textbox', { name: 'Field' })
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  // @edge: Input used outside FormField context — no auto-injected id or aria-describedby
  it('renders without error and has no auto-injected id or aria-describedby when outside FormField', () => {
    render(<Input aria-label="Standalone" />)
    const input = screen.getByRole('textbox', { name: 'Standalone' })
    expect(input).not.toHaveAttribute('id')
    expect(input).not.toHaveAttribute('aria-describedby')
  })

  it('passes axe accessibility check — default', async () => {
    const { axeResults } = await renderWithAxe(
      <label>
        Name
        <Input />
      </label>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — with error', async () => {
    const { axeResults } = await renderWithAxe(
      <div>
        <label htmlFor="slug-field">Slug</label>
        <Input id="slug-field" hasError aria-describedby="slug-error" />
        <span id="slug-error">Invalid slug format</span>
      </div>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
