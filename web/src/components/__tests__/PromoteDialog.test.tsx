import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { APIError } from '../../api'
import { PromoteDialog } from '../PromoteDialog'

const mockPostJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
  }
})

// Mock Radix-based Select so tests can control onValueChange without a portal.
vi.mock('../ui', async () => {
  const actual = await vi.importActual('../ui')
  return {
    ...actual,
    Select: ({
      value,
      onValueChange,
      placeholder,
      children: _children,
      'aria-label': ariaLabel,
      className,
    }: {
      value: string
      onValueChange: (v: string) => void
      placeholder?: string
      children: React.ReactNode
      'aria-label'?: string
      className?: string
    }) => (
      <select
        aria-label={ariaLabel}
        className={className}
        value={value}
        onChange={(e) => onValueChange(e.target.value)}
        data-testid="mock-select"
      >
        {placeholder && <option value="">{placeholder}</option>}
        {_children}
      </select>
    ),
    SelectItem: ({ value, children }: { value: string; children: React.ReactNode }) => (
      <option value={value}>{children}</option>
    ),
  }
})

const ENVS = [
  { id: 'env-staging', slug: 'staging', name: 'Staging' },
  { id: 'env-prod', slug: 'production', name: 'Production' },
]

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const defaultProps = {
  mode: 'single' as const,
  projectSlug: 'acme',
  sourceEnvSlug: 'staging',
  flagKey: 'dark-mode',
  environments: ENVS,
  onClose: vi.fn(),
  onSuccess: vi.fn(),
}

describe('PromoteDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // @happy: dialog renders with title
  it('renders dialog title for single flag mode', () => {
    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText(/dark-mode/)).toBeInTheDocument()
  })

  // @happy: renders title for bulk mode
  it('renders dialog title for bulk mode', () => {
    render(
      <Wrapper>
        <PromoteDialog
          {...defaultProps}
          mode="bulk"
          flagKey={undefined}
        />
      </Wrapper>,
    )
    expect(screen.getByText(/staging/)).toBeInTheDocument()
  })

  // @happy: excludes source env from target dropdown
  it('excludes source environment from target options', () => {
    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)
    // Source env (staging) should not appear as a selectable option
    const options = screen.getAllByRole('option')
    const slugs = options.map((el) => (el as HTMLOptionElement).value)
    expect(slugs).not.toContain('staging')
    expect(slugs).toContain('production')
  })

  // @happy: confirm button disabled until target selected
  it('disables confirm button when no target is selected', () => {
    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)
    const confirmBtn = screen.getByRole('button', { name: /confirm promotion/i })
    expect(confirmBtn).toBeDisabled()
  })

  // @happy: single flag promotion — happy path
  it('calls single-flag promote endpoint and shows result', async () => {
    const diff = {
      flag_key: 'dark-mode',
      enabled_before: false,
      enabled_after: true,
      rules_added: 1,
      rules_removed: 0,
    }
    mockPostJSON.mockResolvedValueOnce(diff)

    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')

    const confirmBtn = screen.getByRole('button', { name: /confirm promotion/i })
    await userEvent.click(confirmBtn)

    await waitFor(() => {
      expect(screen.getByText(/Changes applied/i)).toBeInTheDocument()
    })

    expect(mockPostJSON).toHaveBeenCalledWith(
      '/api/v1/projects/acme/environments/staging/flags/dark-mode/promote',
      { target_env_slug: 'production' },
    )
    expect(defaultProps.onSuccess).toHaveBeenCalled()
    expect(screen.getByText('dark-mode')).toBeInTheDocument()
  })

  // @happy: bulk promotion — happy path
  it('calls bulk promote endpoint and shows all diffs', async () => {
    const bulk = {
      flags: [
        { flag_key: 'flag-a', enabled_before: false, enabled_after: true, rules_added: 0, rules_removed: 0 },
        { flag_key: 'flag-b', enabled_before: true, enabled_after: true, rules_added: 2, rules_removed: 1 },
      ],
    }
    mockPostJSON.mockResolvedValueOnce(bulk)

    render(
      <Wrapper>
        <PromoteDialog
          {...defaultProps}
          mode="bulk"
          flagKey={undefined}
        />
      </Wrapper>,
    )

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')

    await userEvent.click(screen.getByRole('button', { name: /confirm promotion/i }))

    await waitFor(() => {
      expect(screen.getByText(/Changes applied/i)).toBeInTheDocument()
    })

    expect(mockPostJSON).toHaveBeenCalledWith(
      '/api/v1/projects/acme/environments/staging/promote',
      { target_env_slug: 'production' },
    )
    expect(screen.getByText('flag-a')).toBeInTheDocument()
    expect(screen.getByText('flag-b')).toBeInTheDocument()
  })

  // @error-path: 403 forbidden — body: {"error":"forbidden","message":"You must be an admin..."}
  it('shows forbidden error message on 403', async () => {
    mockPostJSON.mockRejectedValueOnce(new APIError(403, 'Forbidden', 'forbidden'))

    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')
    await userEvent.click(screen.getByRole('button', { name: /confirm promotion/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/admin/i)
    })
  })

  // @error-path: 400 validation error — body: {"error":"validation_error","message":"...must differ"}
  it('shows same-env error message on 400', async () => {
    mockPostJSON.mockRejectedValueOnce(
      new APIError(400, 'source and target environments must differ', 'validation_error'),
    )

    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')
    await userEvent.click(screen.getByRole('button', { name: /confirm promotion/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/must differ/i)
    })
  })

  // @error-path: generic server error
  it('shows generic error on unexpected failure', async () => {
    mockPostJSON.mockRejectedValueOnce(new Error('network timeout'))

    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')
    await userEvent.click(screen.getByRole('button', { name: /confirm promotion/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/failed/i)
    })
  })

  // @edge: clicking backdrop calls onClose
  it('calls onClose when backdrop is clicked', async () => {
    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)
    const backdrop = document.querySelector('[aria-hidden="true"]') as HTMLElement
    await userEvent.click(backdrop)
    expect(defaultProps.onClose).toHaveBeenCalled()
  })

  // @edge: result step shows no-change message when nothing changed
  it('shows no-change message when enabled and rules did not change', async () => {
    const diff = {
      flag_key: 'static-flag',
      enabled_before: true,
      enabled_after: true,
      rules_added: 0,
      rules_removed: 0,
    }
    mockPostJSON.mockResolvedValueOnce(diff)

    render(<Wrapper><PromoteDialog {...defaultProps} /></Wrapper>)

    const select = screen.getByTestId('mock-select')
    await userEvent.selectOptions(select, 'production')
    await userEvent.click(screen.getByRole('button', { name: /confirm promotion/i }))

    await waitFor(() => {
      expect(screen.getByText(/Rules unchanged/i)).toBeInTheDocument()
    })
  })
})
