import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TierBadge } from '../TierBadge'

describe('TierBadge', () => {
  // @happy — renders correct label for each tier
  it('renders read tier label', () => {
    render(<TierBadge tier="read" />)
    expect(screen.getByText('Read')).toBeInTheDocument()
  })

  it('renders write tier label', () => {
    render(<TierBadge tier="write" />)
    expect(screen.getByText('Write')).toBeInTheDocument()
  })

  it('renders destructive tier label', () => {
    render(<TierBadge tier="destructive" />)
    expect(screen.getByText('Destructive')).toBeInTheDocument()
  })

  // @happy — colour classes
  it('read tier uses neutral/grey classes', () => {
    const { container } = render(<TierBadge tier="read" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('bg-neutral-100')
    expect(badge?.className).toContain('text-neutral-600')
  })

  it('write tier uses blue classes', () => {
    const { container } = render(<TierBadge tier="write" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('bg-blue-50')
    expect(badge?.className).toContain('text-blue-700')
  })

  it('destructive tier uses amber classes', () => {
    const { container } = render(<TierBadge tier="destructive" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('bg-amber-50')
    expect(badge?.className).toContain('text-amber-700')
  })
})
