import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { User, UserManager } from 'oidc-client-ts'

// ── Mocks ──────────────────────────────────────────────────────────────────

const mockGetUser = vi.fn<() => Promise<User | null>>()
const mockSignoutRedirect = vi.fn<() => Promise<void>>()
const mockRemoveUser = vi.fn<() => Promise<void>>()
const mockAddUserLoaded = vi.fn()
const mockRemoveUserLoaded = vi.fn()

vi.mock('../../auth', () => ({
  getUserManager: (): Partial<UserManager> => ({
    getUser: mockGetUser,
    signoutRedirect: mockSignoutRedirect,
    removeUser: mockRemoveUser,
    events: {
      addUserLoaded: mockAddUserLoaded,
      removeUserLoaded: mockRemoveUserLoaded,
    },
  }),
}))

// ── Import after mock ─────────────────────────────────────────────────────

import { UserMenu } from '../UserMenu'

// ── Helpers ───────────────────────────────────────────────────────────────

function fakeUser(overrides: { name?: string; email?: string } = {}): User {
  return {
    profile: {
      name: overrides.name,
      email: overrides.email,
      sub: 'user-123',
      iss: 'https://auth.example.com',
      aud: 'cuttlegate',
      exp: Math.floor(Date.now() / 1000) + 3600,
      iat: Math.floor(Date.now() / 1000),
    },
    expired: false,
    access_token: 'secret-access-token',
    id_token: 'secret-id-token',
  } as unknown as User
}

// ── Tests ─────────────────────────────────────────────────────────────────

describe('UserMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('displays initial from display name', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    expect(badge).toHaveTextContent('K')
  })

  it('falls back to email when name is absent', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ email: 'zara@example.com' }))
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    expect(badge).toHaveTextContent('Z')
  })

  it('renders nothing when user is null', async () => {
    mockGetUser.mockResolvedValue(null)
    const { container } = render(<UserMenu />)

    // Wait a tick for the effect to run
    await vi.waitFor(() => {
      expect(container.innerHTML).toBe('')
    })
  })

  it('shows email in dropdown', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    const user = userEvent.setup()
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    await user.click(badge)

    expect(screen.getByText('Logged in as')).toBeInTheDocument()
    expect(screen.getByText('karo@example.com')).toBeInTheDocument()
  })

  it('shows logout button in dropdown', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    const user = userEvent.setup()
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    await user.click(badge)

    expect(screen.getByText('Log out')).toBeInTheDocument()
  })

  it('calls signoutRedirect on logout', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    mockSignoutRedirect.mockResolvedValue(undefined)
    const user = userEvent.setup()
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    await user.click(badge)
    await user.click(screen.getByText('Log out'))

    expect(mockSignoutRedirect).toHaveBeenCalledOnce()
  })

  it('falls back to removeUser when signoutRedirect fails', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    mockSignoutRedirect.mockRejectedValue(new Error('end_session_endpoint not found'))
    mockRemoveUser.mockResolvedValue(undefined)

    // Mock window.location
    const locationSpy = vi.spyOn(window, 'location', 'get').mockReturnValue({
      ...window.location,
      href: 'http://localhost:3000/projects/test',
    } as Location)
    const hrefSetter = vi.fn()
    Object.defineProperty(window.location, 'href', {
      set: hrefSetter,
      configurable: true,
    })

    const user = userEvent.setup()
    render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    await user.click(badge)
    await user.click(screen.getByText('Log out'))

    await vi.waitFor(() => {
      expect(mockRemoveUser).toHaveBeenCalledOnce()
    })

    locationSpy.mockRestore()
  })

  it('does not expose token data in rendered output', async () => {
    mockGetUser.mockResolvedValue(fakeUser({ name: 'Karo', email: 'karo@example.com' }))
    const user = userEvent.setup()
    const { container } = render(<UserMenu />)

    const badge = await screen.findByRole('button', { name: 'User menu' })
    await user.click(badge)

    const html = container.innerHTML
    expect(html).not.toContain('secret-access-token')
    expect(html).not.toContain('secret-id-token')
    expect(html).not.toContain('user-123')
  })
})
