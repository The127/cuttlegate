import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

// Mock TanStack Router — control useMatches return value
const mockMatches: any[] = []

vi.mock('@tanstack/react-router', () => ({
  useMatches: () => mockMatches,
  Link: ({ children, to, ...props }: any) => <a href={to} {...props}>{children}</a>,
}))

import { Breadcrumbs } from '../Breadcrumbs'

function setMatches(matches: Array<{ routeId: string; params?: any; loaderData?: any }>) {
  mockMatches.length = 0
  mockMatches.push(...matches)
}

describe('Breadcrumbs', () => {
  it('renders nothing on the home page', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/' },
    ])
    const { container } = render(<Breadcrumbs />)
    expect(container.querySelector('nav')).toBeNull()
  })

  it('renders Home > Project Name on project page', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
    ])
    render(<Breadcrumbs />)

    expect(screen.getByText('Home')).toBeInTheDocument()
    expect(screen.getByText('My Project')).toBeInTheDocument()
  })

  it('uses slug as fallback when loader data is not available', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: undefined },
    ])
    render(<Breadcrumbs />)

    expect(screen.getByText('my-proj')).toBeInTheDocument()
  })

  it('renders full path on rules page', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug', params: { slug: 'my-proj', envSlug: 'staging' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug/flags/$key', params: { slug: 'my-proj', envSlug: 'staging', key: 'dark-mode' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug/flags/$key/rules' },
    ])
    render(<Breadcrumbs />)

    expect(screen.getByText('Home')).toBeInTheDocument()
    expect(screen.getByText('My Project')).toBeInTheDocument()
    expect(screen.getByText('staging')).toBeInTheDocument()
    expect(screen.getByText('dark-mode')).toBeInTheDocument()
    expect(screen.getByText('Rules')).toBeInTheDocument()
  })

  it('last segment is plain text, not a link', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
    ])
    render(<Breadcrumbs />)

    // "My Project" is the last segment — should be a span, not a link
    const lastSegment = screen.getByText('My Project')
    expect(lastSegment.tagName).toBe('SPAN')
    expect(lastSegment).toHaveClass('font-medium')

    // "Home" should be a link
    const homeLink = screen.getByText('Home')
    expect(homeLink.tagName).toBe('A')
  })

  it('all segments except last are links', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug', params: { slug: 'my-proj', envSlug: 'staging' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug/flags/$key', params: { slug: 'my-proj', envSlug: 'staging', key: 'dark-mode' } },
      { routeId: '/_authenticated/projects/$slug/environments/$envSlug/flags/$key/rules' },
    ])
    render(<Breadcrumbs />)

    expect(screen.getByText('Home').tagName).toBe('A')
    expect(screen.getByText('My Project').tagName).toBe('A')
    expect(screen.getByText('staging').tagName).toBe('A')
    expect(screen.getByText('dark-mode').tagName).toBe('A')
    // Last segment
    expect(screen.getByText('Rules').tagName).toBe('SPAN')
  })

  it('does not render undefined or empty segments', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
    ])
    const { container } = render(<Breadcrumbs />)

    const items = container.querySelectorAll('li')
    // Should be exactly 2: Home, My Project
    expect(items).toHaveLength(2)
  })

  it('has a nav element with aria-label', () => {
    setMatches([
      { routeId: '__root__' },
      { routeId: '/_authenticated' },
      { routeId: '/_authenticated/projects/$slug', params: { slug: 'my-proj' }, loaderData: { name: 'My Project' } },
    ])
    render(<Breadcrumbs />)

    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(nav).toBeInTheDocument()
  })
})
