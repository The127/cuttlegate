import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { postJSON, APIError } from '../api'
import { Button, Select, SelectItem, Dialog, DialogContent, DialogTitle } from './ui'

export interface FlagPromotionDiff {
  flag_key: string
  enabled_before: boolean
  enabled_after: boolean
  rules_added: number
  rules_removed: number
}

interface Environment {
  id: string
  slug: string
  name: string
}

interface PromoteDialogProps {
  /** Promotion mode: single flag or bulk (all flags in the source env). */
  mode: 'single' | 'bulk'
  /** Project slug. */
  projectSlug: string
  /** The source environment slug (the one being promoted from). */
  sourceEnvSlug: string
  /** Flag key — only required when mode === 'single'. */
  flagKey?: string
  /** All environments in the project, used to populate the target dropdown. */
  environments: Environment[]
  /**
   * Controls whether the dialog is open. Defaults to true so callers that use
   * conditional rendering (`{condition && <PromoteDialog />}`) continue to work
   * without changes.
   */
  open?: boolean
  onClose: () => void
  /** Called after a successful promotion so the caller can invalidate queries. */
  onSuccess: () => void
}

type DialogStep = 'select' | 'result'

export function PromoteDialog({
  mode,
  projectSlug,
  sourceEnvSlug,
  flagKey,
  environments,
  open = true,
  onClose,
  onSuccess,
}: PromoteDialogProps) {
  const { t } = useTranslation('flags')
  const [targetEnvSlug, setTargetEnvSlug] = useState('')
  const [step, setStep] = useState<DialogStep>('select')
  const [diffs, setDiffs] = useState<FlagPromotionDiff[]>([])
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  // Exclude the source env from the target list.
  const targetOptions = environments.filter((e) => e.slug !== sourceEnvSlug)

  const promoteMutation = useMutation({
    mutationFn: (): Promise<FlagPromotionDiff | { flags: FlagPromotionDiff[] }> => {
      if (mode === 'single' && flagKey) {
        return postJSON(
          `/api/v1/projects/${projectSlug}/environments/${sourceEnvSlug}/flags/${flagKey}/promote`,
          { target_env_slug: targetEnvSlug },
        )
      }
      return postJSON(
        `/api/v1/projects/${projectSlug}/environments/${sourceEnvSlug}/promote`,
        { target_env_slug: targetEnvSlug },
      )
    },
    onSuccess: (data) => {
      if ('flags' in data) {
        setDiffs(data.flags)
      } else {
        setDiffs([data as FlagPromotionDiff])
      }
      setStep('result')
      onSuccess()
    },
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 403) {
          setErrorMessage(t('promote.forbidden_error'))
          return
        }
        if (err.status === 400) {
          setErrorMessage(t('promote.same_env_error'))
          return
        }
      }
      setErrorMessage(t('promote.error'))
    },
  })

  const dialogTitle =
    mode === 'single' && flagKey
      ? t('promote.dialog_title_single', { key: flagKey })
      : t('promote.dialog_title_bulk', { source: sourceEnvSlug })

  function handleConfirm() {
    if (!targetEnvSlug) return
    setErrorMessage(null)
    promoteMutation.mutate()
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose() }}>
      <DialogContent>
        <DialogTitle>{dialogTitle}</DialogTitle>

        {step === 'select' ? (
          <SelectStep
            targetEnvSlug={targetEnvSlug}
            targetOptions={targetOptions}
            errorMessage={errorMessage}
            isPending={promoteMutation.isPending}
            onTargetChange={setTargetEnvSlug}
            onConfirm={handleConfirm}
            onClose={onClose}
          />
        ) : (
          <ResultStep diffs={diffs} onClose={onClose} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function SelectStep({
  targetEnvSlug,
  targetOptions,
  errorMessage,
  isPending,
  onTargetChange,
  onConfirm,
  onClose,
}: {
  targetEnvSlug: string
  targetOptions: Environment[]
  errorMessage: string | null
  isPending: boolean
  onTargetChange: (slug: string) => void
  onConfirm: () => void
  onClose: () => void
}) {
  const { t } = useTranslation('flags')
  return (
    <>
      <div className="mb-4 mt-4">
        <label
          className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1"
        >
          {t('promote.target_label')}
        </label>
        <Select
          value={targetEnvSlug}
          onValueChange={onTargetChange}
          placeholder={t('promote.select_target')}
          aria-label={t('promote.select_target_aria')}
          className="w-full"
        >
          {targetOptions.map((env) => (
            <SelectItem key={env.id} value={env.slug}>
              {env.name} ({env.slug})
            </SelectItem>
          ))}
        </Select>
      </div>

      {errorMessage && (
        <p className="mb-3 text-xs text-red-600 dark:text-red-400" role="alert">
          {errorMessage}
        </p>
      )}

      <div className="flex justify-end gap-3">
        <Button variant="secondary" onClick={onClose} disabled={isPending}>
          {t('actions.cancel', { ns: 'common' })}
        </Button>
        <Button
          onClick={onConfirm}
          disabled={!targetEnvSlug || isPending}
        >
          {isPending ? t('promote.confirming') : t('promote.confirm_button')}
        </Button>
      </div>
    </>
  )
}

function ResultStep({
  diffs,
  onClose,
}: {
  diffs: FlagPromotionDiff[]
  onClose: () => void
}) {
  const { t } = useTranslation('flags')
  return (
    <>
      <p className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-3 mt-4">{t('promote.result_title')}</p>
      <ul className="space-y-2 max-h-64 overflow-y-auto mb-4">
        {diffs.map((diff) => (
          <DiffRow key={diff.flag_key} diff={diff} />
        ))}
      </ul>
      <div className="flex justify-end">
        <Button onClick={onClose}>{t('promote.done_button')}</Button>
      </div>
    </>
  )
}

function DiffRow({ diff }: { diff: FlagPromotionDiff }) {
  const { t } = useTranslation('flags')
  const enabledChanged = diff.enabled_before !== diff.enabled_after
  const rulesChanged = diff.rules_added > 0 || diff.rules_removed > 0

  return (
    <li className="rounded border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-700/50 px-3 py-2 text-xs">
      <span className="font-mono font-medium text-gray-800 dark:text-gray-100">{diff.flag_key}</span>
      <div className="mt-1 space-y-0.5 text-gray-600 dark:text-gray-300">
        {enabledChanged && (
          <div>
            {t('promote.result_enabled_label')}:{' '}
            <span className={diff.enabled_before ? 'text-green-700 dark:text-green-300' : 'text-gray-500 dark:text-gray-400'}>
              {diff.enabled_before ? t('promote.result_enabled_on') : t('promote.result_enabled_off')}
            </span>
            {' \u2192 '}
            <span className={diff.enabled_after ? 'text-green-700 dark:text-green-300' : 'text-gray-500 dark:text-gray-400'}>
              {diff.enabled_after ? t('promote.result_enabled_on') : t('promote.result_enabled_off')}
            </span>
          </div>
        )}
        {diff.rules_added > 0 && (
          <div>{t('promote.result_rules_added', { count: diff.rules_added })}</div>
        )}
        {diff.rules_removed > 0 && (
          <div>{t('promote.result_rules_removed', { count: diff.rules_removed })}</div>
        )}
        {!enabledChanged && !rulesChanged && (
          <div className="text-gray-400 dark:text-gray-500">{t('promote.result_no_rule_change')}</div>
        )}
      </div>
    </li>
  )
}
