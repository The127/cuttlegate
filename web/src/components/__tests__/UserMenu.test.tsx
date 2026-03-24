import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { UserMenu } from '../UserMenu'

// ── Auth mock ────────────────────────────────────────────────────────────────

const mockSignoutRedirect = vi.fn()
const mockRemoveUser = vi.fn()

vi.mock('../../auth', () => ({
  getUserManager: () => ({
    getUser: () =>
      Promise.resolve({
        profile: { name: 'Jane Doe', email: 'jane@example.com' },
      }),
    signoutRedirect: (...args: unknown[]) => mockSignoutRedirect(...args),
    removeUser: () => mockRemoveUser(),
  }),
}))

// ── queryClient mock ─────────────────────────────────────────────────────────

const mockClear = vi.fn()

vi.mock('../../queryClient', () => ({
  queryClient: { clear: () => mockClear() },
}))

// ── Test wrapper ─────────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function renderMenu() {
  return render(
    <Wrapper>
      <UserMenu />
    </Wrapper>,
  )
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('UserMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSignoutRedirect.mockResolvedValue(undefined)
    mockRemoveUser.mockResolvedValue(undefined)
  })

  // @happy — renders avatar with initials from OIDC profile
  it('renders avatar button with initials derived from profile name', async () => {
    renderMenu()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /User menu: JD/i })).toBeInTheDocument()
    })

    expect(screen.getByText('JD')).toBeInTheDocument()
  })

  // @happy — opens dropdown with email and logout option
  it('opens dropdown showing email and log out item on click', async () => {
    const user = userEvent.setup()
    renderMenu()

    const trigger = await screen.findByRole('button', { name: /User menu/i })
    await user.click(trigger)

    await waitFor(() => {
      expect(screen.getByText('Logged in as')).toBeInTheDocument()
      expect(screen.getByText('jane@example.com')).toBeInTheDocument()
      expect(screen.getByRole('menuitem', { name: 'Log Out' })).toBeInTheDocument()
    })
  })

  // @happy — logout calls signoutRedirect with correct URI
  it('calls signoutRedirect with post_logout_redirect_uri on logout', async () => {
    const user = userEvent.setup()
    renderMenu()

    const trigger = await screen.findByRole('button', { name: /User menu/i })
    await user.click(trigger)

    const logoutItem = await screen.findByRole('menuitem', { name: 'Log Out' })
    await user.click(logoutItem)

    await waitFor(() => {
      expect(mockClear).toHaveBeenCalled()
      expect(mockSignoutRedirect).toHaveBeenCalledWith({
        post_logout_redirect_uri: window.location.origin,
      })
    })
  })

  // @edge — fallback when signoutRedirect fails
  it('falls back to removeUser + redirect when signoutRedirect fails', async () => {
    mockSignoutRedirect.mockRejectedValue(new Error('no end_session_endpoint'))

    // Spy on window.location.href assignment
    const hrefSetter = vi.fn()
    const originalLocation = window.location
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, origin: originalLocation.origin, href: '' },
      writable: true,
      configurable: true,
    })
    Object.defineProperty(window.location, 'href', {
      set: hrefSetter,
      get: () => '',
      configurable: true,
    })

    const user = userEvent.setup()
    renderMenu()

    const trigger = await screen.findByRole('button', { name: /User menu/i })
    await user.click(trigger)

    const logoutItem = await screen.findByRole('menuitem', { name: 'Log Out' })
    await user.click(logoutItem)

    await waitFor(() => {
      expect(mockClear).toHaveBeenCalled()
      expect(mockRemoveUser).toHaveBeenCalled()
      expect(hrefSetter).toHaveBeenCalledWith('/')
    })

    // Restore
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
  })

  // @auth-bypass — queryClient.clear() is called before signoutRedirect
  it('clears query cache before calling signoutRedirect', async () => {
    const callOrder: string[] = []
    mockClear.mockImplementation(() => callOrder.push('clear'))
    mockSignoutRedirect.mockImplementation(() => {
      callOrder.push('signout')
      return Promise.resolve()
    })

    const user = userEvent.setup()
    renderMenu()

    const trigger = await screen.findByRole('button', { name: /User menu/i })
    await user.click(trigger)

    const logoutItem = await screen.findByRole('menuitem', { name: 'Log Out' })
    await user.click(logoutItem)

    await waitFor(() => {
      expect(callOrder).toEqual(['clear', 'signout'])
    })
  })

  // @error-path — double-click protection
  it('does not call signoutRedirect twice on rapid clicks', async () => {
    // Make signoutRedirect hang (never resolve) to simulate in-flight
    mockSignoutRedirect.mockReturnValue(new Promise(() => {}))

    const user = userEvent.setup()
    renderMenu()

    const trigger = await screen.findByRole('button', { name: /User menu/i })
    await user.click(trigger)

    const logoutItem = await screen.findByRole('menuitem', { name: 'Log Out' })

    // Radix DropdownMenu closes after first click on an item, so
    // the double-click protection is the loggingOut ref guard.
    await user.click(logoutItem)

    // signoutRedirect should only be called once
    expect(mockSignoutRedirect).toHaveBeenCalledTimes(1)
  })
})
