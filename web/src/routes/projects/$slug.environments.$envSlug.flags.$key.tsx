import { createRoute, useNavigate, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo, type ChangeEvent } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { fetchJSON, patchJSON, postJSON, deleteRequest, APIError } from '../../api'
import { useFlagSSE } from '../../hooks/useFlagSSE'
import { Button, Input, Select, SelectItem, CopyableCode } from '../../components/ui'
import { PromoteDialog } from '../../components/PromoteDialog'
import { FlagAnalyticsPanel } from '../../components/FlagAnalyticsPanel'
import { formatAbsoluteDate, formatRelativeDate } from '../../utils/date'

interface Variant {
  key: string
  name: string
}

interface FlagDetail {
  id: string
  key: string
  name: string
  type: string
  variants: Variant[]
  default_variant_key: string
  enabled: boolean
}

interface Environment {
  id: string
  slug: string
  name: string
}

interface FlagEnvState {
  enabled: boolean
}

const REASON_KEY_MAP: Record<string, string> = {
  disabled: 'eval.reason_disabled',
  default: 'eval.reason_default',
  rule_match: 'eval.reason_rule_match',
}

export const flagDetailRoute = createRoute({
  getParentRoute: () => projectEnvRoute,
  path: '/flags/$key',
  component: FlagDetailPage,
})

function FlagDetailPage() {
  const { t } = useTranslation('flags')
  const { slug, envSlug, key } = flagDetailRoute.useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const queryKey = ['flag', slug, envSlug, key]
  const listQueryKey = ['flags', slug, envSlug]

  useFlagSSE(slug, envSlug)

  const { data: flag, isLoading, error } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<FlagDetail>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}`,
      ),
    retry: (failureCount, err) => {
      if (err instanceof APIError && err.status === 404) return false
      return failureCount < 2
    },
  })

  const toggleMutation = useMutation({
    mutationFn: (enabled: boolean) =>
      patchJSON(`/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}`, { enabled }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteRequest(`/api/v1/projects/${slug}/flags/${key}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listQueryKey })
      void navigate({ to: '/projects/$slug/environments/$envSlug/flags', params: { slug, envSlug } })
    },
  })

  const [pendingDelete, setPendingDelete] = useState(false)
  const [pendingPromote, setPendingPromote] = useState(false)

  const { data: environments } = useQuery({
    queryKey: ['environments', slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(`/api/v1/projects/${slug}/environments`)
        .then((d) => d.environments),
  })

  if (isLoading) return <FlagDetailSkeleton />

  if (error) {
    const is404 = error instanceof APIError && error.status === 404
    return (
      <div className="p-6">
        <p className="text-sm text-[var(--color-status-error)]">
          {is404 ? t('list.not_found') : t('list.failed_to_load')}
        </p>
        <a
          href={`/projects/${slug}/environments/${envSlug}/flags`}
          className="mt-2 inline-block text-sm text-[var(--color-accent)] underline hover:no-underline"
        >
          {t('list.back_to_flags')}
        </a>
      </div>
    )
  }

  if (!flag) return null

  return (
    <div className="p-6 max-w-2xl">
      <FlagDetailCard
        flag={flag}
        slug={slug}
        envSlug={envSlug}
        isToggling={toggleMutation.isPending}
        onToggle={(enabled) => toggleMutation.mutate(enabled)}
        onDeleteIntent={() => setPendingDelete(true)}
        onPromoteIntent={() => setPendingPromote(true)}
        onSaved={() => void queryClient.invalidateQueries({ queryKey })}
      />

      <EnvironmentTogglePanel slug={slug} flagKey={key} />

      <div className="mt-4 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4 flex gap-4">
        <Link
          to="/projects/$slug/environments/$envSlug/flags/$key/rules"
          params={{ slug, envSlug, key }}
          className="text-sm text-[var(--color-accent)] hover:underline"
        >
          {t('detail.targeting_rules')}
        </Link>
        <Link
          to="/projects/$slug/environments/$envSlug/flags/$key/evaluations"
          params={{ slug, envSlug, key }}
          className="text-sm text-[var(--color-accent)] hover:underline"
        >
          {t('audit.tab_title')}
        </Link>
      </div>

      <FlagAnalyticsPanel slug={slug} envSlug={envSlug} flagKey={key} flagType={flag.type} />

      <EvaluationPanel slug={slug} envSlug={envSlug} flagKey={key} />

      <FlagChangeHistoryPanel slug={slug} flagKey={key} />

      {pendingDelete && (
        <DeleteConfirmModal
          flagKey={flag.key}
          isDeleting={deleteMutation.isPending}
          deleteFailed={deleteMutation.isError}
          onConfirm={() => deleteMutation.mutate()}
          onCancel={() => setPendingDelete(false)}
        />
      )}

      {pendingPromote && environments && (
        <PromoteDialog
          mode="single"
          projectSlug={slug}
          sourceEnvSlug={envSlug}
          flagKey={key}
          environments={environments}
          onClose={() => setPendingPromote(false)}
          onSuccess={() => void queryClient.invalidateQueries({ queryKey })}
        />
      )}
    </div>
  )
}

