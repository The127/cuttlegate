import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { Select, SelectItem } from '../Select'

describe('Select', () => {
  it('renders the trigger button', () => {
    render(
      <Select value="" onValueChange={vi.fn()} placeholder="Choose one">
        <SelectItem value="a">Option A</SelectItem>
      </Select>,
    )
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('shows placeholder when no value selected', () => {
    render(
      <Select value="" onValueChange={vi.fn()} placeholder="Pick a variant">
        <SelectItem value="on">On</SelectItem>
      </Select>,
    )
    expect(screen.getByText('Pick a variant')).toBeInTheDocument()
  })

  it('passes axe accessibility check — with aria-label', async () => {
    const { axeResults } = await renderWithAxe(
      <Select value="" onValueChange={vi.fn()} placeholder="Choose variant" aria-label="Default variant">
        <SelectItem value="on">On</SelectItem>
        <SelectItem value="off">Off</SelectItem>
      </Select>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — with selected value', async () => {
    const { axeResults } = await renderWithAxe(
      <Select value="eq" onValueChange={vi.fn()} aria-label="Operator">
        <SelectItem value="eq">equals</SelectItem>
        <SelectItem value="neq">not equals</SelectItem>
      </Select>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
