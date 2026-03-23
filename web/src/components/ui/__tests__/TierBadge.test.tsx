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

  // @happy — M9 design token colour classes per docs/ui-design.md TierBadge spec
  it('read tier uses design token classes', () => {
    const { container } = render(<TierBadge tier="read" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('rgba(255,255,255,0.06)')
    expect(badge?.className).toContain('color-text-secondary')
  })

  it('write tier uses design token classes', () => {
    const { container } = render(<TierBadge tier="write" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('rgba(79,124,255,0.15)')
    expect(badge?.className).toContain('#818cf8')
  })

  it('destructive tier uses design token classes', () => {
    const { container } = render(<TierBadge tier="destructive" />)
    const badge = container.querySelector('span')
    expect(badge?.className).toContain('rgba(251,191,36,0.12)')
    expect(badge?.className).toContain('#fbbf24')
  })
})
