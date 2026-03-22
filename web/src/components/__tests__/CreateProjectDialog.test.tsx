import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { axe } from 'jest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LiveAnnouncerProvider } from '../../hooks/useLiveAnnouncer'
import { CreateProjectDialogProvider, useOpenCreateProjectDialog } from '../CreateProjectDialog'

// jsdom does not implement showModal/close on <dialog> — polyfill for tests.
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute('open', '') }
  HTMLDialogElement.prototype.close = function () { this.removeAttribute('open') }
})

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}))

const mockPostJSON = vi.fn()

vi.mock('../../api', async () => {
  const actual = await vi.importActual('../../api')
  return {
    ...actual,
    postJSON: (...args: unknown[]) => mockPostJSON(...args),
  }
})

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return (
    <QueryClientProvider client={qc}>
      <LiveAnnouncerProvider>
        {children}
      </LiveAnnouncerProvider>
    </QueryClientProvider>
  )
}

function OpenDialogButton() {
  const open = useOpenCreateProjectDialog()
  return <button onClick={open}>Open</button>
}

describe('CreateProjectDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders without accessibility violations when open', async () => {
    const { container } = render(
      <Wrapper>
        <CreateProjectDialogProvider>
          <OpenDialogButton />
        </CreateProjectDialogProvider>
      </Wrapper>
    )

    await userEvent.click(screen.getByRole('button', { name: 'Open' }))

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'New Project' })).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('renders without accessibility violations when showing an API error', async () => {
    mockPostJSON.mockRejectedValue({ message: 'Slug already taken' })

    const { container } = render(
      <Wrapper>
        <CreateProjectDialogProvider>
          <OpenDialogButton />
        </CreateProjectDialogProvider>
      </Wrapper>
    )

    await userEvent.click(screen.getByRole('button', { name: 'Open' }))
    await waitFor(() => screen.getByRole('heading', { name: 'New Project' }))

    await userEvent.type(screen.getByLabelText('Name'), 'My Project')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
