import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { FormField } from '../FormField'

describe('FormField', () => {
  it('renders label and children', () => {
    render(
      <FormField label="Project Name" htmlFor="name">
        <input id="name" />
      </FormField>,
    )
    expect(screen.getByText('Project Name')).toBeInTheDocument()
    expect(screen.getByLabelText('Project Name')).toBeInTheDocument()
  })

  it('renders error message when error prop is set', () => {
    render(
      <FormField label="Slug" htmlFor="slug" error="Invalid slug format">
        <input id="slug" />
      </FormField>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('Invalid slug format')
  })

  it('does not render error element when no error', () => {
    render(
      <FormField label="Name" htmlFor="name">
        <input id="name" />
      </FormField>,
    )
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('passes axe accessibility check — no error', async () => {
    const { axeResults } = await renderWithAxe(
      <FormField label="Email" htmlFor="email">
        <input id="email" type="email" />
      </FormField>,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — with error', async () => {
    const { axeResults } = await renderWithAxe(
      <FormField label="Slug" htmlFor="slug" error="Must be lowercase">
        <input id="slug" aria-invalid="true" />
      </FormField>,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
