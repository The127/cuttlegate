import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { StatusBadge } from '../StatusBadge'

describe('StatusBadge', () => {
  it('renders enabled status with default label', () => {
    render(<StatusBadge status="enabled" />)
    expect(screen.getByText('Enabled')).toBeInTheDocument()
  })

  it('renders disabled status with default label', () => {
    render(<StatusBadge status="disabled" />)
    expect(screen.getByText('Disabled')).toBeInTheDocument()
  })

  it('renders warning status with default label', () => {
    render(<StatusBadge status="warning" />)
    expect(screen.getByText('Warning')).toBeInTheDocument()
  })

  it('renders unknown status with default label', () => {
    render(<StatusBadge status="unknown" />)
    expect(screen.getByText('Unknown')).toBeInTheDocument()
  })

  it('renders custom label when provided', () => {
    render(<StatusBadge status="enabled" label="Live" />)
    expect(screen.getByText('Live')).toBeInTheDocument()
    expect(screen.queryByText('Enabled')).not.toBeInTheDocument()
  })

  it('dot is aria-hidden', () => {
    const { container } = render(<StatusBadge status="enabled" />)
    const dot = container.querySelector('[aria-hidden="true"]')
    expect(dot).toBeInTheDocument()
  })

  it('disabled status uses error token class on dot', () => {
    const { container } = render(<StatusBadge status="disabled" />)
    const dot = container.querySelector('[aria-hidden="true"]')
    expect(dot?.className).toContain('color-status-error')
  })

  it('enabled status uses enabled token class on dot', () => {
    const { container } = render(<StatusBadge status="enabled" />)
    const dot = container.querySelector('[aria-hidden="true"]')
    expect(dot?.className).toContain('color-status-enabled')
  })

  it('warning status uses warning token class on dot', () => {
    const { container } = render(<StatusBadge status="warning" />)
    const dot = container.querySelector('[aria-hidden="true"]')
    expect(dot?.className).toContain('color-status-warning')
  })

  // @happy — all statuses pass axe
  it('passes axe — enabled', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="enabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe — disabled', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="disabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe — warning', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="warning" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe — unknown', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="unknown" />)
    expect(axeResults).toHaveNoViolations()
  })
})
