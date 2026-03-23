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
      useParams: () => ({ slug: 'test-project' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()
const mockPostJSON = vi.fn()
const mockDeleteRequest = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
  }
})

vi.mock('../../utils/date', () => ({
  formatRelativeDate: () => '2 days ago',
}))

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const ENVS_FIXTURE = {
  environments: [{ id: 'env-1', name: 'Production', slug: 'prod' }],
}

const KEYS_FIXTURE = {
  api_keys: [
    {
      id: 'key-1',
      name: 'CI Key',
      display_prefix: 'abcd1234',
      capability_tier: 'write',
      created_at: '2026-03-20T10:00:00Z',
    },
    {
      id: 'key-2',
      name: 'Read-Only Key',
      display_prefix: 'efgh5678',
      capability_tier: 'read',
      created_at: '2026-03-21T10:00:00Z',
    },
  ],
}

async function loadAPIKeyPage() {
  const mod = await import('../projects/$slug.api-keys')
  return mod
}

beforeEach(() => {
  vi.clearAllMocks()
  mockFetchJSON.mockImplementation((url: string) => {
    if (url.includes('/environments') && !url.includes('/api-keys')) {
      return Promise.resolve(ENVS_FIXTURE)
    }
    if (url.includes('/api-keys')) {
      return Promise.resolve(KEYS_FIXTURE)
    }
    return Promise.reject(new Error(`Unexpected fetch: ${url}`))
  })
})

describe('APIKeyPage — tier badge display', () => {
  // @happy — tier badge rendered per row with correct tier value
  it('renders TierBadge for each key with correct tier', async () => {
    const mod = await loadAPIKeyPage()
    const RouteComponent = mod.apiKeyListRoute.options.component as React.ComponentType

    render(
      <Wrapper>
        <RouteComponent />
      </Wrapper>,
    )

    await waitFor(() => {
      expect(screen.getByText('CI Key')).toBeInTheDocument()
    })

    // Write tier badge
    expect(screen.getByText('Write')).toBeInTheDocument()
    // Read tier badge
    expect(screen.getByText('Read')).toBeInTheDocument()
  })
})
