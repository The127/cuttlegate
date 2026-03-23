import { createRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { authenticatedRoute } from './_authenticated'
import { fetchJSON } from '../api'
import { useOpenCreateProjectDialog } from '../components/CreateProjectDialog'
import { Button } from '../components/ui/Button'
import { useDocumentTitle } from '../hooks/useDocumentTitle'

interface Project {
  id: string
  name: string
  slug: string
  created_at: string
}

export const indexRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/',
  component: HomePage,
})

const COUNT_STALE_TIME = 60_000

function useProjectCount<T>(slug: string, resource: string, key: string) {
  return useQuery({
    queryKey: [resource, slug, 'count'],
    queryFn: () =>
      fetchJSON<Record<string, T[]>>(
        `/api/v1/projects/${slug}/${resource}`,
      ).then((d) => d[key].length),
    staleTime: COUNT_STALE_TIME,
  })
}

function HomePage() {
  const { t } = useTranslation('projects')
  const openCreateDialog = useOpenCreateProjectDialog()
  useDocumentTitle(t('list.title'))
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['projects'],
    queryFn: () =>
      fetchJSON<{ projects: Project[] }>('/api/v1/projects').then((d) => d.projects),
  })

  if (isLoading) return <HomePageSkeleton />

  if (isError) {
    return (
      <div className="p-6 text-center">
        <p className="text-sm text-[var(--color-status-error)]">{t('list.error')}</p>
        <button
          onClick={() => void refetch()}
          className="mt-2 text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )
  }

  const projects = data ?? []

  if (projects.length === 0) {
    return (
      <div className="p-6 text-center">
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('list.empty_title')}</h1>
        <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
          {t('list.empty_body')}
        </p>
        <Button onClick={openCreateDialog} size="lg" className="mt-4">
          {t('list.first_project_button')}
        </Button>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('list.title')}</h1>
        <Button onClick={openCreateDialog} size="md">
          {t('list.new_project')}
        </Button>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {projects.map((project) => (
          <ProjectCard key={project.id} project={project} />
        ))}
      </div>
    </div>
  )
}

function ProjectCard({ project }: { project: Project }) {
  const { t } = useTranslation('projects')
  const envsCount = useProjectCount(project.slug, 'environments', 'environments')
  const flagsCount = useProjectCount(project.slug, 'flags', 'flags')
  const membersCount = useProjectCount(project.slug, 'members', 'members')

  return (
    <Link
      to="/projects/$slug"
      params={{ slug: project.slug }}
      className="block border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] p-4 hover:border-[var(--color-border-hover)] hover:shadow-[0_4px_20px_rgba(0,0,0,0.3)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">{project.name}</h2>
      <p className="mt-0.5 font-mono text-xs text-[var(--color-text-secondary)]">{project.slug}</p>
      <div className="mt-3 flex items-center gap-2 text-xs text-[var(--color-text-secondary)]">
        <CountBadge label={t('list.count_environments')} count={envsCount.data} isLoading={envsCount.isLoading} isError={envsCount.isError} />
        <CountBadge label={t('list.count_flags')} count={flagsCount.data} isLoading={flagsCount.isLoading} isError={flagsCount.isError} />
        <CountBadge label={t('list.count_members')} count={membersCount.data} isLoading={membersCount.isLoading} isError={membersCount.isError} />
      </div>
    </Link>
  )
}

function CountBadge({
  label,
  count,
  isLoading,
  isError,
}: {
  label: string
  count: number | undefined
  isLoading: boolean
  isError: boolean
}) {
  return (
    <span className="flex items-center gap-1 bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5">
      {isLoading ? (
        <span className="inline-block h-3.5 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
      ) : isError ? (
        <span className="text-[var(--color-status-error)]">-</span>
      ) : (
        <span className="font-medium text-[var(--color-text-primary)]">{count ?? 0}</span>
      )}
      <span>{label}</span>
    </span>
  )
}

function HomePageSkeleton() {
  return (
    <div className="p-6">
      <div className="h-6 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-6" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] p-4"
          >
            <div className="h-4 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="mt-1.5 h-3 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="mt-3 flex items-center gap-4">
              <div className="h-5 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-5 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-5 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