function FlagDetailCard({
  flag,
  slug,
  envSlug,
  isToggling,
  onToggle,
  onDeleteIntent,
  onPromoteIntent,
  onSaved,
}: {
  flag: FlagDetail
  slug: string
  envSlug: string
  isToggling: boolean
  onToggle: (enabled: boolean) => void
  onDeleteIntent: () => void
  onPromoteIntent: () => void
  onSaved: () => void
}) {
  const { t } = useTranslation('flags')
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState(flag.name)
  const [editVariants, setEditVariants] = useState<Variant[]>(flag.variants)
  const [editDefaultVariantKey, setEditDefaultVariantKey] = useState(flag.default_variant_key)
  const [saveError, setSaveError] = useState<string | null>(null)

  const updateMutation = useMutation({
    mutationFn: () =>
      patchJSON(`/api/v1/projects/${slug}/flags/${flag.key}`, {
        name: editName,
        variants: editVariants,
        default_variant_key: editDefaultVariantKey,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['flags', slug, envSlug] })
      setEditing(false)
      setSaveError(null)
      onSaved()
    },
    onError: (err) => {
      setSaveError(err instanceof APIError ? err.message : t('detail.save_failed'))
    },
  })

  function cancelEdit() {
    setEditName(flag.name)
    setEditVariants(flag.variants)
    setEditDefaultVariantKey(flag.default_variant_key)
    setSaveError(null)
    setEditing(false)
  }

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg">
      {/* Header */}
      <div className="flex items-start justify-between px-5 py-4 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <CopyableCode
            value={flag.key}
            aria-label={t('detail.copy_key_aria', { key: flag.key })}
          />
          <span className="text-xs text-[var(--color-text-muted)] bg-[var(--color-surface-elevated)] rounded px-1.5 py-0.5">
            {flag.type}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {!editing && (
            <>
              <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>
                {t('actions.edit', { ns: 'common' })}
              </Button>
              <Button variant="secondary" size="sm" onClick={onPromoteIntent}>
                {t('promote.button')}
              </Button>
            </>
          )}
          <Button variant="danger-outline" size="sm" aria-label={t('detail.delete_aria')} onClick={onDeleteIntent}>
            {t('actions.delete', { ns: 'common' })}
          </Button>
        </div>
      </div>

      {/* Body */}
      <div className="px-5 py-4 space-y-5">
        {/* Name */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('detail.name_label')}</label>
          {editing ? (
            <Input
              type="text"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              className="py-1.5 px-2"
            />
          ) : (
            <p className="text-sm text-[var(--color-text-primary)]">{flag.name}</p>
          )}
        </div>

        {/* Variants */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('detail.variants_label')}</label>
          <ul className="space-y-1">
            {(editing ? editVariants : flag.variants).map((v: Variant, i: number) => (
              <li key={v.key} className="flex items-center gap-2">
                <span className="font-mono text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5 w-24 shrink-0">
                  {v.key}
                </span>
                {editing ? (
                  <Input
                    type="text"
                    value={editVariants[i].name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => {
                      const updated = editVariants.map((ev: Variant, idx: number) =>
                        idx === i ? { ...ev, name: e.target.value } : ev,
                      )
                      setEditVariants(updated)
                    }}
                    aria-label={t('detail.variant_name_aria', { key: v.key })}
                    className="py-1 px-2"
                  />
                ) : (
                  <span className="text-sm text-[var(--color-text-primary)]">{v.name}</span>
                )}
              </li>
            ))}
          </ul>
        </div>

        {/* Default variant */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('detail.default_variant_label')}</label>
          {editing ? (
            <Select
              value={editDefaultVariantKey}
              onValueChange={setEditDefaultVariantKey}
              aria-label={t('detail.default_variant_aria')}
            >
              {editVariants.map((v: Variant) => (
                <SelectItem key={v.key} value={v.key}>
                  {v.key}
                </SelectItem>
              ))}
            </Select>
          ) : (
            <span className="font-mono text-xs text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5">
              {flag.default_variant_key}
            </span>
          )}
        </div>

        {/* Enabled toggle */}
        <div>
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
            {t('detail.status_label', { env: envSlug })}
          </label>
          <button
            onClick={() => onToggle(!flag.enabled)}
            disabled={isToggling}
            aria-pressed={flag.enabled}
            aria-label={flag.enabled ? t('toggle.disable') : t('toggle.enable')}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border transition-colors focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:opacity-60 ${
              flag.enabled
                ? 'bg-[rgba(16,217,168,0.08)] text-[var(--color-status-enabled)] border-[var(--color-status-enabled)] hover:bg-[rgba(16,217,168,0.08)] focus:ring-green-500
                : 'bg-[var(--color-surface)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:bg-[var(--color-surface-elevated)] focus:ring-[var(--color-accent)]
            }`}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${flag.enabled ? 'bg-[var(--color-status-enabled)]' : 'bg-[var(--color-surface-elevated)]'}`}
              aria-hidden="true"
            />
            {flag.enabled ? t('toggle.enabled') : t('toggle.disabled')}
          </button>
        </div>
      </div>

      {/* Edit actions */}
      {editing && (
        <div className="px-5 py-3 border-t border-[var(--color-border)] flex items-center gap-3">
          <Button
            onClick={() => updateMutation.mutate()}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t('states.saving', { ns: 'common' }) : t('actions.save', { ns: 'common' })}
          </Button>
          <Button
            variant="secondary"
            onClick={cancelEdit}
            disabled={updateMutation.isPending}
          >
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          {saveError && (
            <p className="text-xs text-[var(--color-status-error)]">{saveError}</p>
          )}
        </div>
      )}
    </div>
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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-dialog-title"
    >
      <div className="absolute inset-0 bg-black/30" onClick={onCancel} aria-hidden="true" />
      <div className="relative bg-[var(--color-surface)] rounded-lg shadow-lg max-w-sm w-full mx-4 p-6">
        <h2 id="delete-dialog-title" className="text-base font-semibold text-[var(--color-text-primary)]">
          {t('delete.title')}
        </h2>
        <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
          <Trans
            i18nKey="delete.body"
            ns="flags"
            values={{ key: flagKey }}
            components={{ mono: <span className="font-mono text-[var(--color-text-primary)]" /> }}
          />
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-[var(--color-status-error)]">{t('delete.failed')}</p>
        )}
        <div className="mt-5 flex justify-end gap-3">
          <Button autoFocus variant="secondary" onClick={onCancel} disabled={isDeleting}>
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button variant="danger" onClick={onConfirm} disabled={isDeleting}>
            {isDeleting ? t('states.deleting', { ns: 'common' }) : t('actions.delete', { ns: 'common' })}
          </Button>
        </div>
      </div>
    </div>
  )
}

