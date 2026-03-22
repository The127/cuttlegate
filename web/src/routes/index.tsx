import { createRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { authenticatedRoute } from './_authenticated'
import { fetchJSON } from '../api'

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
    queryKey: [resource, slug],
    queryFn: () =>
      fetchJSON<Record<string, T[]>>(
        `/api/v1/projects/${slug}/${resource}`,
      ).then((d) => d[key].length),
    staleTime: COUNT_STALE_TIME,
  })
}

function HomePage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['projects'],
    queryFn: () =>
      fetchJSON<{ projects: Project[] }>('/api/v1/projects').then((d) => d.projects),
  })

  if (isLoading) return <HomePageSkeleton />

  if (isError) {
    return (
      <div className="p-8 text-center">
        <p className="text-sm text-red-600">Failed to load projects.</p>
        <button
          onClick={() => void refetch()}
          className="mt-2 text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          Retry
        </button>
      </div>
    )
  }

  const projects = data ?? []

  if (projects.length === 0) {
    return (
      <div className="p-8 text-center">
        <h1 className="text-lg font-semibold text-gray-900">No projects yet</h1>
        <p className="mt-2 text-sm text-gray-600">
          Create your first project to get started with feature flags.
        </p>
        {/* CTA wired in #175 (Project creation UI) */}
        <p className="mt-4 text-sm font-medium text-blue-600">
          Create your first project
        </p>
      </div>
    )
  }

  return (
    <div className="p-8">
      <h1 className="text-lg font-semibold text-gray-900 mb-6">Projects</h1>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {projects.map((project) => (
          <ProjectCard key={project.id} project={project} />
        ))}
      </div>
    </div>
  )
}

function ProjectCard({ project }: { project: Project }) {
  const envsCount = useProjectCount(project.slug, 'environments', 'environments')
  const flagsCount = useProjectCount(project.slug, 'flags', 'flags')
  const membersCount = useProjectCount(project.slug, 'members', 'members')

  return (
    <Link
      to="/projects/$slug"
      params={{ slug: project.slug }}
      className="block border border-gray-200 rounded-lg bg-white p-4 hover:border-blue-300 hover:shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
    >
      <h2 className="text-sm font-semibold text-gray-900">{project.name}</h2>
      <p className="mt-0.5 font-mono text-xs text-gray-500">{project.slug}</p>
      <div className="mt-3 flex items-center gap-4 text-xs text-gray-600">
        <CountBadge label="Environments" count={envsCount.data} isLoading={envsCount.isLoading} isError={envsCount.isError} />
        <CountBadge label="Flags" count={flagsCount.data} isLoading={flagsCount.isLoading} isError={flagsCount.isError} />
        <CountBadge label="Members" count={membersCount.data} isLoading={membersCount.isLoading} isError={membersCount.isError} />
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
    <span className="flex items-center gap-1">
      {isLoading ? (
        <span className="inline-block h-3.5 w-4 bg-gray-100 rounded animate-pulse" />
      ) : isError ? (
        <span className="text-red-400">-</span>
      ) : (
        <span className="font-medium text-gray-900">{count ?? 0}</span>
      )}
      <span>{label}</span>
    </span>
  )
}

function HomePageSkeleton() {
  return (
    <div className="p-8">
      <div className="h-6 w-24 bg-gray-100 rounded animate-pulse mb-6" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="border border-gray-200 rounded-lg bg-white p-4"
          >
            <div className="h-4 w-32 bg-gray-100 rounded animate-pulse" />
            <div className="mt-1.5 h-3 w-20 bg-gray-100 rounded animate-pulse" />
            <div className="mt-3 flex items-center gap-4">
              <div className="h-3 w-20 bg-gray-100 rounded animate-pulse" />
              <div className="h-3 w-12 bg-gray-100 rounded animate-pulse" />
              <div className="h-3 w-16 bg-gray-100 rounded animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
