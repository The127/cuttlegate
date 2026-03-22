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

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const SEGMENTS_FIXTURE = {
  segments: [
    {
      id: 'seg-1',
      slug: 'beta-users',
      name: 'Beta Users',
      projectId: 'test-project',
      createdAt: '2026-01-01T00:00:00Z',
    },
  ],
}

async function loadSegmentListPage() {
  const mod = await import('../projects/$slug.segments')
  return mod
}

describe('SegmentListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations when segments load', async () => {
    mockFetchJSON.mockResolvedValue(SEGMENTS_FIXTURE)

    const { segmentListRoute } = await loadSegmentListPage()
    const SegmentListPage = segmentListRoute.options.component

    const { container } = render(<Wrapper><SegmentListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta Users')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('renders without accessibility violations in empty state', async () => {
    mockFetchJSON.mockResolvedValue({ segments: [] })

    const { segmentListRoute } = await loadSegmentListPage()
    const SegmentListPage = segmentListRoute.options.component

    const { container } = render(<Wrapper><SegmentListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/No segments yet/)).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
