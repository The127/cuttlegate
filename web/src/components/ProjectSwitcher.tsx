import { useQuery } from '@tanstack/react-query'
import { useLocation, useNavigate } from '@tanstack/react-router'
import { fetchJSON } from '../api'

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

export function ProjectSwitcher() {
  const navigate = useNavigate()
  const { projectSlug, envSlug } = useActiveParams()

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

  return (
    <div className="flex items-center gap-3 px-4 py-2 border-b border-gray-200 bg-white">
      {/* Project selector */}
      <div className="flex items-center gap-2">
        <label htmlFor="project-select" className="text-xs font-medium text-gray-500 uppercase tracking-wide">
          Project
        </label>
        {projectsQuery.isLoading ? (
          <div className="h-8 w-32 bg-gray-100 rounded animate-pulse" />
        ) : projectsQuery.isError ? (
          <span className="text-xs text-red-600 flex items-center gap-1">
            Failed to load
            <button
              onClick={() => void projectsQuery.refetch()}
              className="underline hover:no-underline"
            >
              retry
            </button>
          </span>
        ) : projectsQuery.data?.length === 0 ? (
          <span className="text-xs text-gray-500">
            No projects yet —{' '}
            <a href="/projects/new" className="text-blue-600 hover:text-blue-800">
              create one
            </a>
          </span>
        ) : (
          <select
            id="project-select"
            value={projectSlug ?? ''}
            onChange={(e) => {
              if (e.target.value) {
                void navigate({ to: '/projects/$slug', params: { slug: e.target.value } })
              }
            }}
            className="text-sm border border-gray-200 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Select project…</option>
            {projectsQuery.data?.map((p) => (
              <option key={p.id} value={p.slug}>
                {p.name}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Environment selector — shown only when a project is active */}
      {projectSlug !== null && (
        <div className="flex items-center gap-2">
          <span className="text-gray-300">/</span>
          <label htmlFor="env-select" className="text-xs font-medium text-gray-500 uppercase tracking-wide">
            Environment
          </label>
          {envsQuery.isLoading ? (
            <div className="h-8 w-28 bg-gray-100 rounded animate-pulse" />
          ) : envsQuery.isError ? (
            <span className="text-xs text-red-600 flex items-center gap-1">
              Failed to load
              <button
                onClick={() => void envsQuery.refetch()}
                className="underline hover:no-underline"
              >
                retry
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
              className="text-sm border border-gray-200 rounded px-2 py-1 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Select environment…</option>
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
