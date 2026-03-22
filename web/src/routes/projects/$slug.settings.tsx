import { createRoute, useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { projectRoute } from './$slug'
import { patchJSON, deleteRequest, APIError } from '../../api'
import { useProjectRole } from '../../hooks/useProjectRole'

export const projectSettingsRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/settings',
  component: ProjectSettingsPage,
})

function ProjectSettingsPage() {
  const project = projectRoute.useLoaderData()
  const roleQuery = useProjectRole(project.slug)

  if (roleQuery.isLoading) return <SettingsSkeleton />
  if (roleQuery.isError)
    return (
      <div className="p-6">
        <span className="text-sm text-red-600">Failed to load settings. </span>
        <button
          onClick={() => void roleQuery.refetch()}
          className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          Retry
        </button>
      </div>
    )

  const isAdmin = roleQuery.data === 'admin'

  return (
    <div className="p-6 max-w-2xl">
      <h1 className="text-2xl font-semibold text-gray-900 mb-6">Settings</h1>

      <GeneralSection project={project} isAdmin={isAdmin} />

      {isAdmin && (
        <>
          <div className="mt-8 border-t border-gray-200" />
          <DangerZone project={project} />
        </>
      )}
    </div>
  )
}

interface ProjectData {
  id: string
  name: string
  slug: string
  created_at: string
}

function GeneralSection({
  project,
  isAdmin,
}: {
  project: ProjectData
  isAdmin: boolean
}) {
  return (
    <section>
      <h2 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">General</h2>
      <div className="bg-white border border-gray-200 rounded-lg divide-y divide-gray-100">
        <NameField project={project} isAdmin={isAdmin} />
        <SlugField slug={project.slug} />
      </div>
    </section>
  )
}

function NameField({ project, isAdmin }: { project: ProjectData; isAdmin: boolean }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(project.name)
  const [saved, setSaved] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  const updateMutation = useMutation({
    mutationFn: () => patchJSON<ProjectData>(`/api/v1/projects/${project.slug}`, { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] })
      setSaved(true)
      setServerError(null)
      setTimeout(() => setSaved(false), 2000)
    },
    onError: (err) => {
      setServerError(err instanceof APIError ? err.message : 'Failed to save. Please try again.')
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || name === project.name) return
    setServerError(null)
    updateMutation.mutate()
  }

  return (
    <div className="px-4 py-4">
      <label htmlFor="project-name" className="block text-xs font-medium text-gray-500 mb-1">
        Project name
      </label>
      {isAdmin ? (
        <form onSubmit={handleSubmit} className="flex items-center gap-3">
          <input
            id="project-name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              setServerError(null)
              setSaved(false)
            }}
            className="flex-1 text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            disabled={updateMutation.isPending || !name.trim() || name === project.name}
            className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {updateMutation.isPending ? 'Saving…' : saved ? 'Saved' : 'Save'}
          </button>
        </form>
      ) : (
        <p className="text-sm text-gray-900">{project.name}</p>
      )}
      {serverError && <p className="mt-1 text-xs text-red-600">{serverError}</p>}
    </div>
  )
}

function SlugField({ slug }: { slug: string }) {
  const [copied, setCopied] = useState(false)

  function copySlug() {
    void navigator.clipboard
      .writeText(slug)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {})
  }

  return (
    <div className="px-4 py-4">
      <p className="text-xs font-medium text-gray-500 mb-1">Project slug</p>
      <div className="flex items-center gap-2">
        <span className="font-mono text-sm text-gray-700 bg-gray-50 border border-gray-200 rounded px-2 py-1">
          {slug}
        </span>
        <div className="relative">
          <button
            onClick={copySlug}
            className="text-xs text-gray-500 hover:text-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded px-2 py-1 border border-gray-200 hover:border-blue-300 transition-colors"
            aria-label={`Copy project slug ${slug}`}
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
        <p className="text-xs text-gray-400">Slug is immutable after creation.</p>
      </div>
    </div>
  )
}

