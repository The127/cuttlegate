import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
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
      useLoaderData: () => ({ id: 'proj-1', name: 'Test Project', slug: 'test-project', created_at: '2026-01-01T00:00:00Z' }),
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
      name: 'Beta users',
      priority: 1,
      conditions: [{ attribute: 'plan', operator: 'eq', values: ['pro'] }],
      variantKey: 'enabled',
      enabled: true,
      createdAt: '2026-03-20T10:00:00Z',
    },
  ],
}

const RULES_MULTI_FIXTURE = {
  rules: [
    {
      id: 'rule-1',
      name: 'Beta users',
      priority: 1,
      conditions: [{ attribute: 'plan', operator: 'eq', values: ['pro'] }],
      variantKey: 'enabled',
      enabled: true,
      createdAt: '2026-03-20T10:00:00Z',
    },
    {
      id: 'rule-2',
      name: 'Internal testers',
      priority: 2,
      conditions: [{ attribute: 'org', operator: 'eq', values: ['acme'] }],
      variantKey: 'enabled',
      enabled: true,
      createdAt: '2026-03-20T11:00:00Z',
    },
    {
      id: 'rule-3',
      name: 'Canary rollout',
      priority: 3,
      conditions: [],
      variantKey: 'disabled',
      enabled: true,
      createdAt: '2026-03-20T12:00:00Z',
    },
  ],
}

const RULES_FIXTURE_BLANK_NAME = {
  rules: [
    {
      id: 'rule-1',
      name: '',
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

  // Scenario 1: rule with a name shows the name in the list
  it('displays rule name in the rules list when name is set', async () => {
    mockFetchJSON.mockImplementation((path: string) => {
      if (path.includes('/rules')) return Promise.resolve(RULES_FIXTURE)
      if (path.includes('/segments')) return Promise.resolve(SEGMENTS_FIXTURE)
      if (path.includes('/flags/')) return Promise.resolve(VARIANTS_FIXTURE)
      return Promise.resolve({})
    })

    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })
  })

  // Scenario 2 & 8: blank name (or empty string from API) shows fallback "Rule {priority}"
  it('displays fallback name when rule name is blank', async () => {
    mockFetchJSON.mockImplementation((path: string) => {
      if (path.includes('/rules')) return Promise.resolve(RULES_FIXTURE_BLANK_NAME)
      if (path.includes('/segments')) return Promise.resolve(SEGMENTS_FIXTURE)
      if (path.includes('/flags/')) return Promise.resolve(VARIANTS_FIXTURE)
      return Promise.resolve({})
    })

    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Rule 1')).toBeInTheDocument()
    })
  })

  // Scenario 3: name input is pre-populated in edit form
  it('pre-populates name input when opening edit form', async () => {
    setupHappyPath()

    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /edit/i }))

    await waitFor(() => {
      const nameInput = screen.getByRole('textbox', { name: /rule name/i })
      expect(nameInput).toHaveValue('Beta users')
    })
  })

  // Scenario 9: name input present in create form
  it('shows name input in the new rule form', async () => {
    setupHappyPath()

    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Targeting Rules')).toBeInTheDocument()
    })

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /add rule/i }))

    await waitFor(() => {
      expect(screen.getByRole('textbox', { name: /rule name/i })).toBeInTheDocument()
    })
  })
})

// ── Drag-and-drop reordering ────────────────────────────────────────────────

