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
    const variants = ['primary', 'secondary', 'ghost', 'danger', 'danger-outline'] as const
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
