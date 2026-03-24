import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { ThemeToggle } from '../ThemeToggle'

const STORAGE_KEY = 'cg:theme.preference'

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.removeItem(STORAGE_KEY)
    document.documentElement.classList.remove('theme-light', 'theme-dark')
  })

  it('renders three radio options: System, Light, Dark', () => {
    render(<ThemeToggle />)
    expect(screen.getByRole('radio', { name: 'System' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Light' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Dark' })).toBeInTheDocument()
  })

  it('defaults to System when no localStorage value', () => {
    render(<ThemeToggle />)
    expect(screen.getByRole('radio', { name: 'System' })).toHaveAttribute('aria-checked', 'true')
    expect(document.documentElement.classList.contains('theme-light')).toBe(false)
    expect(document.documentElement.classList.contains('theme-dark')).toBe(false)
  })

  it('sets .theme-light on <html> when Light is clicked', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    await user.click(screen.getByRole('radio', { name: 'Light' }))
    expect(document.documentElement.classList.contains('theme-light')).toBe(true)
    expect(localStorage.getItem(STORAGE_KEY)).toBe('light')
  })

  it('sets .theme-dark on <html> when Dark is clicked', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    await user.click(screen.getByRole('radio', { name: 'Dark' }))
    expect(document.documentElement.classList.contains('theme-dark')).toBe(true)
    expect(localStorage.getItem(STORAGE_KEY)).toBe('dark')
  })

  it('removes theme class and localStorage when System is clicked', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'light')
    document.documentElement.classList.add('theme-light')
    render(<ThemeToggle />)
    await user.click(screen.getByRole('radio', { name: 'System' }))
    expect(document.documentElement.classList.contains('theme-light')).toBe(false)
    expect(document.documentElement.classList.contains('theme-dark')).toBe(false)
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })

  it('reads existing preference from localStorage on mount', () => {
    localStorage.setItem(STORAGE_KEY, 'dark')
    render(<ThemeToggle />)
    expect(screen.getByRole('radio', { name: 'Dark' })).toHaveAttribute('aria-checked', 'true')
    expect(document.documentElement.classList.contains('theme-dark')).toBe(true)
  })

  it('treats invalid localStorage value as system', () => {
    localStorage.setItem(STORAGE_KEY, 'banana')
    render(<ThemeToggle />)
    expect(screen.getByRole('radio', { name: 'System' })).toHaveAttribute('aria-checked', 'true')
    expect(document.documentElement.classList.contains('theme-light')).toBe(false)
    expect(document.documentElement.classList.contains('theme-dark')).toBe(false)
  })

  it('passes axe accessibility check', async () => {
    const { axeResults } = await renderWithAxe(<ThemeToggle />)
    expect(axeResults).toHaveNoViolations()
  })
})
