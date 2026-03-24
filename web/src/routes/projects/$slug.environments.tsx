import { createRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON } from '../../api'
import { Button } from '../../components/ui'
import { CreateEnvironmentDialog } from '../../components/CreateEnvironmentDialog'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'

interface Environment {
  id: string
  name: string
  slug: string
}

interface EnvFlag {
  id: string
  key: string
  name: string
  enabled: boolean
}

export const environmentsOverviewRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/environments',
  component: EnvironmentsOverviewPage,
})

function EnvironmentsOverviewPage() {
  const { t } = useTranslation('projects')
  const project = projectRoute.useLoaderData()
  useDocumentTitle(t('environments_overview.title'), project.name)

  const envsQuery = useQuery({
    queryKey: ['environments', project.slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${project.slug}/environments`,
      ).then((d) => d.environments),
  })

  const [showCreate, setShowCreate] = useState(false)

  return (
    <div className="p-6 max-w-5xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">
          {t('environments_overview.title')}
        </h1>
        <Button variant="primary" onClick={() => setShowCreate(true)}>
          {t('environments_overview.create_button')}
        </Button>
      </div>

      {envsQuery.isLoading ? (
        <EnvironmentCardsSkeleton />
      ) : envsQuery.isError ? (
        <ErrorState onRetry={() => void envsQuery.refetch()} />
      ) : envsQuery.data!.length === 0 ? (
        <EmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {envsQuery.data!.map((env) => (
            <EnvironmentCard key={env.id} env={env} projectSlug={project.slug} />
          ))}
        </div>
      )}

      <CreateEnvironmentDialog
        open={showCreate}
        projectSlug={project.slug}
        onCreated={() => setShowCreate(false)}
        onCancel={() => setShowCreate(false)}
      />
    </div>
  )
}

function EnvironmentCard({ env, projectSlug }: { env: Environment; projectSlug: string }) {
  const { t } = useTranslation('projects')
  const flagsQuery = useQuery({
    queryKey: ['flags', projectSlug, env.slug],
    queryFn: () =>
      fetchJSON<{ flags: EnvFlag[] }>(
        `/api/v1/projects/${projectSlug}/environments/${env.slug}/flags`,
      ).then((d) => d.flags),
  })

  return (
    <Link
      to="/projects/$slug/environments/$envSlug/flags"
      params={{ slug: projectSlug, envSlug: env.slug }}
      className="block border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] p-4 hover:border-[var(--color-border-hover)] hover:shadow-[0_4px_20px_rgba(0,0,0,0.3)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      <div className="flex items-center gap-2">
        <h3 className="font-mono text-sm font-medium text-[var(--color-text-primary)]">{env.name}</h3>
        <span className="text-xs font-mono text-[var(--color-text-secondary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5">
          {env.slug}
        </span>
      </div>
      {flagsQuery.isLoading ? (
        <div className="mt-2 h-4 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
      ) : flagsQuery.isError ? (
        <p className="mt-2 text-xs text-[var(--color-status-error)]">
          {t('environments_overview.flag_count_error')}
        </p>
      ) : (
        <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
          {t('environments_overview.flag_count', { count: flagsQuery.data!.length })}
        </p>
      )}
    </Link>
  )
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-16 px-6">
      <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
        {t('environments_overview.empty_title')}
      </h2>
      <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
        {t('environments_overview.empty_body')}
      </p>
      <Button size="lg" variant="primary" className="mt-4" onClick={onCreateClick}>
        {t('environments_overview.create_first_cta')}
      </Button>
    </div>
  )
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="py-12 px-6 text-center">
      <p className="text-sm text-[var(--color-status-error)]">
        {t('environments_overview.error')}
      </p>
      <Button variant="secondary" className="mt-3" onClick={onRetry}>
        {t('actions.retry', { ns: 'common' })}
      </Button>
    </div>
  )
}

function EnvironmentCardsSkeleton() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      {[1, 2, 3].map((i) => (
        <div key={i} className="border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] p-4">
          <div className="flex items-center gap-2">
            <div className="h-4 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-4 w-14 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
          <div className="mt-2 h-4 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </div>
      ))}
    </div>
  )
}
