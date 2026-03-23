import { createRoute } from '@tanstack/react-router'
import { useQuery, useQueries, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON } from '../../api'
import { PromoteDialog } from '../../components/PromoteDialog'

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
  default_variant_key: string
}

interface ProjectFlag {
  id: string
  key: string
  name: string
}

interface CellState {
  enabled: boolean
  default_variant_key: string
}

const PAGE_SIZE = 50

export const compareRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/compare',
  component: CompareEnvironmentsPage,
})

function CompareEnvironmentsPage() {
  const { t } = useTranslation('projects')
  const { t: tFlags } = useTranslation('flags')
  const project = projectRoute.useLoaderData()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(0)
  // slug of the env being promoted from; null when dialog is closed
  const [promoteSourceEnv, setPromoteSourceEnv] = useState<string | null>(null)

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

  const envs = envsQuery.data ?? []

  const envFlagQueries = useQueries({
    queries: envs.map((env) => ({
      queryKey: ['flags', project.slug, env.slug],
      queryFn: () =>
        fetchJSON<{ flags: EnvFlag[] }>(
          `/api/v1/projects/${project.slug}/environments/${env.slug}/flags`,
        ).then((d) => d.flags),
    })),
  })

  const isLoading =
    envsQuery.isLoading || flagsQuery.isLoading || envFlagQueries.some((q) => q.isLoading)
  const isError =
    envsQuery.isError || flagsQuery.isError || envFlagQueries.some((q) => q.isError)

  // Build matrix: flagKey → envSlug → CellState
  const matrix = useMemo<Map<string, Map<string, CellState>>>(() => {
    const m = new Map<string, Map<string, CellState>>()
    envFlagQueries.forEach((q, i) => {
      if (!q.data) return
      const env = envs[i]
      q.data.forEach((flag) => {
        if (!m.has(flag.key)) m.set(flag.key, new Map())
        m.get(flag.key)!.set(env.slug, {
          enabled: flag.enabled,
          default_variant_key: flag.default_variant_key,
        })
      })
    })
    return m
  }, [envFlagQueries, envs])

  const sortedFlags = useMemo(
    () => [...(flagsQuery.data ?? [])].sort((a, b) => a.key.localeCompare(b.key)),
    [flagsQuery.data],
  )

  const totalPages = Math.ceil(sortedFlags.length / PAGE_SIZE)
  const pagedFlags = sortedFlags.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)

  function retry() {
    void envsQuery.refetch()
    void flagsQuery.refetch()
    envFlagQueries.forEach((q) => void q.refetch())
  }

  return (
    <div className="p-6">
      <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('compare.title')}</h1>

      {isLoading ? (
        <MatrixSkeleton />
      ) : isError ? (
        <MatrixError onRetry={retry} />
      ) : sortedFlags.length === 0 ? (
        <EmptyState />
      ) : (
        <>
          <div className="overflow-x-auto border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
            <table className="min-w-full text-sm" aria-label={t('compare.matrix_aria')}>
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
                  <th
                    scope="col"
                    className="sticky left-0 z-10 bg-gray-50 dark:bg-gray-900 px-4 py-2.5 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide w-64"
                  >
                    {t('compare.flag_header')}
                  </th>
                  {envs.map((env) => (
                    <th
                      key={env.id}
                      scope="col"
                      className="px-4 py-2.5 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide whitespace-nowrap"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-mono">{env.slug}</span>
                        <button
                          onClick={() => setPromoteSourceEnv(env.slug)}
                          className="text-[var(--color-accent)] hover:opacity-75 text-xs font-normal normal-case tracking-normal focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
                          aria-label={tFlags('promote.bulk_button') + ' ' + env.slug}
                        >
                          {tFlags('promote.bulk_button')}
                        </button>
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {pagedFlags.map((flag) => (
                  <tr key={flag.id} className="group hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                    <td className="sticky left-0 z-10 bg-white dark:bg-gray-800 group-hover:bg-gray-50 dark:group-hover:bg-gray-700/50 px-4 py-2.5 w-64">
                      <span className="font-mono text-xs text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-1.5 py-0.5">
                        {flag.key}
                      </span>
                    </td>
                    {envs.map((env) => {
                      const cell = matrix.get(flag.key)?.get(env.slug)
                      return (
                        <td key={env.id} className="px-4 py-2.5">
                          {cell ? (
                            <MatrixCell cell={cell} />
                          ) : (
                            <span className="text-xs text-gray-300 dark:text-gray-600">—</span>
                          )}
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
          )}
        </>
      )}

      {promoteSourceEnv && (
        <PromoteDialog
          mode="bulk"
          projectSlug={project.slug}
          sourceEnvSlug={promoteSourceEnv}
          environments={envs}
          onClose={() => setPromoteSourceEnv(null)}
          onSuccess={() => {
            // Invalidate all env-flag queries so the matrix refreshes
            void queryClient.invalidateQueries({ queryKey: ['flags', project.slug] })
          }}
        />
      )}
    </div>
  )
}

function MatrixCell({ cell }: { cell: CellState }) {
  const { t } = useTranslation('projects')
  return (
    <div className="flex flex-col gap-1">
      <span
        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border w-fit ${
          cell.enabled
            ? 'bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 border-gray-200 dark:border-gray-600'
        }`}
      >
        <span
          className={`w-1.5 h-1.5 rounded-full ${cell.enabled ? 'bg-green-500 dark:bg-green-400' : 'bg-gray-400 dark:bg-gray-500'}`}
          aria-hidden="true"
        />
        {cell.enabled ? t('compare.enabled') : t('compare.disabled')}
      </span>
      <span className="font-mono text-xs text-gray-500 dark:text-gray-400">{cell.default_variant_key}</span>
    </div>
  )
}

function Pagination({
  page,
  totalPages,
  onPageChange,
}: {
  page: number
  totalPages: number
  onPageChange: (p: number) => void
}) {
  const { t } = useTranslation('projects')
  return (
    <div className="mt-4 flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
      <span>
        {t('compare.page_info', { current: page + 1, total: totalPages })}
      </span>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page === 0}
          className="px-3 py-1.5 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
        >
          {t('compare.previous')}
        </button>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages - 1}
          className="px-3 py-1.5 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
        >
          {t('compare.next')}
        </button>
      </div>
    </div>
  )
}

function EmptyState() {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-16 px-6 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t('compare.empty')}
      </p>
    </div>
  )
}

function MatrixSkeleton() {
  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 overflow-hidden">
      <div className="flex gap-4 px-4 py-2.5 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
        <div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
        ))}
      </div>
      {[1, 2, 3, 4, 5].map((i) => (
        <div
          key={i}
          className="flex gap-4 px-4 py-3 border-b border-gray-100 dark:border-gray-700 last:border-0"
        >
          <div className="h-5 w-36 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
          {[1, 2, 3].map((j) => (
            <div key={j} className="flex flex-col gap-1">
              <div className="h-5 w-16 bg-gray-100 dark:bg-gray-700 rounded-full animate-pulse" />
              <div className="h-3 w-10 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}

function MatrixError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div>
      <span className="text-sm text-red-600 dark:text-red-400">{t('compare.error')} </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
