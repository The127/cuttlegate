import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { projectRoute } from './$slug'
import { fetchJSON, patchJSON, deleteRequest } from '../../api'
import {
  Button,
  Input,
  DataTable,
  CopyableCode,
  StatusBadge,
} from '../../components/ui'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogCloseButton,
} from '../../components/ui/Dialog'
import type { ColumnDef, SortState } from '../../components/ui'
import { formatRelativeDate } from '../../utils/date'
import { DeleteConfirmModal } from '../../components/DeleteConfirmModal'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { PageHeading } from '../../components/PageHeading'
import { CreateFlagModal } from './flag-create-modal'

interface Environment {
  id: string
  name: string
  slug: string
}

interface Variant {
  key: string
  name: string
}

interface FlagItem {
  id: string
  key: string
  name: string
  type: string
  variants: Variant[]
  default_variant_key: string
  enabled: boolean
}

interface FlagStats {
  last_evaluated_at: string | null
  evaluation_count: number
}

function useFlagStats(slug: string, envSlug: string, flagKey: string) {
  return useQuery<FlagStats>({
    queryKey: ['flag-stats', slug, envSlug, flagKey],
    queryFn: () =>
      fetchJSON<FlagStats>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags/${flagKey}/stats`,
      ),
    refetchInterval: 30_000,
  })
}

export const flagListRoute = createRoute({
  getParentRoute: () => projectEnvRoute,
  path: '/flags',
  component: FlagListPage,
})

function FlagListPage() {
  const { t } = useTranslation('flags')
  const { slug, envSlug } = flagListRoute.useParams()
  const project = projectRoute.useLoaderData()
  useDocumentTitle(t('page_title'), envSlug, project.name)
  const queryClient = useQueryClient()
  const queryKey = ['flags', slug, envSlug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ flags: FlagItem[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags`,
      ).then((d) => d.flags),
  })

  const { data: environments } = useQuery({
    queryKey: ['environments', slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${slug}/environments`,
      ).then((d) => d.environments),
    staleTime: 5 * 60_000,
  })
  const envName = environments?.find((e) => e.slug === envSlug)?.name ?? envSlug

  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedTerm, setDebouncedTerm] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedTerm(searchTerm), 200)
    return () => clearTimeout(timer)
  }, [searchTerm])

  const [toggleErrorKey, setToggleErrorKey] = useState<string | null>(null)

  const toggleMutation = useMutation({
    mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) =>
      patchJSON(`/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}`, { enabled }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
    onError: (_err, variables) => {
      setToggleErrorKey(variables.key)
      setTimeout(() => setToggleErrorKey(null), 3000)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (key: string) => deleteRequest(`/api/v1/projects/${slug}/flags/${key}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  const [pendingDelete, setPendingDelete] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  const columns = useMemo<ColumnDef<FlagItem>[]>(
    () => [
      {
        key: 'key',
        header: t('table.col_key'),
        sortable: true,
        sortValue: (f) => f.key,
        cell: (f) => (
          <div className="flex items-center gap-2">
            <Link
              to="/projects/$slug/environments/$envSlug/flags/$key"
              params={{ slug, envSlug, key: f.key }}
              className="font-mono text-sm text-[var(--color-accent-start)] hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] rounded"
            >
              {f.key}
            </Link>
            <CopyableCode
              value={f.key}
              aria-label={t('list.copy_key_aria', { key: f.key })}
              className="text-[var(--color-accent-start)]"
            />
            <span className="font-mono bg-[var(--color-surface-elevated)] border border-[var(--color-border)] text-[var(--color-text-muted)] text-xs rounded px-1.5 py-0.5">
              {t('type_badge.' + f.type, { defaultValue: f.type })}
            </span>
          </div>
        ),
      },
      {
        key: 'name',
        header: t('table.col_name'),
        sortable: true,
        sortValue: (f) => f.name,
        cell: (f) => (
          <Link
            to="/projects/$slug/environments/$envSlug/flags/$key"
            params={{ slug, envSlug, key: f.key }}
            className="text-sm text-[var(--color-text-primary)] truncate hover:underline hover:text-[var(--color-accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] rounded"
          >
            {f.name}
          </Link>
        ),
      },
      {
        key: 'default_variant',
        header: t('table.col_default_variant'),
        cell: (f) => (
          <span className="font-mono text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5">
            {f.default_variant_key}
          </span>
        ),
      },
      {
        key: 'last_evaluated',
        header: t('table.col_last_evaluated'),
        cell: (f) => <LastEvaluatedCell slug={slug} envSlug={envSlug} flagKey={f.key} />,
      },
      {
        key: 'status',
        header: t('table.col_status'),
        cell: (f) => {
          const isToggling = toggleMutation.isPending && toggleMutation.variables?.key === f.key
          const isError = toggleErrorKey === f.key
          const status = isError ? 'warning' : f.enabled ? 'enabled' : 'disabled'
          const label = isError
            ? t('toggle.failed')
            : f.enabled
              ? t('toggle.enabled')
              : t('toggle.disabled')
          return (
            <button
              onClick={() => toggleMutation.mutate({ key: f.key, enabled: !f.enabled })}
              disabled={isToggling}
              aria-pressed={f.enabled}
              aria-label={f.enabled ? t('toggle.disable') : t('toggle.enable')}
              className="focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded-full disabled:opacity-60"
            >
              <StatusBadge status={status} label={label} />
            </button>
          )
        },
      },
      {
        key: 'actions',
        header: '',
        cell: (f) => (
          <button
            onClick={() => setPendingDelete(f.key)}
            aria-label={t('list.delete_flag_aria', { key: f.key })}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-status-error)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded p-0.5"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-4 h-4"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        ),
      },
    ],
    [t, slug, envSlug, toggleMutation.isPending, toggleMutation.variables, toggleMutation.mutate, toggleErrorKey],
  )

  const flags = data ?? []
  const isSearching = debouncedTerm.length > 0
  const filteredFlags = useMemo(() => {
    if (!isSearching) return flags
    const term = debouncedTerm.toLowerCase()
    return flags.filter(
      (f) => f.key.toLowerCase().includes(term) || f.name.toLowerCase().includes(term),
    )
  }, [flags, debouncedTerm, isSearching])

  if (isLoading) return <FlagListSkeleton />
  if (isError) return <FlagListError onRetry={() => void refetch()} />

  return (
    <div className="p-6">
      <PageHeading
        ancestors={[{ label: t('nav.environments', { ns: 'common' }), to: `/projects/${slug}/environments` }]}
        current={envName}
      />
      <div className="flex items-center justify-between mb-6">
        <p className="text-sm text-[var(--color-text-muted)]">
          {isSearching
            ? t('list.search_count', { filtered: filteredFlags.length, total: flags.length })
            : t('list.flag_count', { count: flags.length })}
        </p>
        <Button onClick={() => setShowCreate(true)}>{t('list.new_flag')}</Button>
      </div>

      {flags.length > 0 && (
        <div className="relative mb-4">
          <Input
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder={t('list.search_placeholder')}
            aria-label={t('list.search_aria')}
            className="py-1.5 px-3 pr-8"
          />
          {searchTerm.length > 0 && (
            <button
              onClick={() => setSearchTerm('')}
              aria-label={t('list.search_clear_aria')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded p-0.5"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          )}
        </div>
      )}

      <DataTable
        columns={columns}
        data={filteredFlags}
        aria-label={envName}
        emptyState={
          isSearching
            ? <SearchEmptyState onClear={() => setSearchTerm('')} />
            : <EmptyState onCreateClick={() => setShowCreate(true)} />
        }
      />

      {showCreate && (
        <CreateFlagModal
          slug={slug}
          onCreated={() => {
            setShowCreate(false)
            void queryClient.invalidateQueries({ queryKey })
          }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      {pendingDelete && (
        <DeleteConfirmModal
          flagKey={pendingDelete}
          isDeleting={deleteMutation.isPending}
          deleteFailed={deleteMutation.isError}
          onConfirm={() => {
            deleteMutation.mutate(pendingDelete, {
              onSuccess: () => setPendingDelete(null),
            })
          }}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </div>
  )
}

function LastEvaluatedCell({ slug, envSlug, flagKey }: { slug: string; envSlug: string; flagKey: string }) {
  const { t } = useTranslation('flags')
  const { data } = useFlagStats(slug, envSlug, flagKey)

  if (!data) {
    return <span className="text-xs text-[var(--color-text-muted)]">—</span>
  }

  if (!data.last_evaluated_at || data.evaluation_count === 0) {
    return <span className="text-xs text-[var(--color-text-muted)]">{t('stats.never_evaluated')}</span>
  }

  return (
    <span className="text-xs text-[var(--color-text-secondary)]">{formatRelativeDate(data.last_evaluated_at)}</span>
  )
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="text-center py-16 px-6">
      <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
        {t('list.empty_title')}
      </h2>
      <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
        {t('list.empty_state')}
      </p>
      <Button onClick={onCreateClick} size="lg" className="mt-4">
        {t('list.new_flag')}
      </Button>
    </div>
  )
}

function SearchEmptyState({ onClear }: { onClear: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('list.search_no_results')}
      </p>
      <Button onClick={onClear} variant="secondary" size="lg" className="mt-4">
        {t('list.search_clear_aria')}
      </Button>
    </div>
  )
}

function FlagListSkeleton() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        <div className="h-8 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-40 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-5 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-3">
              <div className="h-6 w-20 bg-[var(--color-surface-elevated)] rounded-full animate-pulse" />
              <div className="h-4 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

function FlagListError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="p-6">
      <span className="text-sm text-[var(--color-status-error)]">{t('list.error')} </span>
      <button
        onClick={onRetry}
        className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