function EnvironmentTogglePanel({ slug, flagKey }: { slug: string; flagKey: string }) {
  const { t } = useTranslation('flags')
  const { data, isLoading, error } = useQuery({
    queryKey: ['environments', slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(`/api/v1/projects/${slug}/environments`)
        .then((d) => d.environments),
  })

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg mt-4">
      <div className="px-5 py-3 border-b border-[var(--color-border)]">
        <h2 className="text-xs font-semibold text-[var(--color-text-secondary)] font-medium">{t('detail.environments_section')}</h2>
      </div>
      {isLoading ? (
        <EnvToggleSkeleton />
      ) : error ? (
        <div className="px-5 py-4">
          <p className="text-sm text-[var(--color-status-error)]">{t('detail.env_error')}</p>
        </div>
      ) : (
        <ul>
          {data!.map((env) => (
            <EnvironmentToggleRow key={env.id} slug={slug} env={env} flagKey={flagKey} />
          ))}
        </ul>
      )}
    </div>
  )
}

function EnvironmentToggleRow({
  slug,
  env,
  flagKey,
}: {
  slug: string
  env: Environment
  flagKey: string
}) {
  const { t } = useTranslation('flags')
  const queryKey = ['flag-env-state', slug, env.slug, flagKey]
  const queryClient = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<FlagEnvState>(`/api/v1/projects/${slug}/environments/${env.slug}/flags/${flagKey}`),
  })

  const toggleMutation = useMutation({
    mutationFn: (enabled: boolean) =>
      patchJSON<{ enabled: boolean }>(
        `/api/v1/projects/${slug}/environments/${env.slug}/flags/${flagKey}`,
        { enabled },
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  return (
    <li className="flex items-center justify-between px-5 py-3 border-b border-[var(--color-border)] last:border-0">
      <div className="flex items-center gap-2">
        <span className="text-sm text-[var(--color-text-primary)]">{env.name}</span>
        <span className="font-mono text-xs text-[var(--color-text-secondary)]">{env.slug}</span>
      </div>
      {isLoading ? (
        <div className="h-6 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" aria-label={t('detail.loading_aria')} />
      ) : error ? (
        <div className="flex items-center gap-2">
          <span className="text-xs text-[var(--color-status-error)]">{t('detail.env_row_error')}</span>
          <button
            onClick={() => void refetch()}
            className="text-xs text-[var(--color-accent)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
          >
            {t('actions.retry', { ns: 'common' })}
          </button>
        </div>
      ) : (
        <button
          onClick={() => toggleMutation.mutate(!data!.enabled)}
          disabled={toggleMutation.isPending}
          aria-pressed={data!.enabled}
          aria-label={data!.enabled ? t('toggle.disable_in_env', { env: env.name }) : t('toggle.enable_in_env', { env: env.name })}
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border transition-colors focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:opacity-60 ${
            data!.enabled
              ? 'bg-[rgba(16,217,168,0.08)] text-[var(--color-status-enabled)] border-[var(--color-status-enabled)] hover:bg-[rgba(16,217,168,0.08)] focus:ring-green-500'
              : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:bg-[var(--color-surface-elevated)] focus:ring-[var(--color-accent)]'
          }`}
        >
          <span
            className={`w-1.5 h-1.5 rounded-full ${data!.enabled ? 'bg-[var(--color-status-enabled)]' : 'bg-[var(--color-surface-elevated)]'}`}
            aria-hidden="true"
          />
          {data!.enabled ? t('toggle.enabled') : t('toggle.disabled')}
        </button>
      )}
    </li>
  )
}

function EnvToggleSkeleton() {
  return (
    <ul>
      {[1, 2, 3].map((i) => (
        <li
          key={i}
          className="flex items-center justify-between px-5 py-3 border-b border-[var(--color-border)] last:border-0"
        >
          <div className="flex items-center gap-2">
            <div className="h-4 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-3 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
          <div className="h-6 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </li>
      ))}
    </ul>
  )
}

interface EvalResponse {
  key: string
  enabled: boolean
  value: string | null
  reason: string
  type: string
}

function EvaluationPanel({
  slug,
  envSlug,
  flagKey,
}: {
  slug: string
  envSlug: string
  flagKey: string
}) {
  const { t } = useTranslation('flags')
  const [open, setOpen] = useState(false)
  const [contextInput, setContextInput] = useState('{}')
  const [jsonError, setJsonError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: (context: unknown) =>
      postJSON<EvalResponse>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags/${flagKey}/evaluate`,
        { context },
      ),
  })

  function handleInputChange(e: ChangeEvent<HTMLTextAreaElement>) {
    const val = e.target.value
    setContextInput(val)
    try {
      JSON.parse(val)
      setJsonError(null)
    } catch {
      setJsonError(t('eval.invalid_json'))
    }
  }

  function handleEvaluate() {
    let parsed: unknown
    try {
      parsed = JSON.parse(contextInput)
    } catch {
      setJsonError(t('eval.invalid_json'))
      return
    }
    setJsonError(null)
    mutation.mutate(parsed)
  }

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg mt-4">
      <button
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="w-full flex items-center justify-between px-5 py-3 text-left focus:outline-none focus:ring-2 focus:ring-inset focus:ring-[var(--color-accent)]"
      >
        <span className="text-xs font-semibold text-[var(--color-text-secondary)] font-medium">
          {t('eval.title')}
        </span>
        <span className="text-[var(--color-text-muted)] text-sm" aria-hidden="true">
          {open ? '▲' : '▼'}
        </span>
      </button>

      {open && (
        <div className="px-5 pb-5 border-t border-[var(--color-border)] pt-4 space-y-3">
          <div>
            <label
              htmlFor="eval-context"
              className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1"
            >
              {t('eval.context_label')}
            </label>
            <textarea
              id="eval-context"
              value={contextInput}
              onChange={handleInputChange}
              rows={4}
              spellCheck={false}
              className="w-full font-mono text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border)] rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] resize-y"
            />
            {jsonError && (
              <p className="mt-1 text-xs text-[var(--color-status-error)]" role="alert">
                {jsonError}
              </p>
            )}
          </div>

          <Button
            onClick={handleEvaluate}
            disabled={mutation.isPending || jsonError !== null}
          >
            {mutation.isPending ? t('eval.evaluating') : t('eval.evaluate')}
          </Button>

          {mutation.isError && (
            <p className="text-xs text-[var(--color-status-error)]" role="alert">
              {mutation.error instanceof APIError
                ? mutation.error.message
                : t('eval.eval_failed')}
            </p>
          )}

          {mutation.isSuccess && mutation.data && (
            <EvalResultDisplay result={mutation.data} />
          )}
        </div>
      )}
    </div>
  )
}

function EvalResultDisplay({ result }: { result: EvalResponse }) {
  const { t } = useTranslation('flags')
  const reasonLabel = REASON_KEY_MAP[result.reason] ? t(REASON_KEY_MAP[result.reason]) : result.reason

  return (
    <div className="rounded border border-[var(--color-border)] bg-[var(--color-surface-elevated)] px-4 py-3 space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-[var(--color-text-secondary)]">{t('eval.result_label')}</span>
        {result.value !== null ? (
          <span className="font-mono text-sm bg-[rgba(79,124,255,0.1)] text-[var(--color-accent)] border border-[rgba(79,124,255,0.3)] rounded px-2 py-0.5">
            {result.value}
          </span>
        ) : (
          <span
            className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border ${
              result.enabled
                ? 'bg-[rgba(16,217,168,0.08)] text-[var(--color-status-enabled)] border-[var(--color-status-enabled)]'
                : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)]'
            }`}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${result.enabled ? 'bg-[var(--color-status-enabled)]' : 'bg-[var(--color-surface-elevated)]'}`}
              aria-hidden="true"
            />
            {result.enabled ? t('toggle.enabled') : t('toggle.disabled')}
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-[var(--color-text-secondary)]">{t('eval.reason_label')}</span>
        <span className="text-xs text-[var(--color-text-primary)]">{reasonLabel}</span>
      </div>
    </div>
  )
}

