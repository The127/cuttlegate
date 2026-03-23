import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { StatusBadge } from '../StatusBadge'

// Helper: override matchMedia to simulate dark mode preference.
// jsdom does not apply CSS media queries, so dark: classes are present in the DOM
// regardless — this ensures the matchMedia API doesn't throw in dark-mode renders.
function mockDarkMode() {
  vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
    matches: query === '(prefers-color-scheme: dark)',
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }))
}

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

  // @happy — dark mode: axe passes with dark: class variants applied
  // Note: jsdom does not apply CSS media queries; dark: classes are present in DOM
  // but not computed as styles. These tests verify structural accessibility in dark mode
  // (ARIA, roles, labels) — not computed colour contrast.
  it('passes axe in dark mode — enabled', async () => {
    mockDarkMode()
    const { axeResults } = await renderWithAxe(<StatusBadge status="enabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — disabled', async () => {
    mockDarkMode()
    const { axeResults } = await renderWithAxe(<StatusBadge status="disabled" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — warning', async () => {
    mockDarkMode()
    const { axeResults } = await renderWithAxe(<StatusBadge status="warning" />)
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe in dark mode — unknown', async () => {
    mockDarkMode()
    const { axeResults } = await renderWithAxe(<StatusBadge status="unknown" />)
    expect(axeResults).toHaveNoViolations()
  })

  // @edge — dark: variant classes are present in the DOM for all statuses
  it('enabled status pill has dark: variant classes', () => {
    const { container } = render(<StatusBadge status="enabled" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('dark:bg-green-950')
    expect(pill?.className).toContain('dark:text-green-300')
  })

  it('disabled status pill has dark: variant classes', () => {
    const { container } = render(<StatusBadge status="disabled" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('dark:bg-red-950')
    expect(pill?.className).toContain('dark:text-red-300')
  })

  it('warning status pill has dark: variant classes', () => {
    const { container } = render(<StatusBadge status="warning" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('dark:bg-amber-950')
    expect(pill?.className).toContain('dark:text-amber-300')
  })

  it('unknown status pill has dark: variant classes', () => {
    const { container } = render(<StatusBadge status="unknown" />)
    const pill = container.querySelector('span')
    expect(pill?.className).toContain('dark:bg-gray-800')
    expect(pill?.className).toContain('dark:text-gray-400')
  })
})
