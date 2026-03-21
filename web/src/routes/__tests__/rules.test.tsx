import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderWithAxe } from '../../test/renderWithAxe'
import { APIError } from '../../api'

// ── Mocks ──────────────────────────────────────────────────────────────────

// Mock TanStack Router — provide route params without a real router
vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'test-project', envSlug: 'production', key: 'my-flag' }),
    }),
    Link: ({ children, ...props }: any) => <a {...props}>{children}</a>,
  }
})

// Mock the API module — control what each endpoint returns
const mockFetchJSON = vi.fn()
const mockPostJSON = vi.fn()
const mockPatchJSON = vi.fn()
const mockDeleteRequest = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
    patchJSON: (...args: unknown[]) => mockPatchJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
  }
})

// ── Helpers ────────────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const RULES_FIXTURE = {
  rules: [
    {
      id: 'rule-1',
      priority: 1,
      conditions: [{ attribute: 'plan', operator: 'eq', values: ['pro'] }],
      variantKey: 'enabled',
      enabled: true,
      createdAt: '2026-03-20T10:00:00Z',
    },
  ],
}

const VARIANTS_FIXTURE = {
  variants: [
    { key: 'enabled', name: 'Enabled' },
    { key: 'disabled', name: 'Disabled' },
  ],
}

const SEGMENTS_FIXTURE = { segments: [] }

function setupHappyPath() {
  mockFetchJSON.mockImplementation((path: string) => {
    if (path.includes('/rules')) return Promise.resolve(RULES_FIXTURE)
    if (path.includes('/segments')) return Promise.resolve(SEGMENTS_FIXTURE)
    if (path.includes('/flags/')) return Promise.resolve(VARIANTS_FIXTURE)
    return Promise.resolve({})
  })
}

// Dynamic import so mocks are in place before the module loads
async function loadRulesPage() {
  const mod = await import('../projects/$slug.environments.$envSlug.flags.$key.rules')
  return mod
}

// ── Tests ──────────────────────────────────────────────────────────────────

describe('RulesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations', async () => {
    setupHappyPath()
    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    const { axeResults } = await renderWithAxe(
      <Wrapper><RulesPage /></Wrapper>,
    )

    expect(axeResults).toHaveNoViolations()
  })

  it('displays inline error when create mutation returns 400', async () => {
    setupHappyPath()
    mockPostJSON.mockRejectedValue(new APIError(400, 'Invalid rule: attribute is required'))

    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    // Wait for rules to load
    await waitFor(() => {
      expect(screen.getByText('Targeting Rules')).toBeInTheDocument()
    })

    // Click "Add rule"
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /add rule/i }))

    // Click "Save" on the new rule form
    await user.click(screen.getByRole('button', { name: /save/i }))

    // Expect inline error — not a crash or unhandled error boundary
    await waitFor(() => {
      expect(screen.getByText('Invalid rule: attribute is required')).toBeInTheDocument()
    })

    // Verify the component didn't crash — the form is still visible
    expect(screen.getByText('New rule')).toBeInTheDocument()
  })
})
