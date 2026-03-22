import { createRoute } from '@tanstack/react-router'
import { useInfiniteQuery } from '@tanstack/react-query'
import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON } from '../../api'
import { formatAbsoluteDate, formatRelativeDate } from '../../utils/date'

interface AuditEntry {
  id: string
  occurred_at: string
  actor_email: string
  action: string
  flag_key: string
  environment_slug: string
  project_slug: string
}

interface AuditListResponse {
  entries: AuditEntry[]
  next_cursor: string | null
}

export const auditRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/audit',
  component: AuditLogPage,
})

function AuditLogPage() {
  const { t } = useTranslation('projects')
  const { slug } = auditRoute.useParams()
  const [flagKey, setFlagKey] = useState('')
  const debouncedFlagKey = useDebounced(flagKey, 300)

  const {
    data,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['audit', slug, debouncedFlagKey],
    queryFn: ({ pageParam }) => {
      const params = new URLSearchParams({ limit: '50' })
      if (debouncedFlagKey) params.set('flag_key', debouncedFlagKey)
      if (pageParam) params.set('cursor', pageParam as string)
      return fetchJSON<AuditListResponse>(
        `/api/v1/projects/${slug}/audit?${params.toString()}`,
      )
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
  })

  const allEntries = data?.pages.flatMap((p) => p.entries) ?? []

  if (isLoading) return <AuditLogSkeleton />

  if (isError) {
    return (
      <div className="p-6">
        <span className="text-sm text-red-600">{t('audit.error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-5xl">
      <h1 className="text-lg font-semibold text-gray-900 mb-4">{t('audit.title')}</h1>

      <div className="mb-4">
        <label htmlFor="audit-flag-filter" className="sr-only">
          {t('audit.filter_label')}
        </label>
        <input
          id="audit-flag-filter"
          type="text"
          value={flagKey}
          onChange={(e) => setFlagKey(e.target.value)}
          placeholder={t('audit.filter_placeholder')}
          className="w-full max-w-sm font-mono text-sm border border-gray-300 rounded px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {allEntries.length === 0 ? (
        <div className="text-center py-16 px-6 border border-gray-200 rounded-lg bg-white">
          <p className="text-sm text-gray-500">{t('audit.empty')}</p>
        </div>
      ) : (
        <div className="border border-gray-200 rounded-lg bg-white overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  {t('audit.column_timestamp')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  {t('audit.column_actor')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  {t('audit.column_action')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  {t('audit.column_flag')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  {t('audit.column_environment')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {allEntries.map((entry) => (
                <AuditEntryRow key={entry.id} entry={entry} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {hasNextPage && (
        <div className="mt-4 flex justify-center">
          <button
            onClick={() => void fetchNextPage()}
            disabled={isFetchingNextPage}
            className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {isFetchingNextPage ? t('audit.loading_more') : t('audit.load_more')}
          </button>
        </div>
      )}
    </div>
  )
}

function AuditEntryRow({ entry }: { entry: AuditEntry }) {
  const absolute = formatAbsoluteDate(entry.occurred_at)
  const relative = formatRelativeDate(entry.occurred_at)
  return (
    <tr className="hover:bg-gray-50">
      <td className="px-4 py-3 whitespace-nowrap">
        <time
          dateTime={entry.occurred_at}
          title={relative}
          className="text-xs text-gray-600 tabular-nums"
        >
          {absolute}
        </time>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className="text-sm text-gray-800">{entry.actor_email}</span>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className="font-mono text-xs text-gray-700 bg-gray-50 border border-gray-200 rounded px-2 py-0.5">
          {entry.action}
        </span>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {entry.flag_key ? (
          <span className="font-mono text-xs text-gray-700 bg-gray-50 border border-gray-200 rounded px-2 py-0.5">
            {entry.flag_key}
          </span>
        ) : (
          <span className="text-xs text-gray-400">—</span>
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {entry.environment_slug ? (
          <span className="font-mono text-xs text-gray-700 bg-gray-50 border border-gray-200 rounded px-2 py-0.5">
            {entry.environment_slug}
          </span>
        ) : (
          <span className="text-xs text-gray-400">—</span>
        )}
      </td>
    </tr>
  )
}

function AuditLogSkeleton() {
  return (
    <div className="p-6 max-w-5xl">
      <div className="h-6 w-24 bg-gray-100 rounded animate-pulse mb-4" />
      <div className="h-8 w-64 bg-gray-100 rounded animate-pulse mb-4" />
      <div className="border border-gray-200 rounded-lg bg-white overflow-hidden">
        <div className="bg-gray-50 border-b border-gray-200 px-4 py-2 flex gap-8">
          <div className="h-3 w-28 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-40 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-20 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-24 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-28 bg-gray-200 rounded animate-pulse" />
        </div>
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-8 px-4 py-3 border-b border-gray-100 last:border-0">
            <div className="h-4 w-28 bg-gray-100 rounded animate-pulse" />
            <div className="h-4 w-40 bg-gray-100 rounded animate-pulse" />
            <div className="h-4 w-20 bg-gray-100 rounded animate-pulse" />
            <div className="h-4 w-24 bg-gray-100 rounded animate-pulse" />
            <div className="h-4 w-20 bg-gray-100 rounded animate-pulse" />
          </div>
        ))}
      </div>
    </div>
  )
}

function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setDebounced(value), delay)
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current)
    }
  }, [value, delay])

  return debounced
}
