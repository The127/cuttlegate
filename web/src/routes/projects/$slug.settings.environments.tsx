import { createRoute, useLocation, useNavigate } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, deleteRequest } from '../../api'
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
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">{t('environments.title')}</h1>
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
  onDeleteIntent,
}: {
  environment: Environment
  onDeleteIntent: () => void
}) {
  const { t } = useTranslation('projects')
  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5 shrink-0">
          {environment.slug}
        </span>
        <span className="text-sm text-[var(--color-text-primary)] truncate">{environment.name}</span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <time
          dateTime={environment.created_at}
          className="text-xs text-[var(--color-text-muted)]"
          title={new Date(environment.created_at).toLocaleString()}
        >
          {formatRelativeDate(environment.created_at)}
        </time>
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
    <div className="p-6 max-w-4xl">
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
