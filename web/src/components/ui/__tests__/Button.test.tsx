import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { Button } from '../Button'

describe('Button', () => {
  it('renders with default variant and size', () => {
    render(<Button>Click me</Button>)
    const btn = screen.getByRole('button', { name: 'Click me' })
    expect(btn).toBeInTheDocument()
  })

  it('renders all variants without throwing', () => {
    const variants = ['primary', 'secondary', 'ghost', 'danger', 'danger-outline', 'destructive'] as const
    for (const variant of variants) {
      render(<Button variant={variant}>{variant}</Button>)
      expect(screen.getByRole('button', { name: variant })).toBeInTheDocument()
    }
  })

  it('renders all sizes without throwing', () => {
    const sizes = ['sm', 'md', 'lg'] as const
    for (const size of sizes) {
      render(<Button size={size}>{size}</Button>)
      expect(screen.getByRole('button', { name: size })).toBeInTheDocument()
    }
  })

  it('forwards disabled prop', () => {
    render(<Button disabled>Disabled</Button>)
    expect(screen.getByRole('button', { name: 'Disabled' })).toBeDisabled()
  })

  // @happy — loading state disables button and shows spinner
  it('is disabled and aria-busy when loading', () => {
    render(<Button loading>Save</Button>)
    const btn = screen.getByRole('button', { name: 'Save' })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('aria-busy', 'true')
  })

  it('renders spinner svg when loading', () => {
    const { container } = render(<Button loading>Save</Button>)
    const spinner = container.querySelector('svg.animate-spin')
    expect(spinner).toBeInTheDocument()
  })

  it('does not render spinner when not loading', () => {
    const { container } = render(<Button>Save</Button>)
    expect(container.querySelector('svg.animate-spin')).not.toBeInTheDocument()
  })

  it('passes axe accessibility check — loading', async () => {
    const { axeResults } = await renderWithAxe(<Button loading>Saving</Button>)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — destructive', async () => {
    const { axeResults } = await renderWithAxe(<Button variant="destructive">Delete</Button>)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — primary', async () => {
    const { axeResults } = await renderWithAxe(<Button variant="primary">Save</Button>)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — danger', async () => {
    const { axeResults } = await renderWithAxe(<Button variant="danger">Delete</Button>)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe accessibility check — disabled', async () => {
    const { axeResults } = await renderWithAxe(<Button disabled>Unavailable</Button>)
    expect(axeResults).toHaveNoViolations()
  })
})
