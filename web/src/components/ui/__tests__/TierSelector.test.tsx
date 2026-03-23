import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TierSelector } from '../TierSelector'

describe('TierSelector', () => {
  // @happy — renders three tier buttons
  it('renders Read, Write, and Destructive buttons', () => {
    render(<TierSelector value="read" onChange={() => {}} />)
    expect(screen.getByText('Read')).toBeInTheDocument()
    expect(screen.getByText('Write')).toBeInTheDocument()
    expect(screen.getByText('Destructive')).toBeInTheDocument()
  })

  // @happy — read is pre-selected (aria-pressed)
  it('read button is pressed when value is read', () => {
    render(<TierSelector value="read" onChange={() => {}} />)
    const readBtn = screen.getByText('Read').closest('button')
    expect(readBtn).toHaveAttribute('aria-pressed', 'true')
    const writeBtn = screen.getByText('Write').closest('button')
    expect(writeBtn).toHaveAttribute('aria-pressed', 'false')
    const destructiveBtn = screen.getByText('Destructive').closest('button')
    expect(destructiveBtn).toHaveAttribute('aria-pressed', 'false')
  })

  // @happy — destructive warning shown only when destructive selected
  it('does not show warning when read is selected', () => {
    render(<TierSelector value="read" onChange={() => {}} />)
    expect(screen.queryByText('Allows delete operations — use with caution')).not.toBeInTheDocument()
  })

  it('does not show warning when write is selected', () => {
    render(<TierSelector value="write" onChange={() => {}} />)
    expect(screen.queryByText('Allows delete operations — use with caution')).not.toBeInTheDocument()
  })

  it('shows warning when destructive is selected', () => {
    render(<TierSelector value="destructive" onChange={() => {}} />)
    expect(screen.getByText('Allows delete operations — use with caution')).toBeInTheDocument()
  })

  // @happy — amber class applied to destructive button when active
  it('destructive button has amber active class when selected', () => {
    render(<TierSelector value="destructive" onChange={() => {}} />)
    const destructiveBtn = screen.getByText('Destructive').closest('button')
    expect(destructiveBtn?.className).toContain('bg-amber-500')
  })

  it('destructive button does not have amber active class when not selected', () => {
    render(<TierSelector value="read" onChange={() => {}} />)
    const destructiveBtn = screen.getByText('Destructive').closest('button')
    expect(destructiveBtn?.className).not.toContain('bg-amber-500')
  })

  // @happy — inline description renders for read and write tiers
  it('shows read description when read is selected', () => {
    render(<TierSelector value="read" onChange={() => {}} />)
    expect(screen.getByText('Allows flag reads only')).toBeInTheDocument()
  })

  it('shows write description when write is selected', () => {
    render(<TierSelector value="write" onChange={() => {}} />)
    expect(screen.getByText('Allows flag reads and writes')).toBeInTheDocument()
  })

  it('does not show a neutral description when destructive is selected', () => {
    render(<TierSelector value="destructive" onChange={() => {}} />)
    expect(screen.queryByText('Allows flag reads only')).not.toBeInTheDocument()
    expect(screen.queryByText('Allows flag reads and writes')).not.toBeInTheDocument()
  })

  it('description updates when tier changes from read to write', () => {
    const { rerender } = render(<TierSelector value="read" onChange={() => {}} />)
    expect(screen.getByText('Allows flag reads only')).toBeInTheDocument()
    rerender(<TierSelector value="write" onChange={() => {}} />)
    expect(screen.getByText('Allows flag reads and writes')).toBeInTheDocument()
    expect(screen.queryByText('Allows flag reads only')).not.toBeInTheDocument()
  })

  // @happy — onChange called with correct tier
  it('calls onChange with write when Write button clicked', () => {
    const onChange = vi.fn()
    render(<TierSelector value="read" onChange={onChange} />)
    fireEvent.click(screen.getByText('Write'))
    expect(onChange).toHaveBeenCalledWith('write')
  })

  it('calls onChange with destructive when Destructive button clicked', () => {
    const onChange = vi.fn()
    render(<TierSelector value="read" onChange={onChange} />)
    fireEvent.click(screen.getByText('Destructive'))
    expect(onChange).toHaveBeenCalledWith('destructive')
  })

  it('calls onChange with read when Read button clicked', () => {
    const onChange = vi.fn()
    render(<TierSelector value="write" onChange={onChange} />)
    fireEvent.click(screen.getByText('Read'))
    expect(onChange).toHaveBeenCalledWith('read')
  })
})
