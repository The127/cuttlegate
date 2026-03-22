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
      useParams: () => ({ slug: 'test-project', envSlug: 'production' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()
const mockPatchJSON = vi.fn()
const mockDeleteRequest = vi.fn()
const mockPostJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    patchJSON: (...args: unknown[]) => mockPatchJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const FLAGS_FIXTURE = {
  flags: [
    {
      id: 'flag-1',
      key: 'dark-mode',
      name: 'Dark Mode',
      type: 'bool',
      variants: [{ key: 'true', name: 'On' }, { key: 'false', name: 'Off' }],
      default_variant_key: 'false',
      enabled: true,
    },
  ],
}

async function loadFlagListPage() {
  const mod = await import('../projects/$slug.environments.$envSlug.flags')
  return mod
}

describe('FlagListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations when flags load', async () => {
    mockFetchJSON.mockResolvedValue(FLAGS_FIXTURE)

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { container } = render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('renders without accessibility violations in empty state', async () => {
    mockFetchJSON.mockResolvedValue({ flags: [] })

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { container } = render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/No flags yet/)).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('renders flag list with correct roles', async () => {
    mockFetchJSON.mockResolvedValue(FLAGS_FIXTURE)

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    // Toggle button must have aria-pressed
    const toggle = screen.getByRole('button', { name: /disable flag/i })
    expect(toggle).toHaveAttribute('aria-pressed', 'true')
  })
})
