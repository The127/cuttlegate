import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { axe } from 'jest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'test-project' }),
      useLoaderData: () => ({ id: 'proj-1', name: 'Test Project', slug: 'test-project', created_at: '2026-01-01T00:00:00Z' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()
const mockDeleteRequest = vi.fn()
const mockPostJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const PAGINATED_FLAGS = {
  flags: [
    {
      id: 'flag-1',
      key: 'dark-mode',
      name: 'Dark Mode',
      type: 'bool',
      variants: [{ key: 'true', name: 'On' }, { key: 'false', name: 'Off' }],
      default_variant_key: 'false',
      created_at: '2026-01-15T00:00:00Z',
    },
    {
      id: 'flag-2',
      key: 'feature-x',
      name: 'Feature X',
      type: 'multivariate',
      variants: [{ key: 'a', name: 'A' }, { key: 'b', name: 'B' }],
      default_variant_key: 'a',
      created_at: '2026-01-16T00:00:00Z',
    },
  ],
  total: 75,
  page: 1,
  per_page: 50,
}

const EMPTY_PAGE = {
  flags: [],
  total: 0,
  page: 1,
  per_page: 50,
}

async function loadProjectFlagListPage() {
  const mod = await import('../projects/$slug.flags')
  return mod
}

function mockFetchDefault(flagsResponse = PAGINATED_FLAGS) {
  mockFetchJSON.mockImplementation((url: string) => {
    if (url.includes('/environments')) return Promise.resolve({ environments: [] })
    return Promise.resolve(flagsResponse)
  })
}

describe('ProjectFlagListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders flags from paginated API response', async () => {
    mockFetchDefault()

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
      expect(screen.getByText('Feature X')).toBeInTheDocument()
    })
  })

  it('renders without accessibility violations', async () => {
    mockFetchDefault()

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    const { container } = render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('shows pagination controls when total exceeds per_page', async () => {
    mockFetchDefault()

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /previous/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /next/i })).not.toBeDisabled()
  })

  it('does not show pagination when total fits in one page', async () => {
    mockFetchDefault({
      ...PAGINATED_FLAGS,
      total: 2,
    })

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    expect(screen.queryByText(/Page \d+ of \d+/)).not.toBeInTheDocument()
  })

  it('passes search term as query param to API', async () => {
    mockFetchDefault()

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const searchInput = screen.getByRole('textbox')
    await user.type(searchInput, 'dark')

    // Wait for debounce and re-fetch
    await waitFor(() => {
      const calls = mockFetchJSON.mock.calls.map((c: unknown[]) => c[0] as string)
      const searchCall = calls.find((url: string) => url.includes('search=dark'))
      expect(searchCall).toBeDefined()
    })
  })

  it('renders empty state when no flags exist', async () => {
    mockFetchDefault(EMPTY_PAGE)

    const { projectFlagListRoute } = await loadProjectFlagListPage()
    const Page = projectFlagListRoute.options.component

    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/no flags/i)).toBeInTheDocument()
    })
  })
})
