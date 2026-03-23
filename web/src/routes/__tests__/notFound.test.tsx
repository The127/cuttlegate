import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { axe } from 'jest-axe'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    useLocation: () => ({ pathname: '/nonexistent-page' }),
    createRootRoute: (opts: any) => ({ ...opts, options: opts }),
  }
})

async function renderNotFoundPage() {
  const mod = await import('../__root')
  const { NotFoundPage } = mod
  return render(<NotFoundPage />)
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  vi.resetModules()
})

describe('NotFoundPage (root-level)', () => {
  it('renders 404 heading', async () => {
    await renderNotFoundPage()
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('404')
  })

  it('includes the requested URL in the body text', async () => {
    await renderNotFoundPage()
    expect(screen.getByText(/\/nonexistent-page/)).toBeInTheDocument()
  })

  it('renders a "Return to home" link pointing to "/"', async () => {
    await renderNotFoundPage()
    const link = screen.getByRole('link', { name: /return to home/i })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', '/')
  })

  it('link has non-empty accessible text', async () => {
    await renderNotFoundPage()
    const link = screen.getByRole('link', { name: /return to home/i })
    expect(link.textContent?.trim()).not.toBe('')
  })

  it('sets document.title to reflect not-found state', async () => {
    await renderNotFoundPage()
    expect(document.title).toMatch(/404/)
  })

  it('has no accessibility violations', async () => {
    const { container } = await renderNotFoundPage()
    const results = await axe(container)
    expect(results.violations).toHaveLength(0)
  })
})
