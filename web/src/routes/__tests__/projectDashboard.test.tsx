import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
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
const mockPostJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
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

const FLAGS_FIXTURE = {
  flags: [
    { id: 'flag-1', key: 'dark-mode', name: 'Dark Mode', created_at: '2026-03-20T00:00:00Z' },
  ],
}

const ENV_FLAGS_FIXTURE = {
  flags: [
    { id: 'flag-1', key: 'dark-mode', name: 'Dark Mode', enabled: true },
  ],
}

async function loadDashboard() {
  const mod = await import('../projects/$slug.index')
  return mod
}

describe('ProjectDashboard — flags section environment awareness', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows environment-creation nudge when no environments exist', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments')) return Promise.resolve({ environments: [] })
      if (url.includes('/flags')) return Promise.resolve({ flags: [] })
      return Promise.resolve({})
    })

    const { projectIndexRoute } = await loadDashboard()
    const Dashboard = projectIndexRoute.options.component

    render(<Wrapper><Dashboard /></Wrapper>)

    await waitFor(() => {
      expect(
        screen.getByText('Environments scope flag states, rules, and API keys. Create one to start managing flags.'),
      ).toBeInTheDocument()
    })

    // The "Create your first flag" CTA must NOT be visible
    expect(screen.queryByText('Ship features safely with flags that control what users see in each environment.')).not.toBeInTheDocument()
  })

  it('shows flag-creation CTA when environments exist but no flags', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments') && !url.includes('/flags')) return Promise.resolve(ENVS_FIXTURE)
      if (url.includes('/flags')) return Promise.resolve({ flags: [] })
      return Promise.resolve({})
    })

    const { projectIndexRoute } = await loadDashboard()
    const Dashboard = projectIndexRoute.options.component

    render(<Wrapper><Dashboard /></Wrapper>)

    await waitFor(() => {
      expect(
        screen.getByText('Ship features safely with flags that control what users see in each environment.'),
      ).toBeInTheDocument()
    })

    // The environment nudge must NOT be visible
    expect(
      screen.queryByText('Environments scope flag states, rules, and API keys. Create one to start managing flags.'),
    ).not.toBeInTheDocument()
  })

  it('renders recent flags as clickable links to first environment', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.includes('/environments') && url.includes('/flags')) return Promise.resolve(ENV_FLAGS_FIXTURE)
      if (url.includes('/environments')) return Promise.resolve(ENVS_FIXTURE)
      if (url.includes('/flags')) return Promise.resolve(FLAGS_FIXTURE)
      return Promise.resolve({})
    })

    const { projectIndexRoute } = await loadDashboard()
    const Dashboard = projectIndexRoute.options.component

    render(<Wrapper><Dashboard /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('dark-mode')).toBeInTheDocument()
    })

    // Flag should be rendered as a link pointing to the first environment
    const link = screen.getByText('dark-mode').closest('a')
    expect(link).toBeTruthy()
    expect(link!.getAttribute('href')).toBe(
      '/projects/acme/environments/staging/flags/dark-mode',
    )
  })
})
