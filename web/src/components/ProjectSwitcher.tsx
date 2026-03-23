import { useQuery } from '@tanstack/react-query'
import { useLocation, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { fetchJSON } from '../api'
import { useOpenCreateProjectDialog } from './CreateProjectDialog'
import { useBrand } from '../brand'

interface Project {
  id: string
  name: string
  slug: string
  created_at: string
}

interface Environment {
  id: string
  project_id: string
  name: string
  slug: string
  created_at: string
}

function useActiveParams() {
  const { pathname } = useLocation()
  const projectMatch = /^\/projects\/([^/]+)/.exec(pathname)
  const envMatch = /^\/projects\/[^/]+\/environments\/([^/]+)/.exec(pathname)
  return {
    projectSlug: projectMatch?.[1] ?? null,
    envSlug: envMatch?.[1] ?? null,
  }
}

const NEW_PROJECT_SENTINEL = '__new__'

export function ProjectSwitcher() {
  const { t } = useTranslation('projects')
  const navigate = useNavigate()
  const { projectSlug, envSlug } = useActiveParams()
  const openCreateDialog = useOpenCreateProjectDialog()
  const { app_name, logo_url } = useBrand()

  const projectsQuery = useQuery({
    queryKey: ['projects'],
    queryFn: () => fetchJSON<{ projects: Project[] }>('/api/v1/projects').then((d) => d.projects),
  })

  const envsQuery = useQuery({
    queryKey: ['environments', projectSlug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${projectSlug}/environments`,
      ).then((d) => d.environments),
    enabled: projectSlug !== null,
  })

  const handleProjectChange = (value: string) => {
    if (value === NEW_PROJECT_SENTINEL) {
      openCreateDialog()
      return
    }
    if (value) {
      void navigate({ to: '/projects/$slug', params: { slug: value } })
    }
  }

  return (
    <div className="flex items-center gap-3 px-4 py-2 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center gap-2 mr-2">
        {logo_url !== null ? (
          <img src={logo_url} alt={app_name} className="h-6 w-auto" />
        ) : (
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">{app_name}</span>
        )}
      </div>
      <div className="w-px h-5 bg-[var(--color-surface-elevated)]" aria-hidden="true" />
      <div className="flex items-center gap-2">
        <label htmlFor="project-select" className="text-xs font-medium text-[var(--color-text-secondary)] font-medium">
          {t('switcher.project_label')}
        </label>
        {projectsQuery.isLoading ? (
          <div className="h-8 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        ) : projectsQuery.isError ? (
          <span className="text-xs text-[var(--color-status-error)] flex items-center gap-1">
            {t('switcher.failed_to_load')}
            <button
              onClick={() => void projectsQuery.refetch()}
              className="underline hover:no-underline"
            >
              {t('switcher.retry')}
            </button>
          </span>
        ) : projectsQuery.data?.length === 0 ? (
          <span className="text-xs text-[var(--color-text-secondary)]">
            {t('switcher.no_projects_prefix')}{' '}
            <button
              onClick={openCreateDialog}
              className="text-[var(--color-accent)] hover:text-[var(--color-accent)]"
            >
              {t('switcher.create_one')}
            </button>
          </span>
        ) : (
          <select
            id="project-select"
            value={projectSlug ?? ''}
            onChange={(e) => handleProjectChange(e.target.value)}
            className="text-sm border border-[var(--color-border)] rounded px-2 py-1 bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
          >
            <option value="">{t('switcher.select_project')}</option>
            {projectsQuery.data?.map((p) => (
              <option key={p.id} value={p.slug}>
                {p.name}
              </option>
            ))}
            <option value={NEW_PROJECT_SENTINEL}>{t('switcher.new_project_option')}</option>
          </select>
        )}
      </div>

      {projectSlug !== null && (
        <div className="flex items-center gap-2">
          <span className="text-[var(--color-text-muted)]">/</span>
          <label htmlFor="env-select" className="text-xs font-medium text-[var(--color-text-secondary)] font-medium">
            {t('switcher.environment_label')}
          </label>
          {envsQuery.isLoading ? (
            <div className="h-8 w-28 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          ) : envsQuery.isError ? (
            <span className="text-xs text-[var(--color-status-error)] flex items-center gap-1">
              {t('switcher.failed_to_load')}
              <button
                onClick={() => void envsQuery.refetch()}
                className="underline hover:no-underline"
              >
                {t('switcher.retry')}
              </button>
            </span>
          ) : (
            <select
              id="env-select"
              value={envSlug ?? ''}
              onChange={(e) => {
                if (e.target.value && projectSlug) {
                  void navigate({
                    to: '/projects/$slug/environments/$envSlug/flags',
                    params: { slug: projectSlug, envSlug: e.target.value },
                  })
                }
              }}
              className="text-sm border border-[var(--color-border)] rounded px-2 py-1 bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            >
              <option value="">{t('switcher.select_environment')}</option>
              {envsQuery.data?.map((e) => (
                <option key={e.id} value={e.slug}>
                  {e.name}
                </option>
              ))}
            </select>
          )}
        </div>
      )}
    </div>
  )
}