function DangerZone({ project }: { project: ProjectData }) {
  const [showDeleteModal, setShowDeleteModal] = useState(false)

  return (
    <section className="mt-8">
      <h2 className="text-sm font-medium text-red-600 uppercase tracking-wide mb-4">Danger zone</h2>
      <div className="bg-white border border-red-200 rounded-lg px-4 py-4 flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-900">Delete this project</p>
          <p className="text-xs text-gray-500 mt-0.5">
            Permanently deletes all flags, environments, rules, members, and API keys.
          </p>
        </div>
        <button
          onClick={() => setShowDeleteModal(true)}
          className="px-3 py-1.5 text-sm font-medium text-red-600 border border-red-300 rounded hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500 shrink-0"
        >
          Delete project
        </button>
      </div>

      {showDeleteModal && (
        <DeleteProjectModal project={project} onCancel={() => setShowDeleteModal(false)} />
      )}
    </section>
  )
}

function DeleteProjectModal({
  project,
  onCancel,
}: {
  project: ProjectData
  onCancel: () => void
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirmName, setConfirmName] = useState('')
  const [serverError, setServerError] = useState<string | null>(null)

  const deleteMutation = useMutation({
    mutationFn: () => deleteRequest(`/api/v1/projects/${project.slug}`),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      void navigate({ to: '/' })
    },
    onError: (err) => {
      setServerError(err instanceof APIError ? err.message : 'Failed to delete. Please try again.')
    },
  })

  function handleDelete(e: React.FormEvent) {
    e.preventDefault()
    if (confirmName !== project.name) return
    setServerError(null)
    deleteMutation.mutate()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-project-title"
    >
      <div
        className="absolute inset-0 bg-black/30"
        onClick={onCancel}
        aria-hidden="true"
      />
      <div className="relative bg-white rounded-lg shadow-lg max-w-md w-full mx-4 p-6">
        <h2 id="delete-project-title" className="text-base font-semibold text-gray-900">
          Delete project?
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          This will permanently delete{' '}
          <span className="font-semibold text-gray-900">{project.name}</span> and everything in it:
        </p>
        <ul className="mt-2 text-sm text-gray-600 list-disc list-inside space-y-0.5">
          <li>Feature flags and all their targeting rules</li>
          <li>Environments and flag state per environment</li>
          <li>Project members and API keys</li>
        </ul>

        <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded text-xs text-red-700">
          This action cannot be undone.
        </div>

        <form onSubmit={handleDelete} className="mt-4 space-y-3">
          <div>
            <label htmlFor="confirm-name" className="block text-xs font-medium text-gray-700 mb-1">
              Type <span className="font-mono font-semibold">{project.name}</span> to confirm
            </label>
            <input
              id="confirm-name"
              type="text"
              value={confirmName}
              onChange={(e) => {
                setConfirmName(e.target.value)
                setServerError(null)
              }}
              autoFocus
              placeholder={project.name}
              className="w-full text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-red-500"
            />
          </div>
          {serverError && <p className="text-xs text-red-600">{serverError}</p>}
          <div className="flex justify-end gap-3 pt-1">
            <button
              type="button"
              onClick={onCancel}
              disabled={deleteMutation.isPending}
              className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={confirmName !== project.name || deleteMutation.isPending}
              className="px-3 py-1.5 text-sm font-medium bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-red-500"
            >
              {deleteMutation.isPending ? 'Deleting…' : 'Delete project'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function SettingsSkeleton() {
  return (
    <div className="p-6 max-w-2xl">
      <div className="h-8 w-24 bg-gray-100 rounded animate-pulse mb-6" />
      <div className="h-4 w-16 bg-gray-100 rounded animate-pulse mb-4" />
      <div className="bg-white border border-gray-200 rounded-lg divide-y divide-gray-100">
        <div className="px-4 py-4">
          <div className="h-3 w-20 bg-gray-100 rounded animate-pulse mb-2" />
          <div className="h-8 w-48 bg-gray-100 rounded animate-pulse" />
        </div>
        <div className="px-4 py-4">
          <div className="h-3 w-20 bg-gray-100 rounded animate-pulse mb-2" />
          <div className="h-7 w-32 bg-gray-100 rounded animate-pulse" />
        </div>
      </div>
    </div>
  )
}
