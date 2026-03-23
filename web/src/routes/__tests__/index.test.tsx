import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { axe } from 'jest-axe'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({}),
      useLoaderData: () => ({ id: 'proj-1', name: 'Test Project', slug: 'test-project', created_at: '2026-01-01T00:00:00Z' }),
    }),
    Link: ({ children, to, ...props }: any) => <a href={to} {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: any[]) => mockFetchJSON(...args),
  }
})

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
}

async function getHomePage() {
  const mod = await import('../index')
  const route = mod.indexRoute as any
  return route.component ?? route.options?.component
}

async function renderHome() {
  const qc = createQueryClient()
  const HomePage = await getHomePage()
  const { CreateProjectDialogProvider } = await import('../../components/CreateProjectDialog')
  return render(
    <QueryClientProvider client={qc}>
      <CreateProjectDialogProvider>
        <HomePage />
      </CreateProjectDialogProvider>
    </QueryClientProvider>,
  )
}

function mockAllEndpoints(projects: any[]) {
  mockFetchJSON.mockImplementation((path: string) => {
    if (path === '/api/v1/projects') return Promise.resolve({ projects })
    if (path.endsWith('/environments')) return Promise.resolve({ environments: [{ id: 'e1', name: 'Prod', slug: 'prod' }] })
    if (path.endsWith('/flags')) return Promise.resolve({ flags: [{ id: 'f1', key: 'test' }, { id: 'f2', key: 'test2' }] })
    if (path.endsWith('/members')) return Promise.resolve({ members: [{ id: 'm1' }] })
    return Promise.resolve({})
  })
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('HomePage', () => {
  it('shows skeleton cards while loading', async () => {
    mockFetchJSON.mockReturnValue(new Promise(() => {}))
    await renderHome()

    const skeletons = document.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it('shows project cards when projects load', async () => {
    const projects = [
      { id: '1', name: 'Alpha', slug: 'alpha', created_at: '2026-01-01T00:00:00Z' },
      { id: '2', name: 'Beta', slug: 'beta', created_at: '2026-01-02T00:00:00Z' },
    ]
    mockAllEndpoints(projects)

    await renderHome()

    await waitFor(() => {
      expect(screen.getByText('Alpha')).toBeInTheDocument()
      expect(screen.getByText('Beta')).toBeInTheDocument()
    })

    expect(screen.getByText('alpha')).toBeInTheDocument()
    expect(screen.getByText('beta')).toBeInTheDocument()

    const alphaLink = screen.getByText('Alpha').closest('a')
    expect(alphaLink).toHaveAttribute('href', '/projects/$slug')
  })

  it('shows counts progressively', async () => {
    const projects = [
      { id: '1', name: 'Alpha', slug: 'alpha', created_at: '2026-01-01T00:00:00Z' },
    ]

    let resolveEnvs: (v: any) => void
    const envsPromise = new Promise((r) => { resolveEnvs = r })

    mockFetchJSON.mockImplementation((path: string) => {
      if (path === '/api/v1/projects') return Promise.resolve({ projects })
      if (path.endsWith('/environments')) return envsPromise
      if (path.endsWith('/flags')) return Promise.resolve({ flags: [] })
      if (path.endsWith('/members')) return Promise.resolve({ members: [] })
      return Promise.resolve({})
    })

    await renderHome()

    await waitFor(() => {
      expect(screen.getByText('Alpha')).toBeInTheDocument()
    })

    resolveEnvs!({ environments: [{ id: 'e1', name: 'Prod', slug: 'prod' }, { id: 'e2', name: 'Staging', slug: 'staging' }] })

    await waitFor(() => {
      expect(screen.getByText('2')).toBeInTheDocument()
    })
  })

  it('shows empty state when no projects', async () => {
    mockFetchJSON.mockResolvedValue({ projects: [] })
    await renderHome()

    await waitFor(() => {
      expect(screen.getByText('No projects yet')).toBeInTheDocument()
      expect(screen.getByText('Create your first project')).toBeInTheDocument()
    })
  })

  it('shows error state with retry button', async () => {
    mockFetchJSON.mockRejectedValue(new Error('Network error'))
    await renderHome()

    await waitFor(() => {
      expect(screen.getByText('Failed to load projects.')).toBeInTheDocument()
      expect(screen.getByText('Retry')).toBeInTheDocument()
    })
  })

  it('has no accessibility violations in project card grid', async () => {
    const projects = [
      { id: '1', name: 'Alpha', slug: 'alpha', created_at: '2026-01-01T00:00:00Z' },
    ]
    mockAllEndpoints(projects)

    const { container } = await renderHome()

    await waitFor(() => {
      expect(screen.getByText('Alpha')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results.violations).toHaveLength(0)
  })
})
