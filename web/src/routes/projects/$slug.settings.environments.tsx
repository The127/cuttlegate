import { createRoute, useLocation, useNavigate } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, postJSON, deleteRequest, APIError } from '../../api'
import { formatRelativeDate } from '../../utils/date'
import { Button } from '../../components/ui/Button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../../components/ui/Dialog'

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
        <span className="text-sm text-red-600 dark:text-red-400">{t('environments.error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  const environments = data ?? []

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('environments.title')}</h1>
        <Button onClick={() => setShowCreate(true)}>
          {t('environments.new_button')}
        </Button>
      </div>

      {environments.length === 0 ? (
        <EnvironmentEmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
          {environments.map((env) => (
            <EnvironmentRow
              key={env.id}
              environment={env}
              onDeleteIntent={() => setPendingDelete(env)}
            />
          ))}
        </ul>
      )}

      <CreateEnvironmentModal
        open={showCreate}
        slug={slug}
        onCreated={() => {
          setShowCreate(false)
          void queryClient.invalidateQueries({ queryKey })
        }}
        onCancel={() => setShowCreate(false)}
      />

      <DeleteEnvironmentModal
        open={pendingDelete !== null}
        environment={pendingDelete}
        isDeleting={deleteMutation.isPending}
        deleteFailed={deleteMutation.isError}
        onConfirm={() => {
          if (!pendingDelete) return
          deleteMutation.mutate(pendingDelete.slug, {
            onSuccess: () => setPendingDelete(null),
          })
        }}
        onCancel={() => setPendingDelete(null)}
      />
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
        <span className="font-mono text-sm text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-2 py-0.5 shrink-0">
          {environment.slug}
        </span>
        <span className="text-sm text-gray-700 dark:text-gray-300 truncate">{environment.name}</span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <time
          dateTime={environment.created_at}
          className="text-xs text-gray-400 dark:text-gray-500"
          title={new Date(environment.created_at).toLocaleString()}
        >
          {formatRelativeDate(environment.created_at)}
        </time>
        <button
          onClick={onDeleteIntent}
          aria-label={t('environments.delete_aria', { slug: environment.slug })}
          className="text-gray-400 dark:text-gray-500 hover:text-red-600 dark:hover:text-red-400 transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 rounded p-0.5"
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
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t('environments.empty')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('environments.new_button')}
      </Button>
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


function CreateEnvironmentModal({
  open,
  slug,
  onCreated,
  onCancel,
}: {
  open: boolean
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

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('environments.create_title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="env-name" className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
              {t('environments.name_label')}
            </label>
            <input
              id="env-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder={t('environments.name_placeholder')}
              className="w-full text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            />
          </div>
          <div>
            <label htmlFor="env-slug" className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
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
              className={`w-full font-mono text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
                slugError
                  ? 'border-red-300 focus:ring-red-500'
                  : 'border-gray-300 dark:border-gray-600 focus:ring-[var(--color-accent)]'
              }`}
            />
            {slugError && (
              <p id="env-slug-error" className="mt-1 text-xs text-red-600 dark:text-red-400">
                {slugError}
              </p>
            )}
          </div>
          {serverError && <p className="text-xs text-red-600 dark:text-red-400">{serverError}</p>}
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={createMutation.isPending}
            >
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button
              type="submit"
              loading={createMutation.isPending}
              disabled={!!slugError || !name.trim() || !envSlug}
            >
              {t('environments.create_button')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function DeleteEnvironmentModal({
  open,
  environment,
  isDeleting,
  deleteFailed,
  onConfirm,
  onCancel,
}: {
  open: boolean
  environment: Environment | null
  isDeleting: boolean
  deleteFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('environments.delete_title')}</DialogTitle>
          {environment && (
            <DialogDescription>
              {t('environments.delete_body', { slug: environment.slug })}
            </DialogDescription>
          )}
        </DialogHeader>
        <p className="text-sm text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-950 border border-amber-200 dark:border-amber-800 rounded px-3 py-2">
          {t('environments.delete_warning')}
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-red-600 dark:text-red-400">{t('environments.delete_failed')}</p>
        )}
        <DialogFooter>
          <Button
            autoFocus
            type="button"
            variant="secondary"
            onClick={onCancel}
            disabled={isDeleting}
          >
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={isDeleting}
            onClick={onConfirm}
          >
            {t('environments.delete_button')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function EnvironmentListSkeleton() {
  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        <div className="h-8 w-40 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
              <div className="h-4 w-40 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-2">
              <div className="h-4 w-12 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
              <div className="h-4 w-4 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
