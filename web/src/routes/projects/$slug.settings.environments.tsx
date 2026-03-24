import { createRoute, useLocation, useNavigate } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, deleteRequest, patchJSON } from '../../api'
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
import { CreateEnvironmentDialog } from '../../components/CreateEnvironmentDialog'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { SettingsTabBar } from '../../components/SettingsTabBar'
import { useProjectRole } from '../../hooks/useProjectRole'

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
  const project = projectRoute.useLoaderData()
  useDocumentTitle(t('environments.title'), t('settings.title'), project.name)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const activeEnvSlug = useActiveEnvSlug()
  const { data: role } = useProjectRole(slug)
  const canEdit = role === 'admin'
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
        <span className="text-sm text-[var(--color-status-error)]">{t('environments.error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  const environments = data ?? []

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-xl font-semibold text-[var(--color-text-primary)] mb-6">{t('settings.title')}</h1>
      <SettingsTabBar slug={slug} />

      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('environments.title')}</h2>
        <Button onClick={() => setShowCreate(true)}>
          {t('environments.new_button')}
        </Button>
      </div>

      {environments.length === 0 ? (
        <EnvironmentEmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
          {environments.map((env) => (
            <EnvironmentRow
              key={env.id}
              environment={env}
              projectSlug={slug}
              canEdit={canEdit}
              onDeleteIntent={() => setPendingDelete(env)}
            />
          ))}
        </ul>
      )}

      <CreateEnvironmentDialog
        open={showCreate}
        projectSlug={slug}
        onCreated={() => setShowCreate(false)}
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
  projectSlug,
  canEdit,
  onDeleteIntent,
}: {
  environment: Environment
  projectSlug: string
  canEdit: boolean
  onDeleteIntent: () => void
}) {
  const { t } = useTranslation('projects')
  const queryClient = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(environment.name)
  const [validationError, setValidationError] = useState(false)
  const [renameError, setRenameError] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isEditing])

  const renameMutation = useMutation({
    mutationFn: (name: string) =>
      patchJSON<Environment>(
        `/api/v1/projects/${projectSlug}/environments/${environment.slug}`,
        { name },
      ),
    onSuccess: () => {
      setIsEditing(false)
      setRenameError(false)
      void queryClient.invalidateQueries({ queryKey: ['environments', projectSlug] })
    },
    onError: () => {
      setIsEditing(false)
      setRenameError(true)
      setEditValue(environment.name)
      setTimeout(() => setRenameError(false), 3000)
    },
  })

  function startEdit() {
    setEditValue(environment.name)
    setValidationError(false)
    setRenameError(false)
    setIsEditing(true)
  }

  function saveEdit() {
    const trimmed = editValue.trim()
    if (trimmed === '') {
      setValidationError(true)
      return
    }
    if (trimmed === environment.name) {
      setIsEditing(false)
      return
    }
    renameMutation.mutate(trimmed)
  }

  function cancelEdit() {
    setEditValue(environment.name)
    setValidationError(false)
    setIsEditing(false)
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      saveEdit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      cancelEdit()
    }
  }

  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5 shrink-0">
          {environment.slug}
        </span>
        {isEditing ? (
          <input
            ref={inputRef}
            type="text"
            value={editValue}
            onChange={(e) => {
              setEditValue(e.target.value)
              if (validationError && e.target.value.trim() !== '') {
                setValidationError(false)
              }
            }}
            onBlur={saveEdit}
            onKeyDown={handleKeyDown}
            className={`text-sm text-[var(--color-text-primary)] bg-[var(--color-surface)] border rounded px-2 py-0.5 min-w-0 focus:outline-none focus:ring-2 ${
              validationError
                ? 'border-[var(--color-status-error)] focus:ring-[var(--color-status-error)]'
                : 'border-[var(--color-border)] focus:ring-[var(--color-accent)]'
            }`}
            aria-invalid={validationError}
            aria-label={t('environments.edit_aria', { slug: environment.slug })}
          />
        ) : (
          <span className="text-sm text-[var(--color-text-primary)] truncate">{environment.name}</span>
        )}
        {renameError && (
          <span className="text-xs text-[var(--color-status-error)]">{t('environments.rename_failed')}</span>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <time
          dateTime={environment.created_at}
          className="text-xs text-[var(--color-text-muted)]"
          title={new Date(environment.created_at).toLocaleString()}
        >
          {formatRelativeDate(environment.created_at)}
        </time>
        {canEdit && !isEditing && (
          <button
            onClick={startEdit}
            aria-label={t('environments.edit_aria', { slug: environment.slug })}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded p-0.5"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-4 h-4"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
            </svg>
          </button>
        )}
        <button
          onClick={onDeleteIntent}
          aria-label={t('environments.delete_aria', { slug: environment.slug })}
          className="text-[var(--color-text-muted)] hover:text-[var(--color-status-error)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded p-0.5"
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
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('environments.empty')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('environments.new_button')}
      </Button>
    </div>
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
        <p className="text-sm text-[var(--color-status-warning)] bg-[rgba(251,191,36,0.08)] border border-[var(--color-status-warning)] rounded px-3 py-2">
          {t('environments.delete_warning')}
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-[var(--color-status-error)]">{t('environments.delete_failed')}</p>
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
    <div className="p-6 max-w-5xl">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        <div className="h-8 w-40 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-40 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-2">
              <div className="h-4 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
