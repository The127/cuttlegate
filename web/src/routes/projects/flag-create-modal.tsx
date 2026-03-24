import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { postJSON, APIError } from '../../api'
import {
  Button,
  Input,
  Label,
  Select,
  SelectItem,
  CopyableCode,
} from '../../components/ui'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogCloseButton,
} from '../../components/ui/Dialog'

// --- Flag key validation ---

const FLAG_KEY_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_KEY_LENGTH = 128

function validateKeyLocally(key: string, t: (k: string, opts?: Record<string, unknown>) => string): string | null {
  if (key.length === 0) return null
  if (key.length > MAX_KEY_LENGTH) return t('create.key_too_long', { max: MAX_KEY_LENGTH })
  if (!FLAG_KEY_RE.test(key)) return t('create.key_invalid')
  return null
}

// --- SDK prompt helpers ---

const SDK_TABS = ['go', 'js', 'python'] as const
type SdkTab = (typeof SDK_TABS)[number]

function safeGetTab(): SdkTab {
  try {
    const stored = localStorage.getItem('cg:sdk_tab')
    if (stored === 'go' || stored === 'js' || stored === 'python') return stored
  } catch {
    // localStorage unavailable
  }
  return 'go'
}

function safeSetTab(tab: SdkTab): void {
  try {
    localStorage.setItem('cg:sdk_tab', tab)
  } catch {
    // QuotaExceededError or security policy
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
    // QuotaExceededError or security policy
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
      className="mt-5 border border-[var(--color-border)] rounded-lg overflow-hidden"
    >
      <div className="px-4 pt-4 pb-3 bg-[var(--color-surface-elevated)] border-b border-[var(--color-border)]">
        <p className="text-sm font-medium text-[var(--color-text-primary)]">
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
                  ? 'border-[var(--color-accent)] text-[var(--color-accent)] bg-[var(--color-surface)]'
                  : 'border-[var(--color-border)] text-[var(--color-text-secondary)] bg-[var(--color-surface)] hover:border-[var(--color-border)]'
              ].join(' ')}
            >
              {t(`create.sdk_prompt.tab_${tab}`)}
            </button>
          ))}
        </div>
      </div>
      <pre className="p-4 text-xs font-mono overflow-x-auto bg-[var(--color-surface)] text-[var(--color-text-primary)] leading-relaxed">
        {buildSnippet(activeTab, flagKey)}
      </pre>
      <div className="flex justify-end px-4 pb-3 bg-[var(--color-surface)]">
        <button
          onClick={handleDismiss}
          aria-label={t('create.sdk_prompt.dismiss_aria')}
          className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
        >
          {t('create.sdk_prompt.dismiss')}
        </button>
      </div>
    </section>
  )
}

// --- CreateFlagModal ---

interface CreateFlagModalProps {
  slug: string
  onCreated: () => void
  onCancel: () => void
}

export function CreateFlagModal({ slug, onCreated, onCancel }: CreateFlagModalProps) {
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
            <p className="text-sm text-[var(--color-text-secondary)] mb-4">
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
                <Label htmlFor="flag-key" className="text-xs text-[var(--color-text-secondary)] mb-1">{t('create.key_label')}</Label>
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
                  <p id="flag-key-error" className="mt-1 text-xs text-[var(--color-status-error)]">
                    {keyError}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="flag-name" className="text-xs text-[var(--color-text-secondary)] mb-1">{t('create.name_label')}</Label>
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
                <Label htmlFor="flag-type" className="text-xs text-[var(--color-text-secondary)] mb-1">{t('create.type_label')}</Label>
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
                <p className="text-xs text-[var(--color-status-error)]">{serverError}</p>
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
