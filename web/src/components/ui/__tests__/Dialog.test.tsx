import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { axe } from 'jest-axe'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogCloseButton,
} from '../Dialog'

// Radix Dialog renders content into a Portal outside the test container.
// axe must be run against document.body, not result.container, to capture portal content.

describe('Dialog', () => {
  it('renders with correct ARIA role when open', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Test dialog</DialogTitle>
          <DialogDescription>A description</DialogDescription>
        </DialogContent>
      </Dialog>,
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('dialog element has role="dialog" (aria-modal is managed by Radix via aria-hidden on siblings)', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Modal check</DialogTitle>
        </DialogContent>
      </Dialog>,
    )
    // Radix Dialog sets role="dialog" on the content element.
    // It enforces modal semantics via aria-hidden on background elements (not aria-modal attribute).
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('aria-labelledby points to DialogTitle', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Labelled dialog</DialogTitle>
        </DialogContent>
      </Dialog>,
    )
    const dialog = screen.getByRole('dialog')
    const labelledById = dialog.getAttribute('aria-labelledby')
    expect(labelledById).toBeTruthy()
    expect(document.getElementById(labelledById!)).toHaveTextContent('Labelled dialog')
  })

  it('aria-describedby points to DialogDescription when present', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>With description</DialogTitle>
          <DialogDescription>Helpful context</DialogDescription>
        </DialogContent>
      </Dialog>,
    )
    const dialog = screen.getByRole('dialog')
    const describedById = dialog.getAttribute('aria-describedby')
    expect(describedById).toBeTruthy()
    expect(document.getElementById(describedById!)).toHaveTextContent('Helpful context')
  })

  it('does not render when open=false', () => {
    render(
      <Dialog open={false} onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Hidden dialog</DialogTitle>
        </DialogContent>
      </Dialog>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('calls onOpenChange(false) when Escape is pressed', () => {
    const onOpenChange = vi.fn()
    render(
      <Dialog open onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogTitle>Escapable</DialogTitle>
        </DialogContent>
      </Dialog>,
    )
    const dialog = screen.getByRole('dialog')
    dialog.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('renders DialogCloseButton with translated aria-label', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogCloseButton />
          <DialogTitle>Close button dialog</DialogTitle>
        </DialogContent>
      </Dialog>,
    )
    expect(screen.getByRole('button', { name: /close/i })).toBeInTheDocument()
  })

  it('renders DialogFooter children', () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Footer dialog</DialogTitle>
          <DialogFooter>
            <button>Cancel</button>
            <button>Confirm</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>,
    )
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument()
  })

  // ── @a11y — axe-core scans ───────────────────────────────────────────────

  it('@a11y — axe passes on open Dialog with title and description', async () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Accessible dialog</DialogTitle>
          <DialogDescription>Some descriptive content</DialogDescription>
        </DialogContent>
      </Dialog>,
    )
    // Run axe on document.body to capture portal-rendered content
    const results = await axe(document.body)
    expect(results).toHaveNoViolations()
  })

  it('@a11y — axe passes on open Dialog with title only (no description)', async () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Title-only dialog</DialogTitle>
          <p>Some content without DialogDescription</p>
        </DialogContent>
      </Dialog>,
    )
    const results = await axe(document.body)
    expect(results).toHaveNoViolations()
  })

  it('@a11y — axe passes on Dialog with close button', async () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogCloseButton />
          <DialogTitle>Dialog with close</DialogTitle>
          <DialogDescription>Has a close affordance</DialogDescription>
        </DialogContent>
      </Dialog>,
    )
    const results = await axe(document.body)
    expect(results).toHaveNoViolations()
  })
})
