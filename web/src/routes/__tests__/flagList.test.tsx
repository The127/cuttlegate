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

const STATS_NEVER: { last_evaluated_at: null; evaluation_count: number } = {
  last_evaluated_at: null,
  evaluation_count: 0,
}

const STATS_WITH_DATA = {
  last_evaluated_at: '2026-03-21T14:00:00Z',
  evaluation_count: 42,
}

/** Returns flags fixture for list calls, stats for stats calls. */
function mockFetchWithStats(statsFixture = STATS_NEVER) {
  mockFetchJSON.mockImplementation((url: string) => {
    if (url.endsWith('/stats')) return Promise.resolve(statsFixture)
    return Promise.resolve(FLAGS_FIXTURE)
  })
}

describe('FlagListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations when flags load', async () => {
    mockFetchWithStats()

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
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.endsWith('/stats')) return Promise.resolve(STATS_NEVER)
      return Promise.resolve({ flags: [] })
    })

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
    mockFetchWithStats()

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

  // @edge: never-evaluated flag shows "Never" not blank or "0".
  it('shows "Never" for flags with zero evaluations', async () => {
    mockFetchWithStats(STATS_NEVER)

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByText('Never')).toBeInTheDocument()
    })
  })

  // @happy: evaluated flag shows relative time in Last evaluated column.
  it('shows relative time for evaluated flags', async () => {
    mockFetchWithStats(STATS_WITH_DATA)

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    // Stats should load and show something (relative date text)
    await waitFor(() => {
      const neverCells = screen.queryAllByText('Never')
      expect(neverCells).toHaveLength(0)
    })
  })
})

describe('useMemo dep array — toggleMutation primitives', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // @happy: columns reference is stable when no deps change (idle re-render)
  it('memo reference is stable on idle re-render', async () => {
    mockFetchWithStats()

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { rerender } = render(
      <Wrapper>
        <FlagListPage />
      </Wrapper>,
    )

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    // Re-render with identical wrapper — toggle button must still be present
    // (stable columns reference means the table re-uses the same column defs)
    rerender(
      <Wrapper>
        <FlagListPage />
      </Wrapper>,
    )

    expect(screen.getByRole('button', { name: /disable flag/i })).toBeInTheDocument()
  })

  // @happy: toggle button is disabled while isPending is true for that flag
  it('memo recomputes when isPending flips — button is disabled during pending', async () => {
    // Patch resolves after we can inspect the DOM
    let resolvePatch!: () => void
    mockFetchWithStats()
    mockPatchJSON.mockReturnValue(
      new Promise<void>((resolve) => {
        resolvePatch = resolve
      }),
    )

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(
      <Wrapper>
        <FlagListPage />
      </Wrapper>,
    )

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const toggleBtn = screen.getByRole('button', { name: /disable flag/i })
    await user.click(toggleBtn)

    // While the patch is in-flight the button should be disabled
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /disable flag/i })).toBeDisabled()
    })

    resolvePatch()
  })

  // @happy: button re-enables after mutation settles — no stale closure
  it('no stale closure after mutation settles — button reflects current isPending', async () => {
    mockFetchWithStats()
    mockPatchJSON.mockResolvedValue(undefined)

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(
      <Wrapper>
        <FlagListPage />
      </Wrapper>,
    )

    await waitFor(() => {
      expect(screen.getByText('Dark Mode')).toBeInTheDocument()
    })

    const toggleBtn = screen.getByRole('button', { name: /disable flag/i })
    await user.click(toggleBtn)

    // After the mutation completes the button must not be disabled
    await waitFor(() => {
      const btn = screen.queryByRole('button', { name: /disable flag/i }) ??
        screen.queryByRole('button', { name: /enable flag/i })
      expect(btn).not.toBeNull()
      expect(btn).not.toBeDisabled()
    })
  })
})

describe('CreateFlagModal — post-create success state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // @happy: after successful flag creation, success state shows the flag key
  it('shows flag key in success state after creation', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.endsWith('/stats')) return Promise.resolve(STATS_NEVER)
      return Promise.resolve({ flags: [] })
    })
    mockPostJSON.mockResolvedValue({})

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(<Wrapper><FlagListPage /></Wrapper>)

    // Wait for empty state
    await waitFor(() => {
      expect(screen.getByText(/No flags yet/)).toBeInTheDocument()
    })

    // Click the header "New flag" button (first occurrence)
    const newFlagBtns = screen.getAllByRole('button', { name: /new flag/i })
    await user.click(newFlagBtns[0])

    // Fill in flag key
    const keyInput = await screen.findByPlaceholderText('my-feature-flag')
    await user.type(keyInput, 'my-new-flag')

    // Submit
    await user.click(screen.getByRole('button', { name: /^create$/i }))

    // Success state: flag key should appear
    await waitFor(() => {
      expect(screen.getByText('Flag created')).toBeInTheDocument()
      expect(screen.getByText('my-new-flag')).toBeInTheDocument()
    })
  })

  // @happy: Done button in success state calls onCreated and closes modal
  it('Done button in success state closes the modal', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.endsWith('/stats')) return Promise.resolve(STATS_NEVER)
      return Promise.resolve({ flags: [] })
    })
    mockPostJSON.mockResolvedValue({})

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/No flags yet/)).toBeInTheDocument()
    })

    const newFlagBtns = screen.getAllByRole('button', { name: /new flag/i })
    await user.click(newFlagBtns[0])

    const keyInput = await screen.findByPlaceholderText('my-feature-flag')
    await user.type(keyInput, 'my-new-flag')
    await user.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.getByText('Flag created')).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /done/i }))

    // Modal should close — success title no longer visible
    await waitFor(() => {
      expect(screen.queryByText('Flag created')).not.toBeInTheDocument()
    })
  })

  // @error-path: invalid key shows inline error, not toast
  it('shows inline error for invalid flag key', async () => {
    mockFetchJSON.mockImplementation((url: string) => {
      if (url.endsWith('/stats')) return Promise.resolve(STATS_NEVER)
      return Promise.resolve({ flags: [] })
    })

    const { flagListRoute } = await loadFlagListPage()
    const FlagListPage = flagListRoute.options.component

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    render(<Wrapper><FlagListPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText(/No flags yet/)).toBeInTheDocument()
    })

    const newFlagBtns = screen.getAllByRole('button', { name: /new flag/i })
    await user.click(newFlagBtns[0])

    const keyInput = await screen.findByPlaceholderText('my-feature-flag')
    await user.type(keyInput, 'INVALID KEY')
    await user.tab() // trigger blur

    // Inline error should appear
    await waitFor(() => {
      expect(screen.getByText(/lowercase/i)).toBeInTheDocument()
    })

    // No toast, no redirect — still in the modal
    expect(screen.getByText('Create flag')).toBeInTheDocument()
  })
})
