import { createContext, useContext, useRef, useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { postJSON, APIError } from '../api'

interface ProjectResponse {
  id: string
  name: string
  slug: string
  created_at: string
}

function toSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

const SLUG_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/

const CreateProjectDialogContext = createContext<(() => void) | null>(null)

export function useOpenCreateProjectDialog(): () => void {
  const open = useContext(CreateProjectDialogContext)
  if (!open) throw new Error('useOpenCreateProjectDialog must be used within CreateProjectDialogProvider')
  return open
}

export function CreateProjectDialogProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false)

  return (
    <CreateProjectDialogContext.Provider value={() => setOpen(true)}>
      {children}
      <CreateProjectDialog open={open} onClose={() => setOpen(false)} />
    </CreateProjectDialogContext.Provider>
  )
}

interface CreateProjectDialogProps {
  open: boolean
  onClose: () => void
}

function CreateProjectDialog({ open, onClose }: CreateProjectDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const nameRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [apiError, setApiError] = useState('')

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    if (open && !dialog.open) {
      dialog.showModal()
      nameRef.current?.focus()
    } else if (!open && dialog.open) {
      dialog.close()
    }
  }, [open])

  function reset() {
    setName('')
    setSlug('')
    setSlugTouched(false)
    setApiError('')
  }

  function handleClose() {
    reset()
    onClose()
  }

  const mutation = useMutation({
    mutationFn: (body: { name: string; slug: string }) =>
      postJSON<ProjectResponse>('/api/v1/projects', body),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] })
      handleClose()
      void navigate({ to: '/projects/$slug', params: { slug: data.slug } })
    },
    onError: (err) => {
      if (err instanceof APIError) {
        setApiError(err.message)
      } else {
        setApiError('An unexpected error occurred.')
      }
    },
  })

  const handleNameChange = (value: string) => {
    setName(value)
    if (!slugTouched) {
      setSlug(toSlug(value))
    }
  }

  const handleSlugChange = (value: string) => {
    setSlugTouched(true)
    setSlug(value)
  }

  const slugValid = slug === '' || SLUG_PATTERN.test(slug)
  const canSubmit = name.trim() !== '' && slug !== '' && slugValid && !mutation.isPending

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setApiError('')
    mutation.mutate({ name: name.trim(), slug })
  }

  return (
    <dialog
      ref={dialogRef}
      onClose={handleClose}
      className="backdrop:bg-black/50 rounded-lg shadow-xl border border-gray-200 p-0 w-full max-w-md"
    >
      <form onSubmit={handleSubmit} className="p-6">
        <h2 className="text-lg font-semibold text-gray-900">New Project</h2>
        <p className="mt-1 text-sm text-gray-500">Create a new feature flag project.</p>

        <div className="mt-4 space-y-4">
          <div>
            <label htmlFor="project-name" className="block text-sm font-medium text-gray-700">
              Name
            </label>
            <input
              ref={nameRef}
              id="project-name"
              type="text"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="My Project"
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label htmlFor="project-slug" className="block text-sm font-medium text-gray-700">
              Slug
            </label>
            <input
              id="project-slug"
              type="text"
              value={slug}
              onChange={(e) => handleSlugChange(e.target.value)}
              placeholder="my-project"
              className={`mt-1 block w-full rounded border px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${
                !slugValid ? 'border-red-400' : 'border-gray-300'
              }`}
            />
            {!slugValid && (
              <p className="mt-1 text-xs text-red-600">
                Slug must be lowercase letters, numbers, and hyphens only.
              </p>
            )}
          </div>
        </div>

        {apiError && (
          <p className="mt-3 text-xs text-red-600">{apiError}</p>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <button
            type="button"
            onClick={handleClose}
            className="px-3 py-2 text-sm text-gray-700 hover:text-gray-900"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!canSubmit}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {mutation.isPending ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </dialog>
  )
}
