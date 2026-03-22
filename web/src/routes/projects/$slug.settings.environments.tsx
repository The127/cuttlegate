import { createRoute, useLocation, useNavigate } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, postJSON, deleteRequest, APIError } from '../../api'
import { formatRelativeDate } from '../../utils/date'

interface Environment {
  id: string
  project_id: string
  name: string
  slug: string
  created_at: string
}

export const environmentSettingsRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/settings/environments',
  component: EnvironmentSettingsPage,
})

function useActiveEnvSlug(): string | null {
  const { pathname } = useLocation()
  const match = /^\/projects\/[^/]+\/environments\/([^/]+)/.exec(pathname)
  return match?.[1] ?? null
}

function EnvironmentSettingsPage() {
  const { t } = useTranslation('projects')
  const { slug } = environmentSettingsRoute.useParams()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const activeEnvSlug = useActiveEnvSlug()
  const queryKey = ['environments', slug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${slug}/environments`,
      ).then((d) => d.environments),
  })

  const deleteMutation = useMutation({
    mutationFn: (envSlug: string) =>
      deleteRequest(`/api/v1/projects/${slug}/environments/${envSlug}`),
    onSuccess: (_, envSlug) => {
      void queryClient.invalidateQueries({ queryKey })
      if (envSlug === activeEnvSlug) {
        void navigate({ to: '/projects/$slug', params: { slug } })
      }
    },
  })

  const [showCreate, setShowCreate] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Environment | null>(null)

  if (isLoading) return <EnvironmentListSkeleton />
  if (isError)
    return (
      <div className="p-6">
        <span className="text-sm text-red-600">{t('environments.error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  const environments = data ?? []

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-gray-900">{t('environments.title')}</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {t('environments.new_button')}
        </button>
      </div>

      {environments.length === 0 ? (
        <EnvironmentEmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <ul className="divide-y divide-gray-100 border border-gray-200 rounded-lg bg-white">
          {environments.map((env) => (
            <EnvironmentRow
              key={env.id}
              environment={env}
              onDeleteIntent={() => setPendingDelete(env)}
            />
          ))}
        </ul>
      )}

      {showCreate && (
        <CreateEnvironmentModal
          slug={slug}
          onCreated={() => {
            setShowCreate(false)
            void queryClient.invalidateQueries({ queryKey })
          }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      {pendingDelete && (
        <DeleteEnvironmentModal
          environment={pendingDelete}
          isDeleting={deleteMutation.isPending}
          deleteFailed={deleteMutation.isError}
          onConfirm={() => {
            deleteMutation.mutate(pendingDelete.slug, {
              onSuccess: () => setPendingDelete(null),
            })
          }}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </div>
  )
}

function EnvironmentRow({
  environment,
  onDeleteIntent,
}: {
  environment: Environment
  onDeleteIntent: () => void
}) {
  const { t } = useTranslation('projects')
  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-sm text-gray-800 bg-gray-50 border border-gray-200 rounded px-2 py-0.5 shrink-0">
          {environment.slug}
        </span>
        <span className="text-sm text-gray-700 truncate">{environment.name}</span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <time
          dateTime={environment.created_at}
          className="text-xs text-gray-400"
          title={new Date(environment.created_at).toLocaleString()}
        >
          {formatRelativeDate(environment.created_at)}
        </time>
        <button
          onClick={onDeleteIntent}
          aria-label={t('environments.delete_aria', { slug: environment.slug })}
          className="text-gray-400 hover:text-red-600 transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 rounded p-0.5"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="w-4 h-4"
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </div>
    </li>
  )
}

function EnvironmentEmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-gray-500">
        {t('environments.empty')}
      </p>
      <button
        onClick={onCreateClick}
        className="mt-4 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        {t('environments.new_button')}
      </button>
    </div>
  )
}

const SLUG_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_SLUG_LENGTH = 128

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/^-+|-+$/g, '')
}

function validateSlug(slug: string, t: (k: string, opts?: Record<string, unknown>) => string): string | null {
  if (slug.length === 0) return null
  if (slug.length > MAX_SLUG_LENGTH) return t('environments.slug_too_long', { max: MAX_SLUG_LENGTH })
  if (!SLUG_RE.test(slug)) return t('environments.slug_invalid')
  return null
}

function useEscapeKey(handler: () => void) {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') handler()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [handler])
}

function Modal({
  labelledBy,
  onClose,
  children,
}: {
  labelledBy: string
  onClose: () => void
  children: React.ReactNode
}) {
  useEscapeKey(onClose)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={labelledBy}
    >
      <div className="absolute inset-0 bg-black/30" onClick={onClose} aria-hidden="true" />
      {children}
    </div>
  )
}

function CreateEnvironmentModal({
  slug,
  onCreated,
  onCancel,
}: {
  slug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  const [name, setName] = useState('')
  const [envSlug, setEnvSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [slugError, setSlugError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () =>
      postJSON(`/api/v1/projects/${slug}/environments`, { name, slug: envSlug }),
    onSuccess: () => onCreated(),
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setSlugError(t('environments.slug_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setSlugError(err.message)
          return
        }
      }
      setServerError(
        err instanceof APIError ? err.message : t('environments.server_error'),
      )
    },
  })

  function handleNameChange(value: string) {
    setName(value)
    if (!slugTouched) {
      const generated = slugify(value)
      setEnvSlug(generated)
      setSlugError(validateSlug(generated, t))
    }
  }

  function handleSlugChange(value: string) {
    setEnvSlug(value)
    setSlugTouched(true)
    setSlugError(null)
    setServerError(null)
  }

  function handleSlugBlur() {
    setSlugError(validateSlug(envSlug, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const err = validateSlug(envSlug, t)
    if (err) {
      setSlugError(err)
      return
    }
    if (!envSlug) {
      setSlugError(t('environments.slug_required'))
      return
    }
    setServerError(null)
    createMutation.mutate()
  }

  return (
    <Modal labelledBy="create-env-title" onClose={onCancel}>
      <div className="relative bg-white rounded-lg shadow-lg max-w-md w-full mx-4 p-6">
        <h2 id="create-env-title" className="text-base font-semibold text-gray-900 mb-4">
          {t('environments.create_title')}
        </h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="env-name" className="block text-xs font-medium text-gray-500 mb-1">
              {t('environments.name_label')}
            </label>
            <input
              id="env-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder={t('environments.name_placeholder')}
              className="w-full text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label htmlFor="env-slug" className="block text-xs font-medium text-gray-500 mb-1">
              {t('environments.slug_label')}
            </label>
            <input
              id="env-slug"
              type="text"
              value={envSlug}
              onChange={(e) => handleSlugChange(e.target.value)}
              onBlur={handleSlugBlur}
              placeholder={t('environments.slug_placeholder')}
              aria-invalid={!!slugError}
              aria-describedby={slugError ? 'env-slug-error' : undefined}
              className={`w-full font-mono text-sm border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
                slugError
                  ? 'border-red-300 focus:ring-red-500'
                  : 'border-gray-300 focus:ring-blue-500'
              }`}
            />
            {slugError && (
              <p id="env-slug-error" className="mt-1 text-xs text-red-600">
                {slugError}
              </p>
            )}
          </div>
          {serverError && <p className="text-xs text-red-600">{serverError}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onCancel}
              disabled={createMutation.isPending}
              className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
            >
              {t('actions.cancel', { ns: 'common' })}
            </button>
            <button
              type="submit"
              disabled={createMutation.isPending || !!slugError || !name.trim() || !envSlug}
              className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {createMutation.isPending ? t('environments.creating') : t('environments.create_button')}
            </button>
          </div>
        </form>
      </div>
    </Modal>
  )
}

function DeleteEnvironmentModal({
  environment,
  isDeleting,
  deleteFailed,
  onConfirm,
  onCancel,
}: {
  environment: Environment
  isDeleting: boolean
  deleteFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  return (
    <Modal labelledBy="delete-env-title" onClose={onCancel}>
      <div className="relative bg-white rounded-lg shadow-lg max-w-sm w-full mx-4 p-6">
        <h2 id="delete-env-title" className="text-base font-semibold text-gray-900">
          {t('environments.delete_title')}
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          {t('environments.delete_body', { slug: environment.slug })}
        </p>
        <p className="mt-2 text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded px-3 py-2">
          {t('environments.delete_warning')}
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-red-600">{t('environments.delete_failed')}</p>
        )}
        <div className="mt-5 flex justify-end gap-3">
          <button
            autoFocus
            onClick={onCancel}
            disabled={isDeleting}
            className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
          >
            {t('actions.cancel', { ns: 'common' })}
          </button>
          <button
            onClick={onConfirm}
            disabled={isDeleting}
            className="px-3 py-1.5 text-sm font-medium bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-red-500"
          >
            {isDeleting ? t('environments.deleting') : t('environments.delete_button')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

function EnvironmentListSkeleton() {
  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-gray-100 rounded animate-pulse" />
        <div className="h-8 w-40 bg-gray-100 rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-gray-100 border border-gray-200 rounded-lg bg-white">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-gray-100 rounded animate-pulse" />
              <div className="h-4 w-40 bg-gray-100 rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-2">
              <div className="h-4 w-12 bg-gray-100 rounded animate-pulse" />
              <div className="h-4 w-4 bg-gray-100 rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
