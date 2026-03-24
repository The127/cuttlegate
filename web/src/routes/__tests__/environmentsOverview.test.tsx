import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'acme' }),
      useLoaderData: () => ({ id: 'proj-1', name: 'Acme', slug: 'acme', created_at: '2026-01-01T00:00:00Z' }),
    }),
    Link: ({ children, to, params, ...props }: any) => (
      <a href={to?.replace?.('$slug', params?.slug)?.replace?.('$envSlug', params?.envSlug)?.replace?.('$key', params?.key) ?? to} {...props}>
        {children}
      </a>
    ),
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    postJSON: vi.fn(),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const ENVS_FIXTURE = {
  environments: [
    { id: 'env-1', name: 'Staging', slug: 'staging' },
    { id: 'env-2', name: 'Production', slug: 'production' },
  ],
}

const ENV_FLAGS_FIXTURE = {
  flags: [
    { id: 'flag-1', key: 'dark-mode', name: 'Dark Mode', enabled: true },
    { id: 'flag-2', key: 'beta', name: 'Beta', enabled: false },
  ],
}

async function loadPage() {
  const mod = await import('../projects/$slug.environments')
  return mod
}

describe('EnvironmentsOverviewPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders environment cards with name, slug badge, and flag count', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments') && url.includes('/flags'))
        return Promise.resolve(ENV_FLAGS_FIXTURE)
      if (url.includes('/environments'))
        return Promise.resolve(ENVS_FIXTURE)
      return Promise.resolve({})
    })

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    expect(screen.getByText('Production')).toBeInTheDocument()
    expect(screen.getByText('staging')).toBeInTheDocument()
    expect(screen.getByText('production')).toBeInTheDocument()

    // Flag counts should appear
    await waitFor(() => {
      expect(screen.getAllByText('2 flags')).toHaveLength(2)
    })
  })

  it('renders environment cards as links to flag list', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments') && url.includes('/flags'))
        return Promise.resolve(ENV_FLAGS_FIXTURE)
      if (url.includes('/environments'))
        return Promise.resolve(ENVS_FIXTURE)
      return Promise.resolve({})
    })

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const stagingLink = screen.getByText('Staging').closest('a')
    expect(stagingLink).toBeTruthy()
    expect(stagingLink!.getAttribute('href')).toBe(
      '/projects/acme/environments/staging/flags',
    )
  })

  it('shows empty state with CTA when no environments', async () => {
    mockFetchJSON.mockResolvedValue({ environments: [] })

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(
        screen.getByText('Environments scope flag states, rules, and API keys for each stage of your workflow.'),
      ).toBeInTheDocument()
    })

    expect(
      screen.getByText('Create your first environment'),
    ).toBeInTheDocument()
  })

  it('shows error state with retry button on fetch failure', async () => {
    mockFetchJSON.mockRejectedValue(new Error('Network error'))

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Failed to load environments.')).toBeInTheDocument()
    })

    expect(screen.getByText('Retry')).toBeInTheDocument()
  })

  it('shows Create Environment button in page header', async () => {
    mockFetchJSON.mockResolvedValue({ environments: [] })

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Create Environment')).toBeInTheDocument()
    })
  })

  it('shows skeleton loaders while loading', async () => {
    // Never resolve to keep loading state
    mockFetchJSON.mockReturnValue(new Promise(() => {}))

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    const { container } = render(<Wrapper><Page /></Wrapper>)

    // Skeleton cards should be visible (animate-pulse elements)
    const pulseElements = container.querySelectorAll('.animate-pulse')
    expect(pulseElements.length).toBeGreaterThan(0)
  })

  it('shows flag count error per card without page-level error', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments') && url.includes('/flags'))
        return Promise.reject(new Error('Flag count failed'))
      if (url.includes('/environments'))
        return Promise.resolve(ENVS_FIXTURE)
      return Promise.resolve({})
    })

    const { environmentsOverviewRoute } = await loadPage()
    const Page = environmentsOverviewRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    // Per-card error indicators, not page-level
    await waitFor(() => {
      expect(screen.getAllByText('Failed to load')).toHaveLength(2)
    })

    // Page-level error should NOT appear
    expect(screen.queryByText('Failed to load environments.')).not.toBeInTheDocument()
  })
})
