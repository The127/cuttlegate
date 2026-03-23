import { createRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, postJSON, patchJSON, putJSON, deleteRequest, APIError } from '../../api'
import { formatRelativeDate } from '../../utils/date'
import { Button } from '../../components/ui/Button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogCloseButton,
} from '../../components/ui/Dialog'

interface Segment {
  id: string
  slug: string
  name: string
  projectId: string
  createdAt: string
  memberCount: number
}

export const segmentListRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/segments',
  component: SegmentListPage,
})

function SegmentListPage() {
  const { t } = useTranslation('segments')
  const { slug } = segmentListRoute.useParams()
  const queryClient = useQueryClient()
  const queryKey = ['segments', slug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ segments: Segment[] }>(`/api/v1/projects/${slug}/segments`).then(
        (d) => d.segments,
      ),
  })

  const deleteMutation = useMutation({
    mutationFn: (segmentSlug: string) =>
      deleteRequest(`/api/v1/projects/${slug}/segments/${segmentSlug}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  const [showCreate, setShowCreate] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Segment | null>(null)
  const [editingSegment, setEditingSegment] = useState<Segment | null>(null)
  const [managingSegment, setManagingSegment] = useState<Segment | null>(null)

  if (isLoading) return <SegmentListSkeleton />
  if (isError)
    return (
      <div className="p-6">
        <span className="text-sm text-[var(--color-status-error)]">{t('error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  const segments = data ?? []

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">{t('title')}</h1>
        <Button onClick={() => setShowCreate(true)}>
          {t('new_segment')}
        </Button>
      </div>

      {segments.length === 0 ? (
        <SegmentEmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
          {segments.map((seg) => (
            <SegmentRow
              key={seg.id}
              segment={seg}
              onEdit={() => setEditingSegment(seg)}
              onManageMembers={() => setManagingSegment(seg)}
              onDeleteIntent={() => setPendingDelete(seg)}
            />
          ))}
        </ul>
      )}

      <CreateSegmentModal
        open={showCreate}
        slug={slug}
        onCreated={() => {
          setShowCreate(false)
          void queryClient.invalidateQueries({ queryKey })
        }}
        onCancel={() => setShowCreate(false)}
      />

      <EditSegmentModal
        open={editingSegment !== null}
        projectSlug={slug}
        segment={editingSegment}
        onSaved={(updated) => {
          setEditingSegment(null)
          queryClient.setQueryData<Segment[]>(queryKey, (prev) =>
            prev?.map((s) => (s.id === updated.id ? updated : s)),
          )
        }}
        onCancel={() => setEditingSegment(null)}
      />

      <ManageMembersModal
        open={managingSegment !== null}
        projectSlug={slug}
        segment={managingSegment}
        onCancel={() => setManagingSegment(null)}
      />

      <DeleteSegmentModal
        open={pendingDelete !== null}
        segment={pendingDelete}
        isDeleting={deleteMutation.isPending}
        deleteFailed={deleteMutation.isError}
        onConfirm={() => {
          if (!pendingDelete) return
          deleteMutation.mutate(pendingDelete.slug, {
            onSuccess: () => setPendingDelete(null),
          })
        }}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  )
}

function SegmentRow({
  segment,
  onEdit,
  onManageMembers,
  onDeleteIntent,
}: {
  segment: Segment
  onEdit: () => void
  onManageMembers: () => void
  onDeleteIntent: () => void
}) {
  const { t } = useTranslation('segments')
  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <span className="font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5 shrink-0">
          {segment.slug}
        </span>
        <span className="text-sm text-[var(--color-text-primary)] truncate">{segment.name}</span>
        <span className="text-xs text-[var(--color-text-muted)] shrink-0">
          {t('members_count', { count: segment.memberCount })}
        </span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <time
          dateTime={segment.createdAt}
          className="text-xs text-[var(--color-text-muted)]"
          title={new Date(segment.createdAt).toLocaleString()}
        >
          {formatRelativeDate(segment.createdAt)}
        </time>
        <Button
          variant="secondary"
          size="sm"
          onClick={onManageMembers}
        >
          {t('members.members_button')}
        </Button>
        <button
          onClick={onEdit}
          aria-label={t('edit_aria', { slug: segment.slug })}
          className="text-[var(--color-text-muted)] hover:text-[var(--color-accent)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded p-0.5"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="w-4 h-4"
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
          </svg>
        </button>
        <button
          onClick={onDeleteIntent}
          aria-label={t('delete_aria', { slug: segment.slug })}
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
      </div>
    </li>
  )
}

function SegmentEmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('segments')
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('empty_state')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('new_segment')}
      </Button>
    </div>
  )
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

function validateSlug(slug: string, t: (k: string, opts?: Record<string, unknown>) => string): string | null {
  if (slug.length === 0) return null
  if (slug.length > MAX_SLUG_LENGTH) return t('create.slug_too_long', { max: MAX_SLUG_LENGTH })
  if (!SLUG_RE.test(slug)) return t('create.slug_invalid')
  return null
}

function CreateSegmentModal({
  open,
  slug,
  onCreated,
  onCancel,
}: {
  open: boolean
  slug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('segments')
  const [name, setName] = useState('')
  const [segSlug, setSegSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [slugError, setSlugError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () => postJSON(`/api/v1/projects/${slug}/segments`, { name, slug: segSlug }),
    onSuccess: () => onCreated(),
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setSlugError(t('create.slug_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setSlugError(err.message)
          return
        }
      }
      setServerError(err instanceof APIError ? err.message : t('create.server_error'))
    },
  })

  function handleNameChange(value: string) {
    setName(value)
    if (!slugTouched) {
      const generated = slugify(value)
      setSegSlug(generated)
      setSlugError(validateSlug(generated, t))
    }
  }

  function handleSlugChange(value: string) {
    setSegSlug(value)
    setSlugTouched(true)
    setSlugError(null)
    setServerError(null)
  }

  function handleSlugBlur() {
    setSlugError(validateSlug(segSlug, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const err = validateSlug(segSlug, t)
    if (err) { setSlugError(err); return }
    if (!segSlug) { setSlugError(t('create.slug_required')); return }
    setServerError(null)
    createMutation.mutate()
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('create.title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="seg-name" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
              {t('create.name_label')}
            </label>
            <input
              id="seg-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder={t('create.name_placeholder')}
              className="w-full text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border)] rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            />
          </div>
          <div>
            <label htmlFor="seg-slug" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
              {t('create.slug_label')}
            </label>
            <input
              id="seg-slug"
              type="text"
              value={segSlug}
              onChange={(e) => handleSlugChange(e.target.value)}
              onBlur={handleSlugBlur}
              placeholder={t('create.slug_placeholder')}
              aria-invalid={!!slugError}
              aria-describedby={slugError ? 'seg-slug-error' : undefined}
              className={`w-full font-mono text-sm border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
                slugError ? 'border-[var(--color-status-error)] focus:ring-[var(--color-status-error)]' : 'border-[var(--color-border)] focus:ring-[var(--color-accent)]'
              }`}
            />
            {slugError && (
              <p id="seg-slug-error" className="mt-1 text-xs text-[var(--color-status-error)]">
                {slugError}
              </p>
            )}
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
              disabled={!!slugError || !name.trim() || !segSlug}
            >
              {t('actions.create', { ns: 'common' })}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function EditSegmentModal({
  open,
  projectSlug,
  segment,
  onSaved,
  onCancel,
}: {
  open: boolean
  projectSlug: string
  segment: Segment | null
  onSaved: (updated: Segment) => void
  onCancel: () => void
}) {
  const { t } = useTranslation('segments')
  const [name, setName] = useState(segment?.name ?? '')
  const [serverError, setServerError] = useState<string | null>(null)

  useEffect(() => {
    if (segment) setName(segment.name)
  }, [segment])

  const updateMutation = useMutation({
    mutationFn: () => {
      if (!segment) return Promise.reject(new Error('No segment'))
      return patchJSON<Segment>(`/api/v1/projects/${projectSlug}/segments/${segment.slug}`, { name })
    },
    onSuccess: (updated) => onSaved(updated),
    onError: (err) => {
      setServerError(err instanceof APIError ? err.message : t('edit.server_error'))
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setServerError(null)
    updateMutation.mutate()
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('edit.title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="edit-seg-name" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
              {t('edit.name_label')}
            </label>
            <input
              id="edit-seg-name"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => { setName(e.target.value); setServerError(null) }}
              className="w-full text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border)] rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('edit.slug_label')}</label>
            <div className="font-mono text-sm text-[var(--color-text-muted)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-1.5 select-none">
              {segment?.slug}
            </div>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">{t('edit.slug_immutable')}</p>
          </div>
          {serverError && <p className="text-xs text-[var(--color-status-error)]">{serverError}</p>}
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={updateMutation.isPending}
            >
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button
              type="submit"
              loading={updateMutation.isPending}
              disabled={!name.trim()}
            >
              {t('actions.save', { ns: 'common' })}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function ManageMembersModal({
  open,
  projectSlug,
  segment,
  onCancel,
}: {
  open: boolean
  projectSlug: string
  segment: Segment | null
  onCancel: () => void
}) {
  const { t } = useTranslation('segments')
  const queryClient = useQueryClient()
  const membersKey = ['segments', projectSlug, segment?.slug, 'members']

  const { data: fetchedMembers, isLoading, isError } = useQuery({
    queryKey: membersKey,
    queryFn: () =>
      fetchJSON<{ members: string[] }>(
        `/api/v1/projects/${projectSlug}/segments/${segment!.slug}/members`,
      ).then((d) => d.members),
    enabled: open && segment !== null,
  })

  const [members, setMembers] = useState<string[]>([])
  const [addKey, setAddKey] = useState('')
  const [addError, setAddError] = useState<string | null>(null)
  const [bulkText, setBulkText] = useState('')
  const [showBulk, setShowBulk] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    if (fetchedMembers !== undefined) {
      setMembers(fetchedMembers)
    }
  }, [fetchedMembers])

  const saveMutation = useMutation({
    mutationFn: (keys: string[]) =>
      putJSON(`/api/v1/projects/${projectSlug}/segments/${segment!.slug}/members`, {
        members: keys,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: membersKey })
      setSaveError(null)
    },
    onError: (err) => {
      setSaveError(err instanceof APIError ? err.message : t('members.save_failed'))
    },
  })

  function handleAddMember() {
    const key = addKey.trim()
    if (!key) return
    if (members.includes(key)) {
      setAddError(t('members.duplicate_key'))
      return
    }
    const next = [...members, key]
    setMembers(next)
    setAddKey('')
    setAddError(null)
    saveMutation.mutate(next)
  }

  function handleRemoveMember(key: string) {
    const next = members.filter((m) => m !== key)
    setMembers(next)
    saveMutation.mutate(next)
  }

  function handleBulkApply() {
    const keys = bulkText
      .split('\n')
      .map((k) => k.trim())
      .filter(Boolean)
    const deduped = [...new Set(keys)]
    setMembers(deduped)
    setShowBulk(false)
    saveMutation.mutate(deduped)
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('members.title', { slug: segment?.slug })}</DialogTitle>
          <DialogCloseButton />
        </DialogHeader>

        {isLoading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-8 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            ))}
          </div>
        ) : isError ? (
          <p className="text-sm text-[var(--color-status-error)]">{t('members.failed_to_load')}</p>
        ) : (
          <>
            <div className="flex gap-2 mb-3">
              <input
                type="text"
                value={addKey}
                onChange={(e) => { setAddKey(e.target.value); setAddError(null) }}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleAddMember() } }}
                placeholder={t('members.add_placeholder')}
                aria-label={t('members.add_aria')}
                aria-invalid={!!addError}
                className={`flex-1 font-mono text-sm border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
                  addError ? 'border-[var(--color-status-error)] focus:ring-[var(--color-status-error)]' : 'border-[var(--color-border)] focus:ring-[var(--color-accent)]'
                }`}
              />
              <Button
                type="button"
                onClick={handleAddMember}
                disabled={!addKey.trim() || saveMutation.isPending}
              >
                {t('members.add_button')}
              </Button>
            </div>
            {addError && <p className="mb-2 text-xs text-[var(--color-status-error)]">{addError}</p>}

            <div className="overflow-y-auto max-h-48 border border-[var(--color-border)] rounded-lg">
              {members.length === 0 ? (
                <p className="text-sm text-[var(--color-text-muted)] text-center py-8">{t('members.no_members')}</p>
              ) : (
                <ul className="divide-y divide-[var(--color-border)]">
                  {members.map((key) => (
                    <li key={key} className="flex items-center justify-between px-3 py-2">
                      <span className="font-mono text-sm text-[var(--color-text-primary)]">{key}</span>
                      <button
                        onClick={() => handleRemoveMember(key)}
                        disabled={saveMutation.isPending}
                        aria-label={t('members.remove_aria', { key })}
                        className="text-[var(--color-text-muted)] hover:text-[var(--color-status-error)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded p-0.5 disabled:opacity-50"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                          <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                        </svg>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div className="mt-3">
              <button
                onClick={() => {
                  if (!showBulk) setBulkText(members.join('\n'))
                  setShowBulk((v) => !v)
                }}
                className="text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-gray-400 rounded"
              >
                {showBulk ? t('members.bulk_hide') : t('members.bulk_show')}
              </button>
              {showBulk && (
                <div className="mt-2 space-y-2">
                  <textarea
                    value={bulkText}
                    onChange={(e) => setBulkText(e.target.value)}
                    rows={5}
                    placeholder={t('members.bulk_placeholder')}
                    aria-label={t('members.bulk_aria')}
                    className="w-full font-mono text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border)] rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] resize-none"
                  />
                  <Button
                    type="button"
                    loading={saveMutation.isPending}
                    onClick={handleBulkApply}
                  >
                    {t('members.bulk_apply')}
                  </Button>
                </div>
              )}
            </div>

            {saveError && <p className="mt-2 text-xs text-[var(--color-status-error)]">{saveError}</p>}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

function DeleteSegmentModal({
  open,
  segment,
  isDeleting,
  deleteFailed,
  onConfirm,
  onCancel,
}: {
  open: boolean
  segment: Segment | null
  isDeleting: boolean
  deleteFailed: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('segments')

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('delete.title')}</DialogTitle>
          {segment && (
            <DialogDescription>
              <Trans
                i18nKey="delete.body"
                ns="segments"
                values={{ slug: segment.slug }}
                components={{ mono: <span className="font-mono text-[var(--color-text-primary)]" /> }}
              />
            </DialogDescription>
          )}
        </DialogHeader>
        <p className="text-sm text-[var(--color-status-warning)] bg-[rgba(251,191,36,0.08)] border border-[var(--color-status-warning)] rounded px-3 py-2">
          {t('delete.warning')}
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-[var(--color-status-error)]">{t('delete.failed')}</p>
        )}
        <DialogFooter>
          <Button
            autoFocus
            type="button"
            variant="secondary"
            onClick={onCancel}
            disabled={isDeleting}
          >
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={isDeleting}
            onClick={onConfirm}
          >
            {t('actions.delete', { ns: 'common' })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SegmentListSkeleton() {
  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        <div className="h-8 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-[var(--color-border)] border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-40 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-2">
              <div className="h-4 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-6 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
              <div className="h-4 w-4 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
