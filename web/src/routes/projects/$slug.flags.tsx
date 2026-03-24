import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, deleteRequest } from '../../api'
import {
  Button,
  Input,
  DataTable,
  CopyableCode,
  StatusBadge,
} from '../../components/ui'
import type { ColumnDef } from '../../components/ui'
import { formatRelativeDate } from '../../utils/date'
import { DeleteConfirmModal } from '../../components/DeleteConfirmModal'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { PageHeading } from '../../components/PageHeading'
import { CreateFlagModal } from './flag-create-modal'

// --- Types ---

interface Variant {
  key: string
  name: string
}

interface ProjectFlagItem {
  id: string
  key: string
  name: string
  type: string
  variants: Variant[]
  default_variant_key: string
  created_at: string
}

interface Environment {
  id: string
  name: string
  slug: string
}

interface EnvFlagItem {
  id: string
  key: string
  enabled: boolean
}

// --- Route ---

export const projectFlagListRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/flags',
  component: ProjectFlagListPage,
})

// --- Page ---

function ProjectFlagListPage() {
  const { t } = useTranslation('flags')
  const { slug } = projectFlagListRoute.useParams()
  const project = projectRoute.useLoaderData()
  useDocumentTitle(t('project_list.page_title'), project.name)
  const queryClient = useQueryClient()
  const queryKey = ['project-flags', slug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ flags: ProjectFlagItem[] }>(
        `/api/v1/projects/${slug}/flags`,
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

  const sortedEnvs = useMemo(
    () =>
      environments
        ? [...environments].sort((a, b) => a.slug.localeCompare(b.slug))
        : [],
    [environments],
  )

  const firstEnvSlug = sortedEnvs.length > 0 ? sortedEnvs[0].slug : null

  // Search / filter (reuse #365 pattern)
  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedTerm, setDebouncedTerm] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedTerm(searchTerm), 200)
    return () => clearTimeout(timer)
  }, [searchTerm])

  const deleteMutation = useMutation({
    mutationFn: (key: string) => deleteRequest(`/api/v1/projects/${slug}/flags/${key}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  const [pendingDelete, setPendingDelete] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)

  const columns = useMemo<ColumnDef<ProjectFlagItem>[]>(
    () => [
      {
        key: 'key',
        header: t('project_list.col_key'),
        sortable: true,
        sortValue: (f) => f.key,
        cell: (f) => (
          <div className="flex items-center gap-2">
            {firstEnvSlug ? (
              <Link
                to="/projects/$slug/environments/$envSlug/flags/$key"
                params={{ slug, envSlug: firstEnvSlug, key: f.key }}
                className="font-mono text-sm text-[var(--color-accent-start)] hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] rounded"
              >
                {f.key}
              </Link>
            ) : (
              <span className="font-mono text-sm text-[var(--color-text-primary)]">
                {f.key}
              </span>
            )}
            <CopyableCode
              value={f.key}
              aria-label={t('project_list.copy_key_aria', { key: f.key })}
              className="text-[var(--color-accent-start)]"
            />
          </div>
        ),
      },
      {
        key: 'name',
        header: t('project_list.col_name'),
        sortable: true,
        sortValue: (f) => f.name,
        cell: (f) => (
          <span className="text-sm text-[var(--color-text-primary)] truncate">
            {f.name}
          </span>
        ),
      },
      {
        key: 'type',
        header: t('project_list.col_type'),
        cell: (f) => (
          <span className="font-mono bg-[var(--color-surface-elevated)] border border-[var(--color-border)] text-[var(--color-text-muted)] text-xs rounded px-1.5 py-0.5">
            {t('type_badge.' + f.type, { defaultValue: f.type })}
          </span>
        ),
      },
      {
        key: 'variants',
        header: t('project_list.col_variants'),
        sortable: true,
        sortValue: (f) => f.variants.length,
        cell: (f) => (
          <span className="text-sm text-[var(--color-text-secondary)]">
            {f.variants.length}
          </span>
        ),
      },
      {
        key: 'created',
        header: t('project_list.col_created'),
        sortable: true,
        sortValue: (f) => f.created_at,
        cell: (f) => (
          <span className="text-xs text-[var(--color-text-secondary)]">
            {formatRelativeDate(f.created_at)}
          </span>
        ),
      },
      ...(sortedEnvs.length > 0
        ? [
            {
              key: 'environments',
              header: t('project_list.col_environments'),
              cell: (f: ProjectFlagItem) => (
                <EnvironmentStatusBadges
                  slug={slug}
                  flagKey={f.key}
                  environments={sortedEnvs}
                />
              ),
            },
          ]
        : []),
      {
        key: 'actions',
        header: '',
        cell: (f) => (
          <button
            onClick={() => setPendingDelete(f.key)}
            aria-label={t('project_list.delete_flag_aria', { key: f.key })}
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
    [t, slug, firstEnvSlug, sortedEnvs, setPendingDelete],
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

  if (isLoading) return <ProjectFlagListSkeleton />
  if (isError) return <ProjectFlagListError onRetry={() => void refetch()} />

  return (
    <div className="p-6">
      <PageHeading ancestors={[]} current={t('project_list.title')} />
      <div className="flex items-center justify-between mb-6">
        <p className="text-sm text-[var(--color-text-muted)]">
          {isSearching
            ? t('project_list.search_count', { filtered: filteredFlags.length, total: flags.length })
            : t('project_list.flag_count', { count: flags.length })}
        </p>
        <Button onClick={() => setShowCreate(true)}>{t('project_list.new_flag')}</Button>
      </div>

      {flags.length > 0 && (
        <div className="relative mb-4">
          <Input
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder={t('project_list.search_placeholder')}
            aria-label={t('project_list.search_aria')}
            className="py-1.5 px-3 pr-8"
          />
          {searchTerm.length > 0 && (
            <button
              onClick={() => setSearchTerm('')}
              aria-label={t('project_list.search_clear_aria')}
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
        aria-label={t('project_list.title')}
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

// --- Environment status badges (supplementary, loaded independently) ---

function EnvironmentStatusBadges({
  slug,
  flagKey,
  environments,
}: {
  slug: string
  flagKey: string
  environments: Environment[]
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {environments.map((env) => (
        <EnvBadge key={env.id} slug={slug} envSlug={env.slug} envName={env.name} flagKey={flagKey} />
      ))}
    </div>
  )
}

function EnvBadge({
  slug,
  envSlug,
  envName,
  flagKey,
}: {
  slug: string
  envSlug: string
  envName: string
  flagKey: string
}) {
  const { t } = useTranslation('flags')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['env-flags', slug, envSlug],
    queryFn: () =>
      fetchJSON<{ flags: EnvFlagItem[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags`,
      ).then((d) => d.flags),
    staleTime: 30_000,
  })

  if (isLoading) {
    return (
      <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] bg-[var(--color-surface-elevated)] text-[var(--color-text-muted)] border border-[var(--color-border)] animate-pulse">
        {envName}
      </span>
    )
  }

  if (isError || !data) return null

  const envFlag = data.find((f) => f.key === flagKey)
  if (!envFlag) return null

  const status = envFlag.enabled ? 'enabled' : 'disabled'
  const label = envFlag.enabled
    ? t('project_list.env_badge_enabled', { env: envName })
    : t('project_list.env_badge_disabled', { env: envName })

  return <StatusBadge status={status} label={label} className="text-[10px] px-1.5 py-0" />
}

// --- Empty states ---

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('project_list.empty_state')}
      </p>
      <Button onClick={onCreateClick} size="lg" className="mt-4">
        {t('project_list.new_flag')}
      </Button>
    </div>
  )
}

function SearchEmptyState({ onClear }: { onClear: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('project_list.search_no_results')}
      </p>
      <Button onClick={onClear} variant="secondary" size="lg" className="mt-4">
        {t('project_list.search_clear_aria')}
      </Button>
    </div>
  )
}

// --- Skeleton & Error ---

function ProjectFlagListSkeleton() {
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
              <div className="h-4 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

function ProjectFlagListError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="p-6">
      <span className="text-sm text-[var(--color-status-error)]">{t('project_list.error')} </span>
      <button
        onClick={onRetry}
        className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
