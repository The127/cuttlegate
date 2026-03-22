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
      useParams: () => ({ slug: 'test-project', envSlug: 'production', key: 'dark-mode' }),
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

// Mock useInfiniteQuery so we control the data fed to EvaluationRow directly.
// The evaluations page only uses useInfiniteQuery — no other react-query hooks.
const mockUseInfiniteQuery = vi.fn()

vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual('@tanstack/react-query')
  return {
    ...actual,
    useInfiniteQuery: (...args: unknown[]) => mockUseInfiniteQuery(...args),
  }
})

function makeEvaluationEvent(overrides: Partial<{
  id: string
  occurred_at: string
  flag_key: string
  user_id: string
  input_context: Record<string, unknown>
  matched_rule: { id: string; name?: string } | null
  variant_key: string
  reason: string
}> = {}) {
  return {
    id: overrides.id ?? 'eval-1',
    occurred_at: overrides.occurred_at ?? '2026-03-22T10:00:00Z',
    flag_key: overrides.flag_key ?? 'dark-mode',
    user_id: overrides.user_id ?? 'user-123',
    input_context: overrides.input_context ?? { plan: 'premium' },
    matched_rule: overrides.matched_rule !== undefined ? overrides.matched_rule : { id: 'rule-abc', name: 'Premium Users' },
    variant_key: overrides.variant_key ?? 'on',
    reason: overrides.reason ?? 'rule_match',
  }
}

function makeQueryResult(items: ReturnType<typeof makeEvaluationEvent>[]) {
  return {
    data: {
      pages: [{ items, next_cursor: null }],
    },
    isLoading: false,
    error: null,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
  }
}

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

async function loadEvaluationsPage() {
  const mod = await import('../projects/$slug.environments.$envSlug.flags.$key.evaluations')
  return mod
}

// @happy — matched rule with a name → shows the name
describe('EvaluationRow — @happy rule with name', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('displays the rule name and includes the rule ID in the title attribute', async () => {
    const item = makeEvaluationEvent({
      matched_rule: { id: 'rule-abc', name: 'Premium Users' },
    })
    mockUseInfiniteQuery.mockReturnValue(makeQueryResult([item]))

    const { flagEvaluationsRoute } = await loadEvaluationsPage()
    const FlagEvaluationsPage = flagEvaluationsRoute.options.component

    render(<Wrapper><FlagEvaluationsPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Premium Users')).toBeInTheDocument()
    })

    // The span wrapping the name carries the rule ID in its title attribute
    const nameSpan = screen.getByText('Premium Users')
    expect(nameSpan.tagName).toBe('SPAN')
    expect(nameSpan.getAttribute('title')).toContain('rule-abc')
  })
})

// @edge — matched rule with blank name → falls back to rule ID
describe('EvaluationRow — @edge blank name', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('displays the rule ID when name is empty string', async () => {
    const item = makeEvaluationEvent({
      matched_rule: { id: 'rule-abc', name: '' },
    })
    mockUseInfiniteQuery.mockReturnValue(makeQueryResult([item]))

    const { flagEvaluationsRoute } = await loadEvaluationsPage()
    const FlagEvaluationsPage = flagEvaluationsRoute.options.component

    render(<Wrapper><FlagEvaluationsPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('rule-abc')).toBeInTheDocument()
    })

    // The span shows the ID, not an empty string — confirm the rendered text is the ID
    const ruleSpan = screen.getByText('rule-abc')
    expect(ruleSpan.tagName).toBe('SPAN')
    expect(ruleSpan.textContent).toBe('rule-abc')
  })
})

// @edge — null matched_rule → shows no-rule i18n string
describe('EvaluationRow — @edge null matched_rule', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('displays the no-rule-matched i18n string when matched_rule is null', async () => {
    const item = makeEvaluationEvent({
      matched_rule: null,
    })
    mockUseInfiniteQuery.mockReturnValue(makeQueryResult([item]))

    const { flagEvaluationsRoute } = await loadEvaluationsPage()
    const FlagEvaluationsPage = flagEvaluationsRoute.options.component

    render(<Wrapper><FlagEvaluationsPage /></Wrapper>)

    await waitFor(() => {
      // audit.no_rule_matched = "No rule"
      expect(screen.getByText('No rule')).toBeInTheDocument()
    })

    // No rule ID or name text rendered
    expect(screen.queryByText('rule-abc')).not.toBeInTheDocument()
  })
})

// @error-path — name key absent from matched_rule (undefined in JS) → falls back to ID, no crash
describe('EvaluationRow — @error-path name key absent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('treats undefined name as falsy and displays rule ID without throwing', async () => {
    // Simulate a runtime API response where the name key is entirely absent
    const item = makeEvaluationEvent({
      matched_rule: { id: 'rule-abc' } as { id: string; name: string },
    })
    mockUseInfiniteQuery.mockReturnValue(makeQueryResult([item]))

    const { flagEvaluationsRoute } = await loadEvaluationsPage()
    const FlagEvaluationsPage = flagEvaluationsRoute.options.component

    // Must not throw — render completes without error
    render(<Wrapper><FlagEvaluationsPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('rule-abc')).toBeInTheDocument()
    })
  })
})
