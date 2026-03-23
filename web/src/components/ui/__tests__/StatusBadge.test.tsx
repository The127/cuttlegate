import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { StatusBadge } from '../StatusBadge'

afterEach(() => {
  vi.restoreAllMocks()
})

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

  // @happy — dark is the default theme; axe structural check
  it('passes axe in dark mode — enabled', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="enabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — disabled', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="disabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — warning', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="warning" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — unknown', async () => {
    const { axeResults } = await renderWithAxe(<StatusBadge status="unknown" />)
    expect(axeResults).toHaveNoViolations()
  })

  // @edge — M9 design token rgba() pill classes are present in the DOM
  it('enabled status pill has design token classes', () => {
    const { container } = render(<StatusBadge status="enabled" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('rgba(16,217,168,0.12)')
    expect(pill?.className).toContain('color-status-enabled')
  })

  it('disabled status pill has design token classes', () => {
    const { container } = render(<StatusBadge status="disabled" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('rgba(248,113,113,0.12)')
    expect(pill?.className).toContain('color-status-error')
  })

  it('warning status pill has design token classes', () => {
    const { container } = render(<StatusBadge status="warning" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('rgba(251,191,36,0.12)')
    expect(pill?.className).toContain('color-status-warning')
  })

  it('unknown status pill has design token classes', () => {
    const { container } = render(<StatusBadge status="unknown" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('color-surface-elevated')
    expect(pill?.className).toContain('color-text-secondary')
  })
})
