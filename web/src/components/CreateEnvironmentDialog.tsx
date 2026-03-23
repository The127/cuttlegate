import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { postJSON, APIError } from '../api'
import { Button, Input, Label } from './ui'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from './ui/Dialog'

export interface CreateEnvironmentDialogProps {
  open: boolean
  projectSlug: string
  onCreated: () => void
  onCancel: () => void
}

const SLUG_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_SLUG_LENGTH = 128

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/^-+|-+$/g, '')
}

function validateSlug(
  slug: string,
  t: (k: string, opts?: Record<string, unknown>) => string,
): string | null {
  if (slug.length === 0) return null
  if (slug.length > MAX_SLUG_LENGTH) return t('environments.slug_too_long', { max: MAX_SLUG_LENGTH })
  if (!SLUG_RE.test(slug)) return t('environments.slug_invalid')
  return null
}

export function CreateEnvironmentDialog({
  open,
  projectSlug,
  onCreated,
  onCancel,
}: CreateEnvironmentDialogProps) {
  const { t } = useTranslation('projects')
  const queryClient = useQueryClient()

  const [name, setName] = useState('')
  const [envSlug, setEnvSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [slugError, setSlugError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)

  function reset() {
    setName('')
    setEnvSlug('')
    setSlugTouched(false)
    setSlugError(null)
    setServerError(null)
  }

  const createMutation = useMutation({
    mutationFn: () =>
      postJSON(`/api/v1/projects/${projectSlug}/environments`, {
        name,
        slug: envSlug,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['environments', projectSlug] })
      reset()
      onCreated()
    },
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setSlugError(t('environments.slug_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setSlugError(err.message)
          return
        }
      }
      setServerError(
        err instanceof APIError ? err.message : t('environments.server_error'),
      )
    },
  })

  function handleNameChange(value: string) {
    setName(value)
    if (!slugTouched) {
      const generated = slugify(value)
      setEnvSlug(generated)
      setSlugError(validateSlug(generated, t))
    }
  }

  function handleSlugChange(value: string) {
    setEnvSlug(value)
    setSlugTouched(true)
    setSlugError(null)
    setServerError(null)
  }

  function handleSlugBlur() {
    setSlugError(validateSlug(envSlug, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const err = validateSlug(envSlug, t)
    if (err) {
      setSlugError(err)
      return
    }
    if (!envSlug) {
      setSlugError(t('environments.slug_required'))
      return
    }
    setServerError(null)
    createMutation.mutate()
  }

  function handleCancel() {
    reset()
    onCancel()
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) handleCancel()
  }

  const canSubmit = name.trim() !== '' && envSlug !== '' && !slugError && !createMutation.isPending

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('environments.create_title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="create-env-name">{t('environments.name_label')}</Label>
            <Input
              id="create-env-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder={t('environments.name_placeholder')}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="create-env-slug">{t('environments.slug_label')}</Label>
            <Input
              id="create-env-slug"
              type="text"
              value={envSlug}
              onChange={(e) => handleSlugChange(e.target.value)}
              onBlur={handleSlugBlur}
              placeholder={t('environments.slug_placeholder')}
              hasError={!!slugError}
              aria-describedby={slugError ? 'create-env-slug-error' : undefined}
              className="mt-1 font-mono"
            />
            {slugError && (
              <p id="create-env-slug-error" role="alert" className="mt-1 text-xs text-[var(--color-status-error)]">
                {slugError}
              </p>
            )}
          </div>
          {serverError && (
            <p role="alert" className="text-xs text-[var(--color-status-error)]">
              {serverError}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={handleCancel}
              disabled={createMutation.isPending}
            >
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={createMutation.isPending}
              disabled={!canSubmit}
            >
              {t('environments.create_button')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
