import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor, screen } from '@testing-library/react'
import { axe } from 'jest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'test-project', envSlug: 'production', key: 'dark-mode' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()
const mockPatchJSON = vi.fn()
const mockPostJSON = vi.fn()
const mockDeleteRequest = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    patchJSON: (...args: unknown[]) => mockPatchJSON(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
  }
})

// useFlagSSE opens a real fetch — stub it out
vi.mock('../../hooks/useFlagSSE', () => ({
  useFlagSSE: () => {},
}))

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const FLAG_FIXTURE = {
  id: 'flag-1',
  key: 'dark-mode',
  name: 'Dark Mode',
  type: 'bool',
  variants: [{ key: 'true', name: 'On' }, { key: 'false', name: 'Off' }],
  default_variant_key: 'false',
  enabled: true,
}

const ENVS_FIXTURE = {
  environments: [{ id: 'env-1', slug: 'production', name: 'Production' }],
}

const ENV_STATE_FIXTURE = { enabled: true }

async function loadFlagDetailPage() {
  const mod = await import('../projects/$slug.environments.$envSlug.flags.$key')
  return mod
}

describe('FlagDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // The environments list and flag detail/env-state share URL patterns.
    // FLAG_FIXTURE satisfies both shapes (has `enabled` for env-state, `variants` for detail).
    mockFetchJSON.mockImplementation((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      return Promise.resolve(FLAG_FIXTURE)
    })
  })

  it('renders without accessibility violations', async () => {
    const { flagDetailRoute } = await loadFlagDetailPage()
    const FlagDetailPage = flagDetailRoute.options.component

    const { container } = render(<Wrapper><FlagDetailPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  // @edge: never-evaluated flag shows "Never" not blank in stats section.
  it('shows Never in stats panel for flag with no evaluations', async () => {
    const { flagDetailRoute } = await loadFlagDetailPage()
    const FlagDetailPage = flagDetailRoute.options.component

    render(<Wrapper><FlagDetailPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getAllByText('Never').length).toBeGreaterThan(0)
    })
  })

  // @happy: stats section shows count when flag has evaluations.
  it('shows evaluation count in stats panel', async () => {
    mockFetchJSON.mockImplementation((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: '2026-03-21T14:00:00Z', evaluation_count: 42 })
      return Promise.resolve(FLAG_FIXTURE)
    })

    const { flagDetailRoute } = await loadFlagDetailPage()
    const FlagDetailPage = flagDetailRoute.options.component

    render(<Wrapper><FlagDetailPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument()
    })
  })
})
