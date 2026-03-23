import { createRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, postJSON, deleteRequest, APIError } from '../../api'
import { formatRelativeDate } from '../../utils/date'
import { Button } from '../../components/ui/Button'
import { TierBadge } from '../../components/ui/TierBadge'
import { TierSelector } from '../../components/ui/TierSelector'
import type { ToolCapabilityTier } from '../../components/ui/TierBadge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../../components/ui/Dialog'

interface Environment {
  id: string
  name: string
  slug: string
}

interface APIKey {
  id: string
  name: string
  display_prefix: string
  capability_tier: ToolCapabilityTier
  created_at: string
  revoked_at?: string
}

interface CreateAPIKeyResult {
  id: string
  name: string
  display_prefix: string
  capability_tier: ToolCapabilityTier
  created_at: string
  key: string
}

export const apiKeyListRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/api-keys',
  component: APIKeyPage,
})

function APIKeyPage() {
  const { t } = useTranslation('projects')
  const { slug } = apiKeyListRoute.useParams()
  const queryClient = useQueryClient()

  const envsQuery = useQuery({
    queryKey: ['environments', slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${slug}/environments`,
      ).then((d) => d.environments),
  })

  const [selectedEnvSlug, setSelectedEnvSlug] = useState<string | null>(null)

  // Default to first environment once loaded
  const envSlug =
    selectedEnvSlug ??
    (envsQuery.data && envsQuery.data.length > 0 ? envsQuery.data[0].slug : null)

  const keysQuery = useQuery({
    queryKey: ['api-keys', slug, envSlug],
    queryFn: () =>
      fetchJSON<{ api_keys: APIKey[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/api-keys`,
      ).then((d) => d.api_keys.filter((k) => !k.revoked_at)),
    enabled: envSlug !== null,
  })

  const [showCreate, setShowCreate] = useState(false)
  const [pendingRevoke, setPendingRevoke] = useState<APIKey | null>(null)

  const revokeMutation = useMutation({
    mutationFn: (keyId: string) =>
      deleteRequest(
        `/api/v1/projects/${slug}/environments/${envSlug!}/api-keys/${keyId}`,
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['api-keys', slug, envSlug] })
      setPendingRevoke(null)
    },
  })

  const keys = keysQuery.data ?? []
  const isLastKey = keys.length === 1

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">{t('api_keys.title')}</h1>
        <Button
          onClick={() => setShowCreate(true)}
          disabled={envSlug === null}
        >
          {t('api_keys.new_key')}
        </Button>
      </div>

      {envsQuery.isLoading ? (
        <div className="h-8 w-48 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-4" />
      ) : envsQuery.isError ? (
        <p className="text-sm text-[var(--color-status-error)] mb-4">{t('api_keys.environments_error')}</p>
      ) : envsQuery.data!.length === 0 ? (
        <p className="text-sm text-[var(--color-text-secondary)]">
          {t('api_keys.no_environments')}
        </p>
      ) : (
        <>
          <div className="mb-4">
            <label
              htmlFor="env-selector"
              className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1"
            >
              {t('api_keys.environment_label')}
            </label>
            <select
              id="env-selector"
              value={envSlug ?? ''}
              onChange={(e) => setSelectedEnvSlug(e.target.value)}
              className="text-sm border border-[var(--color-border)] rounded px-2 py-1.5 bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            >
              {envsQuery.data!.map((env) => (
                <option key={env.id} value={env.slug}>
                  {env.name}
                </option>
              ))}
            </select>
          </div>

          {keysQuery.isLoading ? (
            <APIKeyListSkeleton />
          ) : keysQuery.isError ? (
            <div>
              <span className="text-sm text-[var(--color-status-error)]">{t('api_keys.error')} </span>
              <button
                onClick={() => void keysQuery.refetch()}
                className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
              >
                {t('actions.retry', { ns: 'common' })}
              </button>
            </div>
          ) : keys.length === 0 ? (
            <APIKeyEmptyState onCreateClick={() => setShowCreate(true)} />
          ) : (
            <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
              {keys.map((key) => (
                <APIKeyRow
                  key={key.id}
                  apiKey={key}
                  onRevokeIntent={() => setPendingRevoke(key)}
                />
              ))}
            </ul>
          )}
        </>
      )}

      {envSlug && (
        <CreateAPIKeyModal
          open={showCreate}
          projectSlug={slug}
          envSlug={envSlug}
          onCreated={() => {
            setShowCreate(false)
            void queryClient.invalidateQueries({ queryKey: ['api-keys', slug, envSlug] })
          }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      <RevokeAPIKeyModal
        open={pendingRevoke !== null}
        apiKey={pendingRevoke}
        isLastKey={isLastKey}
        isRevoking={revokeMutation.isPending}
        revokeFailed={revokeMutation.isError}
        onConfirm={() => pendingRevoke && revokeMutation.mutate(pendingRevoke.id)}
        onCancel={() => setPendingRevoke(null)}
      />
    </div>
  )
}

function APIKeyRow({
  apiKey,
  onRevokeIntent,
}: {
  apiKey: APIKey
  onRevokeIntent: () => void
}) {
  const { t } = useTranslation('projects')
  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5 shrink-0">
          cg_{apiKey.display_prefix}…
        </span>
        <span className="text-sm text-[var(--color-text-primary)] truncate">{apiKey.name}</span>
        <TierBadge tier={apiKey.capability_tier} />
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <time
          dateTime={apiKey.created_at}
          className="text-xs text-[var(--color-text-muted)]"
          title={new Date(apiKey.created_at).toLocaleString()}
        >
          {formatRelativeDate(apiKey.created_at)}
        </time>
        <button
          onClick={onRevokeIntent}
          aria-label={t('api_keys.revoke_aria', { name: apiKey.name })}
          className="px-2 py-1 text-xs font-medium text-[var(--color-status-error)] border border-[var(--color-status-error)] rounded hover:bg-[rgba(248,113,113,0.08)] focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)]"
        >
          {t('api_keys.revoke')}
        </button>
      </div>
    </li>
  )
}

function APIKeyEmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('api_keys.empty')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('api_keys.new_key')}
      </Button>
    </div>
  )
}

function APIKeyListSkeleton() {
  return (
    <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
      {[1, 2, 3].map((i) => (
        <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
          <div className="flex items-center gap-3">
            <div className="h-6 w-36 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-4 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
          <div className="flex items-center gap-3">
            <div className="h-3 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-6 w-14 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
        </li>
      ))}
    </ul>
  )
}

type CreatePhase = { type: 'form' } | { type: 'show'; key: string; name: string }

function CreateAPIKeyModal({
  open,
  projectSlug,
  envSlug,
  onCreated,
  onCancel,
}: {
  open: boolean
  projectSlug: string
  envSlug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  const [phase, setPhase] = useState<CreatePhase>({ type: 'form' })
  const [name, setName] = useState('')
  const [tier, setTier] = useState<ToolCapabilityTier>('read')
  const [serverError, setServerError] = useState<string | null>(null)

  // Security: plaintext key is ephemeral — held in component state only, never cached.
  // The mutation result is not stored in TanStack Query; state is lost when the modal closes.
  const createMutation = useMutation({
    mutationFn: () =>
      postJSON<CreateAPIKeyResult>(
        `/api/v1/projects/${projectSlug}/environments/${envSlug}/api-keys`,
        { name, capability_tier: tier },
      ),
    onSuccess: (result) => {
      setPhase({ type: 'show', key: result.key, name: result.name })
    },
    onError: (err) => {
      setServerError(
        err instanceof APIError ? err.message : t('api_keys.server_error'),
      )
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setServerError(null)
    createMutation.mutate()
  }

  if (phase.type === 'show') {
    return (
      <ShowOnceModal
        open={open}
        projectSlug={projectSlug}
        envSlug={envSlug}
        keyName={phase.name}
        plaintextKey={phase.key}
        onDone={onCreated}
      />
    )
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('api_keys.create_title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="key-name" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
              {t('api_keys.name_label')}
            </label>
            <input
              id="key-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                setServerError(null)
              }}
              placeholder={t('api_keys.name_placeholder')}
              className="w-full text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border)] rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
              {t('api_keys.capability_tier_label')}
            </label>
            <TierSelector value={tier} onChange={setTier} />
          </div>
          {serverError && <p className="text-xs text-[var(--color-status-error)]">{serverError}</p>}
          <DialogFooter>
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
              loading={createMutation.isPending}
              disabled={!name.trim()}
            >
              {t('api_keys.create_button')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function ShowOnceModal({
  open,
  projectSlug,
  envSlug,
  keyName,
  plaintextKey,
  onDone,
}: {
  open: boolean
  projectSlug: string
  envSlug: string
  keyName: string
  plaintextKey: string
  onDone: () => void
}) {
  const { t } = useTranslation('projects')
  const [copied, setCopied] = useState(false)
  const copyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (copyTimerRef.current !== null) clearTimeout(copyTimerRef.current)
    }
  }, [])

  const curlSnippet =
    `curl -X POST https://<your-host>/api/v1/projects/${projectSlug}/environments/${envSlug}/flags/<flag-key>/evaluate \\\n` +
    `  -H "Authorization: Bearer ${plaintextKey}" \\\n` +
    `  -H "Content-Type: application/json" \\\n` +
    `  -d '{"context": {"key": "user-123"}}'`

  function copyKey() {
    void navigator.clipboard
      .writeText(plaintextKey)
      .then(() => {
        setCopied(true)
        if (copyTimerRef.current !== null) clearTimeout(copyTimerRef.current)
        copyTimerRef.current = setTimeout(() => setCopied(false), 2000)
      })
      .catch(() => {})
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onDone() }}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('api_keys.show_once_title', { name: keyName })}</DialogTitle>
        </DialogHeader>
        <p className="text-xs text-[var(--color-status-warning)] bg-[rgba(251,191,36,0.08)] border border-[var(--color-status-warning)] rounded px-3 py-2 mb-4">
          {t('api_keys.show_once_warning')}
        </p>

        <div className="mb-4">
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('api_keys.your_key_label')}</label>
          <div className="flex items-center gap-2">
            <code className="flex-1 font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-3 py-2 break-all select-all">
              {plaintextKey}
            </code>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={copyKey}
              aria-label={t('api_keys.copy_aria')}
            >
              {copied ? t('api_keys.copied') : t('api_keys.copy')}
            </Button>
          </div>
        </div>

        <div className="mb-6">
          <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
            {t('api_keys.getting_started_label')}
          </label>
          <pre className="font-mono text-xs text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-3 py-2 overflow-x-auto whitespace-pre">
            {curlSnippet}
          </pre>
        </div>

        <DialogFooter>
          <Button onClick={onDone}>
            {t('api_keys.done')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function RevokeAPIKeyModal({
  open,
  apiKey,
  isLastKey,
  isRevoking,
  revokeFailed,
  onConfirm,
  onCancel,
}: {
  open: boolean
  apiKey: APIKey | null
  isLastKey: boolean
  isRevoking: boolean
  revokeFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('api_keys.revoke_title')}</DialogTitle>
          {apiKey && (
            <DialogDescription>
              {t('api_keys.revoke_body', {
                prefix: `cg_${apiKey.display_prefix}…`,
                name: apiKey.name,
              })}
            </DialogDescription>
          )}
        </DialogHeader>
        {isLastKey && (
          <p className="text-sm text-[var(--color-status-warning)] bg-[rgba(251,191,36,0.08)] border border-[var(--color-status-warning)] rounded px-3 py-2">
            {t('api_keys.revoke_last_warning')}
          </p>
        )}
        {revokeFailed && (
          <p className="mt-3 text-xs text-[var(--color-status-error)]">{t('api_keys.revoke_failed')}</p>
        )}
        <DialogFooter>
          <Button
            autoFocus
            type="button"
            variant="secondary"
            onClick={onCancel}
            disabled={isRevoking}
          >
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={isRevoking}
            onClick={onConfirm}
          >
            {t('api_keys.revoke')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
