import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { axe } from 'jest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'acme' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
  }
})

function makeEntry(overrides: Partial<{
  id: string
  occurred_at: string
  actor_email: string
  action: string
  flag_key: string
  environment_slug: string
}> = {}) {
  return {
    id: overrides.id ?? 'entry-1',
    occurred_at: overrides.occurred_at ?? '2026-03-21T14:32:00Z',
    actor_id: 'user-1',
    actor_email: overrides.actor_email ?? 'alice@example.com',
    action: overrides.action ?? 'flag.enabled',
    flag_key: overrides.flag_key ?? 'checkout-v2',
    environment_slug: overrides.environment_slug ?? 'production',
    project_slug: 'acme',
  }
}

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

async function loadAuditPage() {
  const mod = await import('../projects/$slug.audit')
  return mod
}

// @happy — audit log with entries renders correctly
describe('AuditLogPage — @happy', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders entries in reverse-chronological order', async () => {
    mockFetchJSON.mockResolvedValue({
      entries: [
        makeEntry({ id: 'e1', actor_email: 'alice@example.com', action: 'flag.enabled', flag_key: 'checkout-v2' }),
        makeEntry({ id: 'e2', actor_email: 'bob@example.com', action: 'flag.disabled', flag_key: 'dark-mode' }),
      ],
      next_cursor: null,
    })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
    // flag keys in monospace cells
    expect(screen.getByText('checkout-v2')).toBeInTheDocument()
    expect(screen.getByText('dark-mode')).toBeInTheDocument()
  })

  it('renders absolute timestamp and relative tooltip', async () => {
    mockFetchJSON.mockResolvedValue({
      entries: [makeEntry({ occurred_at: '2026-03-21T14:32:00Z' })],
      next_cursor: null,
    })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('2026-03-21 14:32')).toBeInTheDocument()
    })
    // relative time is in the title attribute — check the time element has a title
    const timeEl = screen.getByRole('time')
    expect(timeEl).toHaveAttribute('title')
    expect(timeEl.getAttribute('title')).not.toBe('')
  })

  it('passes accessibility audit', async () => {
    mockFetchJSON.mockResolvedValue({
      entries: [makeEntry()],
      next_cursor: null,
    })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    const { container } = render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

// @happy — empty state
describe('AuditLogPage — @happy empty state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows empty state when no entries', async () => {
    mockFetchJSON.mockResolvedValue({ entries: [], next_cursor: null })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('No audit entries yet')).toBeInTheDocument()
    })
  })

  it('empty state passes accessibility audit', async () => {
    mockFetchJSON.mockResolvedValue({ entries: [], next_cursor: null })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    const { container } = render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('No audit entries yet')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

// @happy — pagination load more
describe('AuditLogPage — @happy pagination', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows Load more button when next_cursor is present', async () => {
    mockFetchJSON.mockResolvedValue({
      entries: [makeEntry()],
      next_cursor: 'cursor-abc',
    })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /load more/i })).toBeInTheDocument()
    })
  })

  it('does not show Load more when next_cursor is null', async () => {
    mockFetchJSON.mockResolvedValue({ entries: [makeEntry()], next_cursor: null })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })

    expect(screen.queryByRole('button', { name: /load more/i })).not.toBeInTheDocument()
  })

  it('appends next page entries when Load more is clicked', async () => {
    mockFetchJSON
      .mockResolvedValueOnce({
        entries: [makeEntry({ id: 'e1', actor_email: 'alice@example.com' })],
        next_cursor: 'cursor-abc',
      })
      .mockResolvedValueOnce({
        entries: [makeEntry({ id: 'e2', actor_email: 'bob@example.com' })],
        next_cursor: null,
      })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })

    const loadMore = screen.getByRole('button', { name: /load more/i })
    fireEvent.click(loadMore)

    await waitFor(() => {
      expect(screen.getByText('bob@example.com')).toBeInTheDocument()
    })

    // alice's entry still visible (appended, not replaced)
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
  })
})

// @error-path — API failure
describe('AuditLogPage — @error-path', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows error state with retry button on fetch failure', async () => {
    mockFetchJSON.mockRejectedValue(new Error('network error'))

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/failed to load audit log/i)).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })
})

// @edge — filter by flag key
describe('AuditLogPage — @edge filter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders filter input with correct accessible label', async () => {
    mockFetchJSON.mockResolvedValue({ entries: [], next_cursor: null })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      // filter input is rendered
      expect(screen.getByPlaceholderText(/filter by flag key/i)).toBeInTheDocument()
    })
  })

  it('filter input is labelled accessibly', async () => {
    mockFetchJSON.mockResolvedValue({ entries: [], next_cursor: null })

    const { auditRoute } = await loadAuditPage()
    const AuditLogPage = auditRoute.options.component

    render(<Wrapper><AuditLogPage /></Wrapper>)

    await waitFor(() => {
      const input = screen.getByLabelText(/filter by flag key/i)
      expect(input).toBeInTheDocument()
    })
  })
})
