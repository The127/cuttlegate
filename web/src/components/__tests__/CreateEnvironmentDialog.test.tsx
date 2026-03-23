import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { axe } from 'jest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CreateEnvironmentDialog } from '../CreateEnvironmentDialog'

// jsdom does not implement showModal/close on <dialog> — polyfill for tests.
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute('open', '') }
  HTMLDialogElement.prototype.close = function () { this.removeAttribute('open') }
})

const mockPostJSON = vi.fn()
const mockInvalidateQueries = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
  }
})

vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual('@tanstack/react-query')
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: (...args: unknown[]) => mockInvalidateQueries(...args),
    }),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function renderDialog(props: {
  open?: boolean
  onCreated?: () => void
  onCancel?: () => void
}) {
  const onCreated = props.onCreated ?? vi.fn()
  const onCancel = props.onCancel ?? vi.fn()
  return render(
    <Wrapper>
      <CreateEnvironmentDialog
        open={props.open ?? true}
        projectSlug="my-project"
        onCreated={onCreated}
        onCancel={onCancel}
      />
    </Wrapper>,
  )
}

describe('CreateEnvironmentDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations when open', async () => {
    const { container } = renderDialog({})
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Create environment' })).toBeInTheDocument()
    })
    const results = await axe(document.body)
    expect(results).toHaveNoViolations()
  })

  it('does not render when closed', () => {
    renderDialog({ open: false })
    expect(screen.queryByRole('heading', { name: 'Create environment' })).not.toBeInTheDocument()
  })

  it('auto-generates slug from name', async () => {
    renderDialog({})
    await waitFor(() => screen.getByLabelText('Name'))

    await userEvent.type(screen.getByLabelText('Name'), 'My Production Env')

    const slugInput = screen.getByLabelText('Slug')
    expect(slugInput).toHaveValue('my-production-env')
  })

  it('disables Create button when name is empty', async () => {
    renderDialog({})
    await waitFor(() => screen.getByRole('button', { name: 'Create' }))
    expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled()
  })

  it('disables Create button when slug has validation error', async () => {
    renderDialog({})
    await waitFor(() => screen.getByLabelText('Name'))

    await userEvent.type(screen.getByLabelText('Name'), 'Prod')
    // clear slug and type an invalid one
    await userEvent.clear(screen.getByLabelText('Slug'))
    await userEvent.type(screen.getByLabelText('Slug'), 'my production env')
    await userEvent.tab() // blur to trigger validation

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: 'Create' })).toBeDisabled()
  })

  it('calls postJSON and onCreated on successful submission', async () => {
    const onCreated = vi.fn()
    mockPostJSON.mockResolvedValue({})
    renderDialog({ onCreated })

    await waitFor(() => screen.getByLabelText('Name'))

    await userEvent.type(screen.getByLabelText('Name'), 'Production')
    // slug auto-generated as "production"
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(mockPostJSON).toHaveBeenCalledWith(
        '/api/v1/projects/my-project/environments',
        { name: 'Production', slug: 'production' },
      )
      expect(onCreated).toHaveBeenCalledTimes(1)
    })
  })

  it('invalidates environments query on success', async () => {
    mockPostJSON.mockResolvedValue({})
    renderDialog({})

    await waitFor(() => screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Staging')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: ['environments', 'my-project'],
      })
    })
  })

  it('shows inline slug error on 409 conflict without closing', async () => {
    const { APIError } = await import('../../api')
    mockPostJSON.mockRejectedValue(
      Object.assign(new APIError('conflict', 'conflict', 409), { status: 409, code: 'conflict' }),
    )
    const onCreated = vi.fn()
    renderDialog({ onCreated })

    await waitFor(() => screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Production')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(
        screen.getByText('An environment with this slug already exists in this project'),
      ).toBeInTheDocument()
    })
    expect(onCreated).not.toHaveBeenCalled()
    // Dialog stays open — heading still visible
    expect(screen.getByRole('heading', { name: 'Create environment' })).toBeInTheDocument()
  })

  it('shows server error on 500 without closing', async () => {
    mockPostJSON.mockRejectedValue(new Error('network failure'))
    const onCreated = vi.fn()
    renderDialog({ onCreated })

    await waitFor(() => screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Staging')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(
        screen.getByText('Something went wrong. Please try again.'),
      ).toBeInTheDocument()
    })
    expect(onCreated).not.toHaveBeenCalled()
  })

  it('calls onCancel when Cancel button is clicked', async () => {
    const onCancel = vi.fn()
    renderDialog({ onCancel })

    await waitFor(() => screen.getByRole('button', { name: 'Cancel' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('renders without accessibility violations on error state', async () => {
    mockPostJSON.mockRejectedValue(new Error('server error'))
    const { container: _container } = renderDialog({})

    await waitFor(() => screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Staging')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(
        screen.getByText('Something went wrong. Please try again.'),
      ).toBeInTheDocument()
    })

    const results = await axe(document.body)
    expect(results).toHaveNoViolations()
  })
})
