import { useQuery } from '@tanstack/react-query'
import { useLocation, useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchJSON } from '../api'
import { useOpenCreateProjectDialog } from './CreateProjectDialog'
import { useBrand } from '../brand'
import { getUserManager } from '../auth'
import { Select, SelectItem } from './ui/Select'

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

/** Derive up to two initials from a full name string. Returns "?" if name is absent. */
function deriveInitials(name: string | undefined): string {
  if (!name) return '?'
  const words = name.trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) return '?'
  if (words.length === 1) return words[0].charAt(0).toUpperCase()
  return (words[0].charAt(0) + words[words.length - 1].charAt(0)).toUpperCase()
}

const NEW_PROJECT_SENTINEL = '__new__'

export function ProjectSwitcher() {
  const { t } = useTranslation('projects')
  const navigate = useNavigate()
  const { projectSlug, envSlug } = useActiveParams()
  const openCreateDialog = useOpenCreateProjectDialog()
  const { app_name, logo_url } = useBrand()
  const [initials, setInitials] = useState<string>('?')

  useEffect(() => {
    void getUserManager()
      .getUser()
      .then((user) => {
        if (user) {
          setInitials(deriveInitials(user.profile.name))
        }
      })
  }, [])

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

  const handleEnvChange = (value: string) => {
    if (value && projectSlug) {
      void navigate({
        to: '/projects/$slug/environments/$envSlug/flags',
        params: { slug: projectSlug, envSlug: value },
      })
    }
  }

  const noEnvironments =
    !envsQuery.isLoading && !envsQuery.isError && (envsQuery.data?.length ?? 0) === 0

  return (
    <header className="h-14 shrink-0 flex items-center gap-3 px-4 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
      {/* Wordmark / logo */}
      <div className="flex items-center gap-2 mr-2">
        {logo_url !== null ? (
          <img src={logo_url} alt={app_name} className="h-6 w-auto" />
        ) : (
          <span
            className="text-sm font-semibold text-[var(--color-text-primary)]"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            {app_name}
          </span>
        )}
      </div>

      <div className="w-px h-5 bg-[var(--color-border-hover)]" aria-hidden="true" />

      {/* Project switcher */}
      <div className="flex items-center gap-2">
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
          <Select
            value={projectSlug ?? ''}
            onValueChange={handleProjectChange}
            placeholder={t('switcher.select_project')}
            aria-label={t('switcher.project_label')}
          >
            {projectsQuery.data?.map((p) => (
              <SelectItem key={p.id} value={p.slug}>
                {p.name}
              </SelectItem>
            ))}
            <SelectItem value={NEW_PROJECT_SENTINEL}>
              {t('switcher.new_project_option')}
            </SelectItem>
          </Select>
        )}
      </div>

      {/* Environment switcher — only when inside a project */}
      {projectSlug !== null && (
        <>
          <span className="text-[var(--color-text-muted)]" aria-hidden="true">/</span>
          <div className="flex items-center gap-2">
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
            ) : noEnvironments ? (
              <button
                onClick={() =>
                  void navigate({ to: '/projects/$slug/settings/environments', params: { slug: projectSlug } })
                }
                className="text-xs text-[var(--color-accent)] hover:text-[var(--color-accent)] flex items-center gap-1"
              >
                {t('switcher.no_environments_nudge')}
                <span aria-hidden="true">→</span>
              </button>
            ) : (
              <Select
                value={envSlug ?? ''}
                onValueChange={handleEnvChange}
                placeholder={t('switcher.select_environment')}
                aria-label={t('switcher.environment_label')}
              >
                {envsQuery.data?.map((e) => (
                  <SelectItem key={e.id} value={e.slug}>
                    {e.name}
                  </SelectItem>
                ))}
              </Select>
            )}
          </div>
        </>
      )}

      {/* Spacer */}
      <div className="flex-1" aria-hidden="true" />

      {/* User avatar — initials derived from OIDC profile.name; no API call */}
      <div
        className="h-8 w-8 rounded-full flex items-center justify-center text-xs font-semibold text-[var(--color-text-primary)] select-none shrink-0"
        style={{
          background: 'linear-gradient(135deg, var(--color-accent-start), var(--color-accent-end))',
        }}
        aria-label={`User avatar: ${initials}`}
      >
        {initials}
      </div>
    </header>
  )
}
