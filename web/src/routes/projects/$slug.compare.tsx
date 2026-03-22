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
      <h1 className="text-lg font-semibold text-gray-900 mb-6">{t('compare.title')}</h1>

      {isLoading ? (
        <MatrixSkeleton />
      ) : isError ? (
        <MatrixError onRetry={retry} />
      ) : sortedFlags.length === 0 ? (
        <EmptyState />
      ) : (
        <>
          <div className="overflow-x-auto border border-gray-200 rounded-lg bg-white">
            <table className="min-w-full text-sm" aria-label={t('compare.matrix_aria')}>
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th
                    scope="col"
                    className="sticky left-0 z-10 bg-gray-50 px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wide w-64"
                  >
                    {t('compare.flag_header')}
                  </th>
                  {envs.map((env) => (
                    <th
                      key={env.id}
                      scope="col"
                      className="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-mono">{env.slug}</span>
                        <button
                          onClick={() => setPromoteSourceEnv(env.slug)}
                          className="text-blue-500 hover:text-blue-700 text-xs font-normal normal-case tracking-normal focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
                          aria-label={tFlags('promote.bulk_button') + ' ' + env.slug}
                        >
                          {tFlags('promote.bulk_button')}
                        </button>
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {pagedFlags.map((flag) => (
                  <tr key={flag.id} className="group hover:bg-gray-50 transition-colors">
                    <td className="sticky left-0 z-10 bg-white group-hover:bg-gray-50 px-4 py-2.5 w-64">
                      <span className="font-mono text-xs text-gray-800 bg-gray-50 border border-gray-200 rounded px-1.5 py-0.5">
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
                            <span className="text-xs text-gray-300">—</span>
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
            ? 'bg-green-50 text-green-700 border-green-200'
            : 'bg-gray-100 text-gray-500 border-gray-200'
        }`}
      >
        <span
          className={`w-1.5 h-1.5 rounded-full ${cell.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
          aria-hidden="true"
        />
        {cell.enabled ? t('compare.enabled') : t('compare.disabled')}
      </span>
      <span className="font-mono text-xs text-gray-500">{cell.default_variant_key}</span>
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
    <div className="mt-4 flex items-center justify-between text-sm text-gray-600">
      <span>
        {t('compare.page_info', { current: page + 1, total: totalPages })}
      </span>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page === 0}
          className="px-3 py-1.5 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {t('compare.previous')}
        </button>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages - 1}
          className="px-3 py-1.5 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-blue-500"
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
    <div className="text-center py-16 px-6 border border-gray-200 rounded-lg bg-white">
      <p className="text-sm text-gray-500">
        {t('compare.empty')}
      </p>
    </div>
  )
}

function MatrixSkeleton() {
  return (
    <div className="border border-gray-200 rounded-lg bg-white overflow-hidden">
      <div className="flex gap-4 px-4 py-2.5 border-b border-gray-200 bg-gray-50">
        <div className="h-4 w-32 bg-gray-200 rounded animate-pulse" />
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-4 w-20 bg-gray-200 rounded animate-pulse" />
        ))}
      </div>
      {[1, 2, 3, 4, 5].map((i) => (
        <div
          key={i}
          className="flex gap-4 px-4 py-3 border-b border-gray-100 last:border-0"
        >
          <div className="h-5 w-36 bg-gray-100 rounded animate-pulse" />
          {[1, 2, 3].map((j) => (
            <div key={j} className="flex flex-col gap-1">
              <div className="h-5 w-16 bg-gray-100 rounded-full animate-pulse" />
              <div className="h-3 w-10 bg-gray-100 rounded animate-pulse" />
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
      <span className="text-sm text-red-600">{t('compare.error')} </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
