import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { Label } from '../Label'

describe('Label', () => {
  it('renders a label element', () => {
    render(<Label>Project Name</Label>)
    expect(screen.getByText('Project Name').tagName).toBe('LABEL')
  })

  it('associates with an input via htmlFor', () => {
    render(
      <div>
        <Label htmlFor="name-field">Name</Label>
        <input id="name-field" />
      </div>,
    )
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
  })

  it('passes axe accessibility check', async () => {
    const { axeResults } = await renderWithAxe(
      <div>
        <Label htmlFor="email">Email</Label>
        <input id="email" type="email" />
      </div>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
