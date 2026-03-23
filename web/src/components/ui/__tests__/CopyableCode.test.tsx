import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { CopyableCode } from '../CopyableCode'

// setup.ts installs a configurable plain-object clipboard stub on window.navigator.
// We replace writeText with a fresh vi.fn() before each test via direct assignment.
let writeTextMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  writeTextMock = vi.fn().mockResolvedValue(undefined)
  ;(navigator.clipboard as Record<string, unknown>).writeText = writeTextMock
})

afterEach(() => {
  vi.useRealTimers()
})

describe('CopyableCode', () => {
  it('renders the value', () => {
    render(<CopyableCode value="my-flag-key" />)
    expect(screen.getByText('my-flag-key')).toBeInTheDocument()
  })

  it('renders as a button', () => {
    render(<CopyableCode value="my-flag-key" />)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('uses default aria-label derived from value', () => {
    render(<CopyableCode value="my-flag-key" />)
    expect(screen.getByRole('button', { name: 'Copy my-flag-key' })).toBeInTheDocument()
  })

  it('uses provided aria-label', () => {
    render(<CopyableCode value="abc123" aria-label="Copy API key" />)
    expect(screen.getByRole('button', { name: 'Copy API key' })).toBeInTheDocument()
  })

  // @happy — clicking writes value to clipboard
  it('writes value to clipboard when clicked', async () => {
    render(<CopyableCode value="flag-abc" />)
    fireEvent.click(screen.getByRole('button'))
    // Let the void promise chain settle
    await act(async () => {})
    expect(writeTextMock).toHaveBeenCalledWith('flag-abc')
  })

  it('shows checkmark icon immediately after copy', async () => {
    // Only fake setTimeout so the revert is held back, but promises resolve normally.
    vi.useFakeTimers({ toFake: ['setTimeout'] })
    const { container } = render(<CopyableCode value="flag-abc" />)

    // Before copy: no check icon (fill-rule="evenodd" belongs to CheckIcon only)
    expect(container.querySelector('[fill-rule="evenodd"]')).not.toBeInTheDocument()

    // Click and flush microtasks so the .then() sets copied=true
    await act(async () => {
      fireEvent.click(container.querySelector('button')!)
      // Flush promises — let the writeText mock resolve and setCopied(true) run
      await Promise.resolve()
    })

    // After copy: check icon present
    expect(container.querySelector('[fill-rule="evenodd"]')).toBeInTheDocument()
  })

  it('reverts to clipboard icon after 1 second', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout'] })
    const { container } = render(<CopyableCode value="flag-abc" />)

    await act(async () => {
      fireEvent.click(container.querySelector('button')!)
      await Promise.resolve()
    })

    // Icon is showing
    expect(container.querySelector('[fill-rule="evenodd"]')).toBeInTheDocument()

    // Advance past the 1000ms revert timer
    act(() => { vi.advanceTimersByTime(1001) })
    expect(container.querySelector('[fill-rule="evenodd"]')).not.toBeInTheDocument()
  })

  // @error-path — clipboard write failure is silent
  it('does not throw when clipboard write fails', async () => {
    ;(navigator.clipboard as Record<string, unknown>).writeText = vi.fn().mockRejectedValue(new Error('denied'))
    render(<CopyableCode value="flag-abc" />)
    // Should not throw — errors are silently caught in .catch()
    await expect(
      act(async () => { fireEvent.click(screen.getByRole('button')) }),
    ).resolves.not.toThrow()
  })

  // @happy — passes axe
  it('passes axe accessibility check', async () => {
    const { axeResults } = await renderWithAxe(<CopyableCode value="my-flag-key" />)
    expect(axeResults).toHaveNoViolations()
  })

  // @edge — dark mode: axe passes with dark: class variants applied
  it('passes axe in dark mode', async () => {
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
    const { axeResults } = await renderWithAxe(<CopyableCode value="my-flag-key" />)
    expect(axeResults).toHaveNoViolations()
  })

  // @edge — dark: variant classes are present in the DOM
  it('has dark: variant classes on the button', () => {
    const { container } = render(<CopyableCode value="my-flag-key" />)
    const btn = container.querySelector('button')
    expect(btn?.className).toContain('dark:text-gray-200')
    expect(btn?.className).toContain('dark:bg-gray-800')
  })
})