describe('RulesPage — drag-and-drop reordering', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  function setupMultiRules() {
    mockFetchJSON.mockImplementation((path: string) => {
      if (path.includes('/rules')) return Promise.resolve(RULES_MULTI_FIXTURE)
      if (path.includes('/segments')) return Promise.resolve(SEGMENTS_FIXTURE)
      if (path.includes('/flags/')) return Promise.resolve(VARIANTS_FIXTURE)
      return Promise.resolve({})
    })
  }

  it('renders drag handles with correct aria attributes', async () => {
    setupMultiRules()
    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })

    const dragHandles = screen.getAllByRole('button', { name: /drag to reorder/i })
    expect(dragHandles).toHaveLength(3)

    // Each drag handle should have aria-label and role attributes from dnd-kit
    for (const handle of dragHandles) {
      expect(handle).toHaveAttribute('aria-label')
      // dnd-kit sets aria-roledescription="sortable" and tabindex
      expect(handle).toHaveAttribute('aria-roledescription', 'sortable')
      expect(handle.tabIndex).toBe(0)
    }
  })

  // Helper: simulate a keyboard-driven dnd-kit reorder (Space to pick up, ArrowDown to move, Space to drop).
  // dnd-kit's KeyboardSensor needs getBoundingClientRect to compute positions.
  async function performKeyboardReorder(handle: HTMLElement) {
    // Mock bounding rects for sortable items so dnd-kit can compute positions
    const sortableItems = document.querySelectorAll('[data-testid]').length > 0
      ? Array.from(document.querySelectorAll('[data-testid]'))
      : Array.from(handle.closest('[class*="space-y"]')?.children ?? [])

    sortableItems.forEach((el, i) => {
      vi.spyOn(el as HTMLElement, 'getBoundingClientRect').mockReturnValue({
        x: 0, y: i * 80, width: 600, height: 70, top: i * 80, bottom: i * 80 + 70, left: 0, right: 600,
        toJSON: () => {},
      })
    })

    fireEvent.keyDown(handle, { code: 'Space' })
    await new Promise((r) => setTimeout(r, 50))
    fireEvent.keyDown(handle, { code: 'ArrowDown' })
    await new Promise((r) => setTimeout(r, 50))
    fireEvent.keyDown(handle, { code: 'Space' })
    await new Promise((r) => setTimeout(r, 50))
  }

  it('PATCHes only changed priorities on keyboard reorder', async () => {
    setupMultiRules()
    mockPatchJSON.mockResolvedValue({})
    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })

    const dragHandles = screen.getAllByRole('button', { name: /drag to reorder/i })
    await performKeyboardReorder(dragHandles[0])

    // Wait for PATCH calls
    await waitFor(() => {
      expect(mockPatchJSON).toHaveBeenCalled()
    })

    // Moving rule-1 from position 1→2 and rule-2 from 2→1 means only those two priorities change.
    // rule-3 stays at priority 3. So exactly 2 PATCH calls.
    expect(mockPatchJSON).toHaveBeenCalledTimes(2)

    // Verify each PATCH sends the correct priority
    const calls = mockPatchJSON.mock.calls
    const patchedPriorities = calls.map((call: any[]) => ({
      ruleId: (call[0] as string).split('/').pop(),
      priority: (call[1] as any).priority,
    }))

    // rule-1 moved to priority 2, rule-2 moved to priority 1
    expect(patchedPriorities).toContainEqual({ ruleId: 'rule-1', priority: 2 })
    expect(patchedPriorities).toContainEqual({ ruleId: 'rule-2', priority: 1 })
  })

  it('shows optimistic reorder in DOM immediately', async () => {
    setupMultiRules()
    // Make PATCH hang so we can observe the optimistic state
    mockPatchJSON.mockImplementation(() => new Promise(() => {}))
    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })

    const dragHandles = screen.getAllByRole('button', { name: /drag to reorder/i })
    await performKeyboardReorder(dragHandles[0])

    // Wait for PATCH to be called (optimistic update happened)
    await waitFor(() => {
      expect(mockPatchJSON).toHaveBeenCalled()
    })

    // Verify DOM order: "Internal testers" should now appear before "Beta users"
    const ruleNames = screen.getAllByText(/Beta users|Internal testers|Canary rollout/)
    const nameTexts = ruleNames.map((el) => el.textContent)
    expect(nameTexts.indexOf('Internal testers')).toBeLessThan(nameTexts.indexOf('Beta users'))
  })

  it('reverts to original order on PATCH failure', async () => {
    setupMultiRules()
    mockPatchJSON.mockRejectedValue(new Error('Network error'))
    const { flagRulesRoute } = await loadRulesPage()
    const RulesPage = flagRulesRoute.options.component

    render(<Wrapper><RulesPage /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Beta users')).toBeInTheDocument()
    })

    const dragHandles = screen.getAllByRole('button', { name: /drag to reorder/i })
    await performKeyboardReorder(dragHandles[0])

    // Wait for PATCH to fail and rollback
    await waitFor(() => {
      expect(mockPatchJSON).toHaveBeenCalled()
    })

    // After rollback, the original order is restored: Beta users first
    await waitFor(() => {
      const ruleNames = screen.getAllByText(/Beta users|Internal testers|Canary rollout/)
      const nameTexts = ruleNames.map((el) => el.textContent)
      expect(nameTexts.indexOf('Beta users')).toBeLessThan(nameTexts.indexOf('Internal testers'))
    })
  })
})