interface HistoryEntry {
  id: string
  occurred_at: string
  actor_email: string
  action: string
  environment_slug: string
}

interface HistoryListResponse {
  entries: HistoryEntry[]
  next_cursor: string | null
}

const ACTION_LABELS: Record<string, string> = {
  'flag.created': 'history.action_flag_created',
  'flag.updated': 'history.action_flag_updated',
  'flag.variant_added': 'history.action_flag_variant_added',
  'flag.variant_renamed': 'history.action_flag_variant_renamed',
  'flag.variant_deleted': 'history.action_flag_variant_deleted',
  'flag.state_changed': 'history.action_flag_state_changed',
  'flag.deleted': 'history.action_flag_deleted',
  'flag.promoted': 'history.action_flag_promoted',
}

const ENV_FILTER_ALL = '__all__'

function FlagChangeHistoryPanel({ slug, flagKey }: { slug: string; flagKey: string }) {
  const { t } = useTranslation('flags')
  const [open, setOpen] = useState(false)
  const [envFilter, setEnvFilter] = useState(ENV_FILTER_ALL)

  const { data, isError } = useQuery<HistoryListResponse>({
    queryKey: ['flag-history', slug, flagKey],
    queryFn: () =>
      fetchJSON<HistoryListResponse>(
        `/api/v1/projects/${slug}/audit?flag_key=${encodeURIComponent(flagKey)}&limit=20`,
      ),
  })

  const allEntries = useMemo(() => data?.entries ?? [], [data])

  const distinctEnvs = useMemo(() => {
    const envs = new Set<string>()
    for (const e of allEntries) {
      if (e.environment_slug) envs.add(e.environment_slug)
    }
    return Array.from(envs).sort()
  }, [allEntries])

  const showEnvFilter = distinctEnvs.length >= 2

  const visibleEntries = useMemo(() => {
    if (envFilter === ENV_FILTER_ALL) return allEntries
    return allEntries.filter((e) => e.environment_slug === envFilter)
  }, [allEntries, envFilter])

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg mt-4">
      <button
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="w-full flex items-center justify-between px-5 py-3 text-left focus:outline-none focus:ring-2 focus:ring-inset focus:ring-[var(--color-accent)]"
      >
        <span className="text-xs font-semibold text-[var(--color-text-secondary)] font-medium">
          {t('history.title')}
        </span>
        <span className="text-[var(--color-text-muted)] text-sm" aria-hidden="true">
          {open ? '▲' : '▼'}
        </span>
      </button>

      {open && (
        <div className="border-t border-[var(--color-border)]">
          {isError ? (
            <div className="px-5 py-4" role="status">
              <p className="text-sm text-[var(--color-status-error)]">{t('history.error')}</p>
            </div>
          ) : (
            <>
              {showEnvFilter && (
                <div className="px-5 pt-4 pb-2">
                  <Select
                    value={envFilter}
                    onValueChange={setEnvFilter}
                    aria-label={t('history.env_filter_aria')}
                  >
                    <SelectItem value="__all__">{t('history.env_filter_all')}</SelectItem>
                    {distinctEnvs.map((env) => (
                      <SelectItem key={env} value={env}>
                        {env}
                      </SelectItem>
                    ))}
                  </Select>
                </div>
              )}
              {visibleEntries.length === 0 ? (
                <div className="px-5 py-8 text-center" role="status">
                  <p className="text-sm text-[var(--color-text-secondary)]">
                    {envFilter === '__all__'
                      ? t('history.empty')
                      : t('history.empty_filtered')}
                  </p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table
                    className="w-full text-sm"
                    aria-label={t('history.table_aria')}
                  >
                    <thead className="bg-[var(--color-surface-elevated)] border-b border-[var(--color-border)]">
                      <tr>
                        <th scope="col" className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                          {t('history.col_timestamp')}
                        </th>
                        <th scope="col" className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                          {t('history.col_environment')}
                        </th>
                        <th scope="col" className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                          {t('history.col_actor')}
                        </th>
                        <th scope="col" className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                          {t('history.col_action')}
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[var(--color-border)]">
                      {visibleEntries.map((entry) => (
                        <HistoryEntryRow key={entry.id} entry={entry} />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function HistoryEntryRow({ entry }: { entry: HistoryEntry }) {
  const { t } = useTranslation('flags')
  const actionLabel = ACTION_LABELS[entry.action]
    ? t(ACTION_LABELS[entry.action])
    : entry.action
  const absolute = formatAbsoluteDate(entry.occurred_at)
  const relative = formatRelativeDate(entry.occurred_at)

  return (
    <tr className="hover:bg-[var(--color-surface)]
      <td className="px-4 py-3 whitespace-nowrap">
        <time
          dateTime={entry.occurred_at}
          title={relative}
          className="text-xs text-[var(--color-text-secondary)] tabular-nums"
        >
          {absolute}
        </time>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {entry.environment_slug ? (
          <span className="font-mono text-xs text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5">
            {entry.environment_slug}
          </span>
        ) : (
          <span className="text-xs text-[var(--color-text-muted)]">{t('history.env_none')}</span>
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className="text-sm text-[var(--color-text-primary)]">{entry.actor_email}</span>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className="font-mono text-xs text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5">
          {actionLabel}
        </span>
      </td>
    </tr>
  )
}

function FlagDetailSkeleton() {
  return (
    <div className="p-6 max-w-2xl">
      <div className="h-4 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-4" />
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg">
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border)]">
          <div className="flex gap-2">
            <div className="h-6 w-28 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-6 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
          <div className="h-7 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </div>
        <div className="px-5 py-4 space-y-5">
          {[1, 2, 3, 4].map((i) => (
            <div key={i}>
              <div className="h-3 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-1" />
              <div className="h-5 w-48 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
