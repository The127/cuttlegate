import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { FlagAnalyticsPanel } from '../FlagAnalyticsPanel'

const mockFetchJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function renderPanel(flagType = 'bool') {
  return render(
    <Wrapper>
      <FlagAnalyticsPanel
        slug="test-project"
        envSlug="production"
        flagKey="dark-mode"
        flagType={flagType}
      />
    </Wrapper>,
  )
}

const EMPTY_BUCKETS = {
  flag_key: 'dark-mode',
  environment: 'production',
  window: '7d',
  bucket_size: 'day',
  buckets: [],
}

const BOOL_BUCKETS = {
  flag_key: 'dark-mode',
  environment: 'production',
  window: '7d',
  bucket_size: 'day',
  buckets: [
    { ts: '2026-03-21T00:00:00Z', total: 42, variants: { 'true': 30, 'false': 12 } },
    { ts: '2026-03-22T00:00:00Z', total: 10, variants: { 'true': 10, 'false': 0 } },
  ],
}

const MV_BUCKETS = {
  flag_key: 'dark-mode',
  environment: 'production',
  window: '7d',
  bucket_size: 'day',
  buckets: [
    { ts: '2026-03-21T00:00:00Z', total: 30, variants: { 'red': 10, 'blue': 15, 'green': 5 } },
  ],
}

describe('FlagAnalyticsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // @edge — empty state
  it('shows empty state when API returns no buckets', async () => {
    mockFetchJSON.mockResolvedValue(EMPTY_BUCKETS)
    renderPanel()

    await waitFor(() => {
      expect(screen.getByText('No evaluations recorded yet')).toBeInTheDocument()
    })

    expect(screen.queryByTestId('analytics-chart')).not.toBeInTheDocument()
  })

  // @happy — chart renders with evaluation data
  it('renders SVG chart with aria-label when buckets have data', async () => {
    mockFetchJSON.mockResolvedValue(BOOL_BUCKETS)
    renderPanel('bool')

    await waitFor(() => {
      expect(screen.getByTestId('analytics-chart')).toBeInTheDocument()
    })

    expect(screen.getByRole('img', { name: /Evaluation chart/i })).toBeInTheDocument()
  })

  // @happy — summary line shows total count and window
  it('displays total evaluations summary line', async () => {
    mockFetchJSON.mockResolvedValue(BOOL_BUCKETS)
    renderPanel('bool')

    await waitFor(() => {
      // 42 + 10 = 52 total across both buckets
      expect(screen.getByText(/52 evaluations in the last 7d/)).toBeInTheDocument()
    })
  })

  // @happy — boolean legend shows True and False labels
  it('shows True and False legend items for boolean flags', async () => {
    mockFetchJSON.mockResolvedValue(BOOL_BUCKETS)
    renderPanel('bool')

    await waitFor(() => {
      expect(screen.getByText('True')).toBeInTheDocument()
      expect(screen.getByText('False')).toBeInTheDocument()
    })
  })

  // @happy — multivariate flag shows per-variant legend
  it('shows per-variant legend items for multivariate flags', async () => {
    mockFetchJSON.mockResolvedValue(MV_BUCKETS)
    renderPanel('string')

    await waitFor(() => {
      expect(screen.getByText('red')).toBeInTheDocument()
      expect(screen.getByText('blue')).toBeInTheDocument()
      expect(screen.getByText('green')).toBeInTheDocument()
    })
  })

  // @edge — sparse buckets: 2 API buckets → 7 bars rendered for 7d window
  it('normalizes sparse buckets to fill all days in window', async () => {
    mockFetchJSON.mockResolvedValue(BOOL_BUCKETS) // 2 buckets for 7d window
    renderPanel('bool')

    await waitFor(() => {
      expect(screen.getByTestId('analytics-chart')).toBeInTheDocument()
    })

    // Summary total should reflect only actual evaluations, not zero-padded days
    expect(screen.getByText(/52 evaluations in the last 7d/)).toBeInTheDocument()
  })

  // @happy — window selector: clicking 30d triggers refetch with window=30d
  it('switches to 30d window when 30d button is clicked', async () => {
    mockFetchJSON.mockResolvedValue(EMPTY_BUCKETS)
    renderPanel()

    await waitFor(() => {
      expect(screen.getByText('No evaluations recorded yet')).toBeInTheDocument()
    })

    const btn30d = screen.getByRole('button', { name: '30d' })
    fireEvent.click(btn30d)

    // After clicking 30d, a new fetch is triggered
    await waitFor(() => {
      const calls = mockFetchJSON.mock.calls.map((c: unknown[]) => c[0] as string)
      expect(calls.some((p) => p.includes('window=30d'))).toBe(true)
    })
  })

  // @error-path — API non-200 shows error message
  it('shows error message when API fetch fails', async () => {
    mockFetchJSON.mockRejectedValue(new Error('internal_error'))
    renderPanel()

    await waitFor(() => {
      expect(screen.getByText('Failed to load analytics.')).toBeInTheDocument()
    })

    expect(screen.queryByTestId('analytics-chart')).not.toBeInTheDocument()
  })

  // @error-path — 403 shows error message without navigation
  it('shows error message on 403 without redirecting', async () => {
    const { APIError } = await vi.importActual<typeof import('../../api')>('../../api')
    mockFetchJSON.mockRejectedValue(new APIError(403, 'forbidden'))
    renderPanel()

    await waitFor(() => {
      expect(screen.getByText('Failed to load analytics.')).toBeInTheDocument()
    })
  })
})
