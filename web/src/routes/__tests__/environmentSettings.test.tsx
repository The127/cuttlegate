import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router')
  return {
    ...actual,
    createRoute: (opts: any) => ({
      ...opts,
      options: opts,
      useParams: () => ({ slug: 'acme' }),
      useLoaderData: () => ({ id: 'proj-1', name: 'Acme', slug: 'acme', created_at: '2026-01-01T00:00:00Z' }),
    }),
    useLocation: () => ({ pathname: '/projects/acme/settings/environments' }),
    useNavigate: () => vi.fn(),
  }
})

const mockFetchJSON = vi.fn()
const mockPatchJSON = vi.fn()
const mockDeleteRequest = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    fetchJSON: (...args: unknown[]) => mockFetchJSON(...args),
    patchJSON: (...args: unknown[]) => mockPatchJSON(...args),
    deleteRequest: (...args: unknown[]) => mockDeleteRequest(...args),
  }
})

const mockUseProjectRole = vi.fn()
vi.mock('../../hooks/useProjectRole', () => ({
  useProjectRole: (...args: unknown[]) => mockUseProjectRole(...args),
}))

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const ENVS_FIXTURE = {
  environments: [
    {
      id: 'env-1',
      project_id: 'proj-1',
      name: 'Staging',
      slug: 'staging',
      created_at: '2026-01-15T10:00:00Z',
    },
    {
      id: 'env-2',
      project_id: 'proj-1',
      name: 'Production',
      slug: 'production',
      created_at: '2026-01-15T11:00:00Z',
    },
  ],
}

async function loadPage() {
  const mod = await import('../projects/$slug.settings.environments')
  return mod
}

describe('EnvironmentSettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockUseProjectRole.mockReturnValue({ data: 'admin' })
  })

  it('renders environment list with edit icons for admin', async () => {
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editButtons = screen.getAllByLabelText(/Edit name of environment/)
    expect(editButtons).toHaveLength(2)
  })

  it('does not render edit icons for viewer', async () => {
    mockUseProjectRole.mockReturnValue({ data: 'viewer' })
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editButtons = screen.queryAllByLabelText(/Edit name of environment/)
    expect(editButtons).toHaveLength(0)
  })

  it('opens inline input on edit click, saves on Enter', async () => {
    const user = userEvent.setup()
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    mockPatchJSON.mockResolvedValue({
      id: 'env-1',
      project_id: 'proj-1',
      name: 'QA',
      slug: 'staging',
      created_at: '2026-01-15T10:00:00Z',
    })
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editBtn = screen.getByLabelText('Edit name of environment staging')
    await user.click(editBtn)

    const input = screen.getByRole('textbox', { name: /Edit name of environment staging/ })
    expect(input).toHaveValue('Staging')

    await user.clear(input)
    await user.type(input, 'QA')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(mockPatchJSON).toHaveBeenCalledWith(
        '/api/v1/projects/acme/environments/staging',
        { name: 'QA' },
      )
    })
  })

  it('cancels edit on Escape without calling API', async () => {
    const user = userEvent.setup()
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editBtn = screen.getByLabelText('Edit name of environment staging')
    await user.click(editBtn)

    const input = screen.getByRole('textbox', { name: /Edit name of environment staging/ })
    await user.keyboard('{Escape}')

    expect(mockPatchJSON).not.toHaveBeenCalled()
    expect(screen.getByText('Staging')).toBeInTheDocument()
  })

  it('rejects empty name client-side with validation error', async () => {
    const user = userEvent.setup()
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editBtn = screen.getByLabelText('Edit name of environment staging')
    await user.click(editBtn)

    const input = screen.getByRole('textbox', { name: /Edit name of environment staging/ })
    await user.clear(input)
    // Fire Enter via keyboard event to trigger save without losing focus
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(mockPatchJSON).not.toHaveBeenCalled()
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('does not call API when name is unchanged', async () => {
    const user = userEvent.setup()
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editBtn = screen.getByLabelText('Edit name of environment staging')
    await user.click(editBtn)

    const input = screen.getByRole('textbox', { name: /Edit name of environment staging/ })
    await user.keyboard('{Enter}')

    expect(mockPatchJSON).not.toHaveBeenCalled()
  })

  it('reverts name on rename failure', async () => {
    const user = userEvent.setup()
    mockFetchJSON.mockResolvedValue(ENVS_FIXTURE)
    mockPatchJSON.mockRejectedValue(new Error('Server error'))
    const { environmentSettingsRoute } = await loadPage()
    const Page = environmentSettingsRoute.options.component
    render(<Wrapper><Page /></Wrapper>)

    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })

    const editBtn = screen.getByLabelText('Edit name of environment staging')
    await user.click(editBtn)

    const input = screen.getByRole('textbox', { name: /Edit name of environment staging/ })
    await user.clear(input)
    await user.type(input, 'NewName')
    // Blur triggers saveEdit which calls the mutation
    fireEvent.blur(input)

    await waitFor(() => {
      expect(mockPatchJSON).toHaveBeenCalledWith(
        '/api/v1/projects/acme/environments/staging',
        { name: 'NewName' },
      )
    })

    // After error, row should revert to read-only with original name
    await waitFor(() => {
      expect(screen.getByText('Staging')).toBeInTheDocument()
    })
  })
})
