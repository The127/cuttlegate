import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor, screen, fireEvent } from '@testing-library/react'
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

const EMPTY_HISTORY = { entries: [], next_cursor: null }

// @happy: history with entries across two environments
const MULTI_ENV_HISTORY = {
  entries: [
    {
      id: 'h1',
      occurred_at: '2026-03-22T09:30:00Z',
      actor_email: 'alice@example.com',
      action: 'flag.created',
      environment_slug: '',
    },
    {
      id: 'h2',
      occurred_at: '2026-03-21T12:00:00Z',
      actor_email: 'bob@example.com',
      action: 'flag.state_changed',
      environment_slug: 'production',
    },
    {
      id: 'h3',
      occurred_at: '2026-03-20T09:30:00Z',
      actor_email: 'alice@example.com',
      action: 'flag.state_changed',
      environment_slug: 'staging',
    },
  ],
  next_cursor: null,
}

function mockFetchBase(path: string) {
  if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
  if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
  if (path.includes('/audit')) return Promise.resolve(EMPTY_HISTORY)
  return Promise.resolve(FLAG_FIXTURE)
}

async function loadFlagDetailPage() {
  const mod = await import('../projects/$slug.environments.$envSlug.flags.$key')
  return mod
}

async function renderAndOpenHistory(mockImpl?: (path: string) => unknown) {
  if (mockImpl) {
    mockFetchJSON.mockImplementation(mockImpl)
  }
  const { flagDetailRoute } = await loadFlagDetailPage()
  const FlagDetailPage = flagDetailRoute.options.component
  const result = render(<Wrapper><FlagDetailPage /></Wrapper>)
  await waitFor(() => expect(screen.getByText('Dark Mode')).toBeInTheDocument())
  const toggleBtn = screen.getByRole('button', { name: /change history/i })
  fireEvent.click(toggleBtn)
  return result
}

describe('FlagDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // The environments list and flag detail/env-state share URL patterns.
    // FLAG_FIXTURE satisfies both shapes (has `enabled` for env-state, `variants` for detail).
    mockFetchJSON.mockImplementation(mockFetchBase)
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
      if (path.includes('/audit')) return Promise.resolve(EMPTY_HISTORY)
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

describe('FlagChangeHistoryPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetchJSON.mockImplementation(mockFetchBase)
  })

  // @happy — flag with changes renders full history table
  it('renders change history table with entries when opened', async () => {
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.resolve(MULTI_ENV_HISTORY)
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    const rows = screen.getAllByRole('row')
    // 1 header row + 3 data rows
    expect(rows).toHaveLength(4)
    expect(screen.getAllByText('alice@example.com')).toHaveLength(2)
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
    // Known action mapped to human-readable label
    expect(screen.getByText('Created')).toBeInTheDocument()
    expect(screen.getAllByText('State changed')).toHaveLength(2)
  })

  // @happy — env filter visible when >=2 distinct non-empty env slugs present
  it('shows env filter dropdown when 2+ distinct environment values exist', async () => {
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.resolve(MULTI_ENV_HISTORY)
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    // production and staging are non-empty env slugs → filter shown
    expect(screen.getByRole('combobox', { name: /filter by environment/i })).toBeInTheDocument()
  })

  // @happy — project-scoped events only: env filter hidden, "—" shown for env column
  it('hides env filter and shows dash for entries without environment_slug', async () => {
    const projectScopedHistory = {
      entries: [
        { id: 'p1', occurred_at: '2026-03-20T10:00:00Z', actor_email: 'alice@example.com', action: 'flag.created', environment_slug: '' },
        { id: 'p2', occurred_at: '2026-03-21T10:00:00Z', actor_email: 'alice@example.com', action: 'flag.updated', environment_slug: '' },
      ],
      next_cursor: null,
    }
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.resolve(projectScopedHistory)
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    // No env filter when <2 distinct envs
    expect(screen.queryByRole('combobox', { name: /filter by environment/i })).not.toBeInTheDocument()
    // Each project-scoped entry shows "—" in environment column
    expect(screen.getAllByText('—')).toHaveLength(2)
  })

  // @edge — empty state: no entries returned
  it('shows global empty state message when API returns no entries', async () => {
    await renderAndOpenHistory()

    await waitFor(() => {
      expect(screen.getByRole('status')).toBeInTheDocument()
      expect(screen.getByText('No changes recorded yet')).toBeInTheDocument()
    })

    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  })

  // @edge — cap at 20: fetches with limit=20, no pagination control shown
  it('fetches with limit=20 and renders no load-more control', async () => {
    const twentyEntries = Array.from({ length: 20 }, (_, i) => ({
      id: `e${i}`,
      occurred_at: '2026-03-20T10:00:00Z',
      actor_email: 'actor@example.com',
      action: 'flag.state_changed',
      environment_slug: 'production',
    }))
    let capturedAuditPath = ''
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) {
        capturedAuditPath = path
        return Promise.resolve({ entries: twentyEntries, next_cursor: null })
      }
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    expect(capturedAuditPath).toContain('limit=20')
    expect(screen.queryByRole('button', { name: /load more/i })).not.toBeInTheDocument()
    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(21) // 1 header + 20 data rows
  })

  // @edge — env filter with only 1 distinct env → filter hidden
  it('hides env filter when only 1 distinct environment value exists', async () => {
    const singleEnvHistory = {
      entries: [
        { id: 's1', occurred_at: '2026-03-21T12:00:00Z', actor_email: 'bob@example.com', action: 'flag.state_changed', environment_slug: 'production' },
        { id: 's2', occurred_at: '2026-03-20T08:00:00Z', actor_email: 'alice@example.com', action: 'flag.state_changed', environment_slug: 'production' },
      ],
      next_cursor: null,
    }
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.resolve(singleEnvHistory)
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    expect(screen.queryByRole('combobox', { name: /filter by environment/i })).not.toBeInTheDocument()
  })

  // @edge — unknown action string renders as raw text without crash
  it('renders unknown action strings as raw text', async () => {
    const unknownActionHistory = {
      entries: [
        { id: 'u1', occurred_at: '2026-03-20T10:00:00Z', actor_email: 'actor@example.com', action: 'flag.rules_reordered', environment_slug: '' },
      ],
      next_cursor: null,
    }
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.resolve(unknownActionHistory)
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('table', { name: /change history/i })).toBeInTheDocument()
    })

    // Unknown action rendered as raw string, not blank, not error
    expect(screen.getByText('flag.rules_reordered')).toBeInTheDocument()
  })

  // @error-path — API error: shows error message without crashing rest of page
  it('shows error state when audit API fails and leaves rest of page intact', async () => {
    await renderAndOpenHistory((path: string) => {
      if (path.match(/\/environments$/)) return Promise.resolve(ENVS_FIXTURE)
      if (path.endsWith('/stats')) return Promise.resolve({ last_evaluated_at: null, evaluation_count: 0 })
      if (path.includes('/audit')) return Promise.reject(new Error('internal_error'))
      return Promise.resolve(FLAG_FIXTURE)
    })

    await waitFor(() => {
      expect(screen.getByRole('status')).toBeInTheDocument()
      expect(screen.getByText('Could not load change history')).toBeInTheDocument()
    })

    // Rest of page intact
    expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
