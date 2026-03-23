import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { fetchJSON, patchJSON, postJSON, deleteRequest, APIError } from '../../api'
import {
  Button,
  Input,
  Label,
  Select,
  SelectItem,
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
import type { ColumnDef } from '../../components/ui'
import { formatRelativeDate } from '../../utils/date'

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
  const queryClient = useQueryClient()
  const queryKey = ['flags', slug, envSlug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ flags: FlagItem[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags`,
      ).then((d) => d.flags),
  })

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
          <CopyableCode
            value={f.key}
            aria-label={t('list.copy_key_aria', { key: f.key })}
          />
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
            className="text-sm text-gray-700 dark:text-gray-200 truncate hover:text-[var(--color-accent)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
          >
            {f.name}
          </Link>
        ),
      },
      {
        key: 'default_variant',
        header: t('table.col_default_variant'),
        cell: (f) => (
          <span className="font-mono text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-1.5 py-0.5">
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
            className="text-gray-400 dark:text-gray-500 hover:text-red-600 dark:hover:text-red-400 transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 rounded p-0.5"
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

  if (isLoading) return <FlagListSkeleton />
  if (isError) return <FlagListError onRetry={() => void refetch()} />

  const flags = data ?? []

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('list.title')}</h1>
        <Button onClick={() => setShowCreate(true)}>{t('list.new_flag')}</Button>
      </div>

      <div className="border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-900 overflow-hidden">
        <DataTable
          columns={columns}
          data={flags}
          aria-label={t('list.title')}
          emptyState={<EmptyState onCreateClick={() => setShowCreate(true)} />}
        />
      </div>

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
    return <span className="text-xs text-gray-400 dark:text-gray-500">—</span>
  }

  if (!data.last_evaluated_at || data.evaluation_count === 0) {
    return <span className="text-xs text-gray-400 dark:text-gray-500">{t('stats.never_evaluated')}</span>
  }

  return (
    <span className="text-xs text-gray-500 dark:text-gray-400">{formatRelativeDate(data.last_evaluated_at)}</span>
  )
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('flags')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t('list.empty_state')}
      </p>
      <Button onClick={onCreateClick} size="lg" className="mt-4">
        {t('list.new_flag')}
      </Button>
    </div>
  )
}

// --- SDK prompt helpers ---

const SDK_TABS = ['go', 'js', 'python'] as const
type SdkTab = (typeof SDK_TABS)[number]

function safeGetTab(): SdkTab {
  try {
    const stored = localStorage.getItem('cg:sdk_tab')
    if (stored === 'go' || stored === 'js' || stored === 'python') return stored
  } catch {
    // localStorage unavailable — fall back to default
  }
  return 'go'
}

function safeSetTab(tab: SdkTab): void {
  try {
    localStorage.setItem('cg:sdk_tab', tab)
  } catch {
    // QuotaExceededError or security policy — UI state already updated, ignore
  }
}

function safeGetDismissed(): boolean {
  try {
    return localStorage.getItem('cg:sdk_prompt_dismissed') === 'true'
  } catch {
    return false
  }
}

function safeSetDismissed(): void {
  try {
    localStorage.setItem('cg:sdk_prompt_dismissed', 'true')
  } catch {
    // QuotaExceededError or security policy — UI state already updated, ignore
  }
}

// SDK snippet methods verified against actual source (2026-03-24):
//   Go:     CachedClient.Bool(ctx, key, ec)         — sdk/go/cached_client.go:92
//   JS:     evaluateFlag(key, context)               — sdk/js/src/client.ts:146
//   Python: CuttlegateClient.bool(key, ctx)          — sdk/python/cuttlegate/client.py:83
function buildSnippet(tab: SdkTab, flagKey: string): string {
  if (tab === 'go') {
    return `result, err := client.Bool(ctx, "${flagKey}", evalCtx)\nif err != nil {\n    // handle error\n}`
  }
  if (tab === 'js') {
    return `const result = await client.evaluateFlag('${flagKey}', context);\n`
  }
  // python
  return `result = client.bool("${flagKey}", ctx)\n`
}

function SdkPrompt({ flagKey }: { flagKey: string }) {
  const { t } = useTranslation('flags')
  const [activeTab, setActiveTab] = useState<SdkTab>(safeGetTab)
  const [dismissed, setDismissed] = useState<boolean>(safeGetDismissed)

  if (dismissed) return null

  function handleTabClick(tab: SdkTab) {
    setActiveTab(tab)
    safeSetTab(tab)
  }

  function handleDismiss() {
    setDismissed(true)
    safeSetDismissed()
  }

  return (
    <section
      role="region"
      aria-label={t('create.sdk_prompt.region_aria')}
      className="mt-5 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden"
    >
      <div className="px-4 pt-4 pb-3 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <p className="text-sm font-medium text-gray-800 dark:text-gray-100">
          {t('create.sdk_prompt.heading')}
        </p>
        <div className="mt-2 flex gap-1" role="tablist">
          {SDK_TABS.map((tab) => (
            <button
              key={tab}
              role="tab"
              aria-selected={activeTab === tab}
              onClick={() => handleTabClick(tab)}
              className={[
                'px-3 py-1 text-xs font-medium rounded border transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]',
                activeTab === tab
                  ? 'border-[var(--color-accent)] text-[var(--color-accent)] bg-white dark:bg-gray-900'
                  : 'border-gray-200 dark:border-gray-600 text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-900 hover:border-gray-400 dark:hover:border-gray-400',
              ].join(' ')}
            >
              {t(`create.sdk_prompt.tab_${tab}`)}
            </button>
          ))}
        </div>
      </div>
      <pre className="p-4 text-xs font-mono overflow-x-auto bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-100 leading-relaxed">
        {buildSnippet(activeTab, flagKey)}
      </pre>
      <div className="flex justify-end px-4 pb-3 bg-white dark:bg-gray-900">
        <button
          onClick={handleDismiss}
          aria-label={t('create.sdk_prompt.dismiss_aria')}
          className="text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
        >
          {t('create.sdk_prompt.dismiss')}
        </button>
      </div>
    </section>
  )
}

// --- Flag key validation ---

const FLAG_KEY_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_KEY_LENGTH = 128

function validateKeyLocally(key: string, t: (k: string, opts?: Record<string, unknown>) => string): string | null {
  if (key.length === 0) return null
  if (key.length > MAX_KEY_LENGTH) return t('create.key_too_long', { max: MAX_KEY_LENGTH })
  if (!FLAG_KEY_RE.test(key)) return t('create.key_invalid')
  return null
}

function CreateFlagModal({
  slug,
  onCreated,
  onCancel,
}: {
  slug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('flags')
  const [key, setKey] = useState('')
  const [name, setName] = useState('')
  const [type, setType] = useState('bool')
  const [keyError, setKeyError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)
  const [keyTouched, setKeyTouched] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () => {
      const variants =
        type === 'bool'
          ? [{ key: 'true', name: 'On' }, { key: 'false', name: 'Off' }]
          : [{ key: 'default', name: 'Default' }]
      const default_variant_key = type === 'bool' ? 'false' : 'default'
      return postJSON(`/api/v1/projects/${slug}/flags`, {
        key,
        name,
        type,
        variants,
        default_variant_key,
      })
    },
    onSuccess: () => setCreatedKey(key),
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setKeyError(t('create.key_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setKeyError(err.message)
          return
        }
      }
      setServerError(
        err instanceof APIError ? err.message : t('create.server_error'),
      )
    },
  })

  function handleKeyChange(value: string) {
    setKey(value)
    setKeyError(null)
    setServerError(null)
    if (keyTouched) {
      setKeyError(validateKeyLocally(value, t))
    }
  }

  function handleKeyBlur() {
    setKeyTouched(true)
    setKeyError(validateKeyLocally(key, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const localError = validateKeyLocally(key, t)
    if (localError) {
      setKeyError(localError)
      return
    }
    if (key.length === 0) {
      setKeyError(t('create.key_required'))
      return
    }
    setServerError(null)
    createMutation.mutate()
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) { if (createdKey) onCreated(); else onCancel() } }}>
      <DialogContent>
        <DialogCloseButton />
        {createdKey ? (
          <>
            <DialogHeader>
              <DialogTitle>{t('create.success_title')}</DialogTitle>
            </DialogHeader>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              {t('create.success_body')}
            </p>
            <CopyableCode
              value={createdKey}
              aria-label={t('create.success_copy_aria', { key: createdKey })}
              className="w-full justify-between"
            />
            <SdkPrompt flagKey={createdKey} />
            <DialogFooter>
              <Button variant="primary" onClick={onCreated}>
                {t('create.success_done')}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{t('create.title')}</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="flag-key" className="text-xs text-gray-500 mb-1">{t('create.key_label')}</Label>
                <Input
                  id="flag-key"
                  type="text"
                  autoFocus
                  value={key}
                  onChange={(e) => handleKeyChange(e.target.value)}
                  onBlur={handleKeyBlur}
                  placeholder={t('create.key_placeholder')}
                  aria-invalid={!!keyError}
                  aria-describedby={keyError ? 'flag-key-error' : undefined}
                  hasError={!!keyError}
                  className="font-mono py-1.5 px-2"
                />
                {keyError && (
                  <p id="flag-key-error" className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {keyError}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="flag-name" className="text-xs text-gray-500 mb-1">{t('create.name_label')}</Label>
                <Input
                  id="flag-name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t('create.name_placeholder')}
                  className="py-1.5 px-2"
                />
              </div>

              <div>
                <Label htmlFor="flag-type" className="text-xs text-gray-500 mb-1">{t('create.type_label')}</Label>
                <Select
                  value={type}
                  onValueChange={setType}
                  aria-label={t('create.type_aria')}
                  className="w-full"
                >
                  <SelectItem value="bool">{t('create.type_bool')}</SelectItem>
                  <SelectItem value="string">{t('create.type_string')}</SelectItem>
                  <SelectItem value="number">{t('create.type_number')}</SelectItem>
                  <SelectItem value="json">{t('create.type_json')}</SelectItem>
                </Select>
              </div>

              {serverError && (
                <p className="text-xs text-red-600 dark:text-red-400">{serverError}</p>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={onCancel}
                  disabled={createMutation.isPending}
                >
                  {t('actions.cancel', { ns: 'common' })}
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  loading={createMutation.isPending}
                  disabled={!!keyError}
                >
                  {createMutation.isPending ? t('states.creating', { ns: 'common' }) : t('actions.create', { ns: 'common' })}
                </Button>
              </div>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

function DeleteConfirmModal({
  flagKey,
  isDeleting,
  deleteFailed,
  onConfirm,
  onCancel,
}: {
  flagKey: string
  isDeleting: boolean
  deleteFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('flags')

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onCancel() }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('delete.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          <Trans
            i18nKey="delete.body"
            ns="flags"
            values={{ key: flagKey }}
            components={{ mono: <span className="font-mono text-gray-800 dark:text-gray-200" /> }}
          />
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-red-600 dark:text-red-400">{t('delete.failed')}</p>
        )}
        <DialogFooter>
          <Button autoFocus variant="secondary" onClick={onCancel} disabled={isDeleting}>
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button variant="destructive" onClick={onConfirm} loading={isDeleting}>
            {isDeleting ? t('states.deleting', { ns: 'common' }) : t('actions.delete', { ns: 'common' })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function FlagListSkeleton() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        <div className="h-8 w-24 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
              <div className="h-4 w-40 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
              <div className="h-5 w-16 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-3">
              <div className="h-6 w-20 bg-gray-100 dark:bg-gray-700 rounded-full animate-pulse" />
              <div className="h-4 w-4 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
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
      <span className="text-sm text-red-600 dark:text-red-400">{t('list.error')} </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
