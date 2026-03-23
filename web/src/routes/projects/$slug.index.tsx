import { createRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON } from '../../api'
import { formatRelativeDate } from '../../utils/date'

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

interface ProjectFlag {
  id: string
  key: string
  name: string
  created_at: string
}

export const projectIndexRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/',
  component: ProjectDashboard,
})

function ProjectDashboard() {
  const { t } = useTranslation('projects')
  const project = projectRoute.useLoaderData()

  const envsQuery = useQuery({
    queryKey: ['environments', project.slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${project.slug}/environments`,
      ).then((d) => d.environments),
  })

  const flagsQuery = useQuery({
    queryKey: ['flags', project.slug],
    queryFn: () =>
      fetchJSON<{ flags: ProjectFlag[] }>(
        `/api/v1/projects/${project.slug}/flags`,
      ).then((d) => d.flags),
  })

  return (
    <div className="p-6 max-w-5xl">
      <ProjectHeader name={project.name} slug={project.slug} />

      <section className="mt-6">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.environments_section')}
        </h2>
        {envsQuery.isLoading ? (
          <EnvironmentCardsSkeleton />
        ) : envsQuery.isError ? (
          <SectionError label="environments" onRetry={() => void envsQuery.refetch()} />
        ) : envsQuery.data!.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t('dashboard.no_environments')}
          </p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {envsQuery.data!.map((env) => (
              <EnvironmentCard key={env.id} env={env} projectSlug={project.slug} />
            ))}
          </div>
        )}
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.quick_links_section')}
        </h2>
        <div className="flex gap-3 flex-wrap">
          <QuickLink label={t('dashboard.compare')} to={`/projects/${project.slug}/compare`} />
          <QuickLink label={t('dashboard.segments')} to={`/projects/${project.slug}/segments`} />
          <QuickLink label={t('dashboard.settings')} to={`/projects/${project.slug}/settings`} />
          <QuickLink label={t('dashboard.members')} to={`/projects/${project.slug}/members`} />
          <QuickLink label={t('dashboard.api_keys')} to={`/projects/${project.slug}/api-keys`} />
          <QuickLink label={t('dashboard.audit_log')} to={`/projects/${project.slug}/audit`} />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.recent_flags_section')}
        </h2>
        {flagsQuery.isLoading ? (
          <RecentFlagsSkeleton />
        ) : flagsQuery.isError ? (
          <SectionError label="flags" onRetry={() => void flagsQuery.refetch()} />
        ) : flagsQuery.data!.length === 0 ? (
          <p className="text-sm text-gray-500">{t('dashboard.no_flags')}</p>
        ) : (
          <RecentFlagsList flags={flagsQuery.data!} />
        )}
      </section>
    </div>
  )
}

function ProjectHeader({ name, slug }: { name: string; slug: string }) {
  const { t } = useTranslation('projects')
  const [copied, setCopied] = useState(false)

  function copySlug() {
    void navigator.clipboard
      .writeText(slug)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {})
  }

  return (
    <>
      <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{name}</h1>
      <div className="mt-1 flex items-center gap-2">
        <button
          onClick={copySlug}
          className="relative font-mono text-sm text-gray-500 dark:text-gray-400 hover:text-[var(--color-accent)] bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded px-2 py-0.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
          aria-label={t('dashboard.copy_slug_aria', { slug })}
        >
          {slug}
          {copied && (
            <span className="absolute -top-7 left-1/2 -translate-x-1/2 text-xs bg-gray-800 dark:bg-gray-600 text-white rounded px-2 py-0.5 whitespace-nowrap pointer-events-none">
              {t('settings.copied')}
            </span>
          )}
        </button>
      </div>
    </>
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

  const total = flagsQuery.data?.length ?? 0
  const enabled = flagsQuery.data?.filter((f) => f.enabled).length ?? 0

  return (
    <Link
      to="/projects/$slug/environments/$envSlug/flags"
      params={{ slug: projectSlug, envSlug: env.slug }}
      className="block border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 p-4 hover:border-[var(--color-accent)] hover:shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      <h3 className="font-mono text-sm font-medium text-gray-900 dark:text-gray-100">{env.name}</h3>
      {flagsQuery.isLoading ? (
        <div className="mt-2 h-4 w-20 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
      ) : flagsQuery.isError ? (
        <p className="mt-2 text-xs text-red-600 dark:text-red-400">{t('dashboard.failed_to_load_env')}</p>
      ) : (
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          <span className="font-medium text-gray-800 dark:text-gray-200">{enabled}</span>
          <span className="text-gray-400 dark:text-gray-500"> / </span>
          <span>{total}</span>
          <span className="text-gray-400 dark:text-gray-500"> {t('toggle.enabled', { ns: 'flags' }).toLowerCase()}</span>
        </p>
      )}
    </Link>
  )
}

function QuickLink({ label, to }: { label: string; to: string }) {
  return (
    <Link
      to={to}
      className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-700 rounded hover:bg-gray-50 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600 transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      {label}
    </Link>
  )
}

function RecentFlagsList({ flags }: { flags: ProjectFlag[] }) {
  const recent = [...flags]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 5)

  return (
    <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
      {recent.map((flag) => (
        <li key={flag.id} className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3 min-w-0">
            <span className="font-mono text-sm text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-2 py-0.5">
              {flag.key}
            </span>
            <span className="text-sm text-gray-600 dark:text-gray-400 truncate">{flag.name}</span>
          </div>
          <time
            dateTime={flag.created_at}
            className="text-xs text-gray-400 dark:text-gray-500 shrink-0 ml-4"
            title={new Date(flag.created_at).toLocaleString()}
          >
            {formatRelativeDate(flag.created_at)}
          </time>
        </li>
      ))}
    </ul>
  )
}


function EnvironmentCardsSkeleton() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      {[1, 2, 3].map((i) => (
        <div key={i} className="border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 p-4">
          <div className="h-4 w-20 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
          <div className="mt-2 h-4 w-24 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        </div>
      ))}
    </div>
  )
}

function RecentFlagsSkeleton() {
  return (
    <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
      {[1, 2, 3].map((i) => (
        <li key={i} className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            <div className="h-6 w-28 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            <div className="h-4 w-40 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
          </div>
          <div className="h-3 w-12 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        </li>
      ))}
    </ul>
  )
}

function SectionError({ label, onRetry }: { label: string; onRetry: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div>
      <span className="text-sm text-red-600 dark:text-red-400">{t('dashboard.failed_to_load', { resource: label })} </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
