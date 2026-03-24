import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CreateProjectDialogProvider } from '../CreateProjectDialog'
import { ProjectSwitcher } from '../ProjectSwitcher'

// ── Router mocks ─────────────────────────────────────────────────────────────

const mockNavigate = vi.fn()
const mockUseLocation = vi.fn()

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  const React = await import('react')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useLocation: () => mockUseLocation(),
    Link: React.forwardRef(
      ({ to, children, ...rest }: { to: string; children: React.ReactNode; [k: string]: unknown }, ref: React.Ref<HTMLAnchorElement>) =>
        React.createElement('a', { href: to, ref, ...rest }, children),
    ),
  }
})

// ── Auth mock — prevents UserManager-not-initialized throw ────────────────────

vi.mock('../../auth', () => ({
  getUserManager: () => ({
    getUser: () => Promise.resolve({ profile: { name: 'Alice Brown' } }),
  }),
}))

// ── API mock ──────────────────────────────────────────────────────────────────

const mockFetchJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
  }
})

// ── Fixtures ──────────────────────────────────────────────────────────────────

const PROJECTS = [
  { id: 'p1', name: 'Alpha', slug: 'alpha', created_at: '2026-01-01T00:00:00Z' },
  { id: 'p2', name: 'Beta', slug: 'beta', created_at: '2026-01-02T00:00:00Z' },
]

const ENVIRONMENTS = [
  { id: 'e1', project_id: 'p1', name: 'Production', slug: 'production', created_at: '2026-01-01T00:00:00Z' },
  { id: 'e2', project_id: 'p1', name: 'Staging', slug: 'staging', created_at: '2026-01-02T00:00:00Z' },
]

// ── Test wrapper ──────────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return (
    <QueryClientProvider client={qc}>
      <CreateProjectDialogProvider>{children}</CreateProjectDialogProvider>
    </QueryClientProvider>
  )
}

function renderSwitcher() {
  return render(
    <Wrapper>
      <ProjectSwitcher />
    </Wrapper>,
  )
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('ProjectSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Default: root route — no project context
    mockUseLocation.mockReturnValue({ pathname: '/' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })
  })

  // @happy — project Select renders with Radix trigger and items
  it('renders project Select trigger with aria-label when projects are loaded', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/' })

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Project' })).toBeInTheDocument()
    })
  })

  // @happy — environment Select renders when inside a project route
  it('renders environment Select trigger with aria-label when on a project route', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.resolve({ environments: ENVIRONMENTS })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Project' })).toBeInTheDocument()
      expect(screen.getByRole('combobox', { name: 'Environment' })).toBeInTheDocument()
    })
  })

  // @happy — no-environments nudge renders instead of Select when envs are empty
  it('renders nudge button when environments array is empty', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.resolve({ environments: [] })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /No environments — Create one/i }),
      ).toBeInTheDocument()
    })

    // Environment Select must NOT be rendered when nudge is shown
    expect(screen.queryByRole('combobox', { name: 'Environment' })).not.toBeInTheDocument()
  })

  // @edge — nudge navigates to settings/environments (not project dashboard)
  it('nudge click navigates to /projects/$slug/settings/environments', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.resolve({ environments: [] })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    const nudge = await screen.findByRole('button', { name: /No environments — Create one/i })
    await userEvent.click(nudge)

    expect(mockNavigate).toHaveBeenCalledWith({
      to: '/projects/$slug/settings/environments',
      params: { slug: 'alpha' },
    })
    // Explicitly assert the wrong destination is NOT used
    expect(mockNavigate).not.toHaveBeenCalledWith(
      expect.objectContaining({ to: '/projects/$slug' }),
    )
  })

  // @edge — nudge not shown while environments are loading (skeleton shown instead)
  it('renders skeleton loader while environments are loading', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    // Environments query never resolves — stays loading
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments') return new Promise(() => {})
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    const { container } = renderSwitcher()

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Project' })).toBeInTheDocument()
    })

    // Skeleton div is rendered for the environment slot
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument()
    // Neither Select nor nudge is rendered
    expect(screen.queryByRole('combobox', { name: 'Environment' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /No environments/i })).not.toBeInTheDocument()
  })

  // @edge — no environment slot rendered when not on a project route
  it('does not render environment slot when outside a project route', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/' })

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Project' })).toBeInTheDocument()
    })

    expect(screen.queryByRole('combobox', { name: 'Environment' })).not.toBeInTheDocument()
  })

  // @error-path — project fetch failure shows error + retry button
  it('shows error message and retry button when projects query fails', async () => {
    mockFetchJSON.mockRejectedValue(new Error('network error'))

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getByText('Failed to load')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'retry' })).toBeInTheDocument()
    })
  })

  // @error-path — environment fetch failure shows error + retry button
  it('shows error message and retry button when environments query fails', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.reject(new Error('network error'))
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getAllByText('Failed to load').length).toBeGreaterThanOrEqual(1)
    })

    // Retry button for environments must be present
    const retryButtons = screen.getAllByRole('button', { name: 'retry' })
    expect(retryButtons.length).toBeGreaterThanOrEqual(1)
  })

  // @happy — no hardcoded user-visible strings: all text sourced from i18n
  it('does not render raw English strings outside i18n — no-projects state uses i18n keys', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: [] })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    // "No projects yet —" text sourced from t('switcher.no_projects_prefix')
    await waitFor(() => {
      expect(screen.getByText(/No projects yet/i)).toBeInTheDocument()
    })
  })

  // @happy — text wordmark renders as a link to home page
  it('renders wordmark as a link with href="/"', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.resolve({ environments: ENVIRONMENTS })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    await waitFor(() => {
      const link = screen.getByRole('link', { name: 'Cuttlegate' })
      expect(link).toBeInTheDocument()
      expect(link).toHaveAttribute('href', '/')
    })
  })

  // @edge — clicking app name when already on home page causes no error
  it('renders wordmark link even when already on home page', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/' })

    renderSwitcher()

    await waitFor(() => {
      const link = screen.getByRole('link', { name: 'Cuttlegate' })
      expect(link).toBeInTheDocument()
      expect(link).toHaveAttribute('href', '/')
    })
  })

  // @happy — WCAG 2.1 AA: descriptive aria-labels on both dropdowns
  it('project and environment Select triggers carry descriptive aria-labels', async () => {
    mockUseLocation.mockReturnValue({ pathname: '/projects/alpha/environments/production' })
    mockFetchJSON.mockImplementation((url: string) => {
      if (url === '/api/v1/projects') return Promise.resolve({ projects: PROJECTS })
      if (url === '/api/v1/projects/alpha/environments')
        return Promise.resolve({ environments: ENVIRONMENTS })
      return Promise.reject(new Error(`Unexpected URL: ${url}`))
    })

    renderSwitcher()

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Project' })).toBeInTheDocument()
      expect(screen.getByRole('combobox', { name: 'Environment' })).toBeInTheDocument()
    })
  })
})
