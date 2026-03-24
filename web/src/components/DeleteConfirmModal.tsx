import { useRef, useEffect, useState } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from './ui/Dialog'
import { Input } from './ui/Input'
import { Button } from './ui/Button'

interface DeleteConfirmModalProps {
  flagKey: string
  isDeleting: boolean
  deleteFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function DeleteConfirmModal({
  flagKey,
  isDeleting,
  deleteFailed,
  onConfirm,
  onCancel,
}: DeleteConfirmModalProps) {
  const { t } = useTranslation('flags')
  const [confirmValue, setConfirmValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const isMatch = confirmValue === flagKey

  useEffect(() => {
    // Auto-focus the input when the modal opens
    inputRef.current?.focus()
  }, [])

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onCancel() }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('delete.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-[var(--color-text-secondary)]">
          <Trans
            i18nKey="delete.body"
            ns="flags"
            values={{ key: flagKey }}
            components={{ mono: <span className="font-mono text-[var(--color-text-primary)]" /> }}
          />
        </p>
        <p className="mt-3 text-sm text-[var(--color-text-secondary)]">
          <Trans
            i18nKey="delete.confirm_instruction"
            ns="flags"
            values={{ key: flagKey }}
            components={{ mono: <span className="font-mono font-semibold text-[var(--color-text-primary)]" /> }}
          />
        </p>
        <Input
          ref={inputRef}
          className="mt-2"
          value={confirmValue}
          onChange={(e) => setConfirmValue(e.target.value)}
          placeholder={t('delete.confirm_placeholder')}
          aria-label={t('delete.confirm_aria')}
          autoComplete="off"
          spellCheck={false}
          disabled={isDeleting}
        />
        {deleteFailed && (
          <p className="mt-3 text-xs text-[var(--color-status-error)]">{t('delete.failed')}</p>
        )}
        <DialogFooter>
          <Button variant="secondary" onClick={onCancel} disabled={isDeleting}>
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={!isMatch || isDeleting}
            loading={isDeleting}
          >
            {isDeleting ? t('states.deleting', { ns: 'common' }) : t('actions.delete', { ns: 'common' })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
