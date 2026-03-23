import { createRoute, useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { projectRoute } from './$slug'
import { patchJSON, deleteRequest, APIError } from '../../api'
import { useProjectRole } from '../../hooks/useProjectRole'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../../components/ui/Dialog'

export const projectSettingsRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/settings',
  component: ProjectSettingsPage,
})

function ProjectSettingsPage() {
  const { t } = useTranslation('projects')
  const project = projectRoute.useLoaderData()
  const roleQuery = useProjectRole(project.slug)

  if (roleQuery.isLoading) return <SettingsSkeleton />
  if (roleQuery.isError)
    return (
      <div className="p-6">
        <span className="text-sm text-[var(--color-status-error)]">{t('settings.failed_to_load')} </span>
        <button
          onClick={() => void roleQuery.refetch()}
          className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  const isAdmin = roleQuery.data === 'admin'

  return (
    <div className="p-6 max-w-2xl">
      <h1 className="text-xl font-semibold text-[var(--color-text-primary)] mb-6">{t('settings.title')}</h1>

      <GeneralSection project={project} isAdmin={isAdmin} />

      {isAdmin && (
        <>
          <div className="mt-8 border-t border-[var(--color-border)]" />
          <DangerZone project={project} />
        </>
      )}
    </div>
  )
}

interface ProjectData {
  id: string
  name: string
  slug: string
  created_at: string
}

function GeneralSection({
  project,
  isAdmin,
}: {
  project: ProjectData
  isAdmin: boolean
}) {
  const { t } = useTranslation('projects')
  return (
    <section>
      <h2 className="text-sm font-medium text-[var(--color-text-secondary)] font-medium mb-4">{t('settings.general_section')}</h2>
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg divide-y divide-[var(--color-border)]">
        <NameField project={project} isAdmin={isAdmin} />
        <SlugField slug={project.slug} />
      </div>
    </section>
  )
}

function NameField({ project, isAdmin }: { project: ProjectData; isAdmin: boolean }) {
  const { t } = useTranslation('projects')
  const queryClient = useQueryClient()
  const [name, setName] = useState(project.name)
  const [saved, setSaved] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  const updateMutation = useMutation({
    mutationFn: () => patchJSON<ProjectData>(`/api/v1/projects/${project.slug}`, { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] })
      setSaved(true)
      setServerError(null)
      setTimeout(() => setSaved(false), 2000)
    },
    onError: (err) => {
      setServerError(err instanceof APIError ? err.message : t('settings.save_failed'))
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || name === project.name) return
    setServerError(null)
    updateMutation.mutate()
  }

  return (
    <div className="px-4 py-4">
      <label htmlFor="project-name" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
        {t('settings.name_label')}
      </label>
      {isAdmin ? (
        <form onSubmit={handleSubmit} className="flex items-center gap-3">
          <Input
            id="project-name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              setServerError(null)
              setSaved(false)
            }}
            className="flex-1"
          />
          <Button
            type="submit"
            loading={updateMutation.isPending}
            disabled={!name.trim() || name === project.name}
          >
            {saved ? t('settings.saved') : t('settings.save')}
          </Button>
        </form>
      ) : (
        <p className="text-sm text-[var(--color-text-primary)]">{project.name}</p>
      )}
      {serverError && <p className="mt-1 text-xs text-[var(--color-status-error)]">{serverError}</p>}
    </div>
  )
}

function SlugField({ slug }: { slug: string }) {
  const { t } = useTranslation('projects')
  const [copied, setCopied] = useState(false)

  function copySlug() {
    void navigator.clipboard
      .writeText(slug)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {})
  }

  return (
    <div className="px-4 py-4">
      <p className="text-xs font-medium text-[var(--color-text-secondary)] mb-1">{t('settings.slug_label')}</p>
      <div className="flex items-center gap-2">
        <span className="font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-1">
          {slug}
        </span>
        <div className="relative">
          <button
            onClick={copySlug}
            className="text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded px-2 py-1 border border-[var(--color-border)] hover:border-[var(--color-accent)] transition-colors"
            aria-label={t('settings.copy_slug_aria', { slug })}
          >
            {copied ? t('settings.copied') : t('settings.copy')}
          </button>
        </div>
        <p className="text-xs text-[var(--color-text-muted)]">{t('settings.slug_immutable')}</p>
      </div>
    </div>
  )
}

function DangerZone({ project }: { project: ProjectData }) {
  const { t } = useTranslation('projects')
  const [showDeleteModal, setShowDeleteModal] = useState(false)

  return (
    <section className="mt-8">
      <h2 className="text-sm font-medium text-[var(--color-status-error)] font-medium mb-4">{t('settings.danger_section')}</h2>
      <div className="bg-[var(--color-surface)] border border-[var(--color-status-error)] rounded-lg px-4 py-4 flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-[var(--color-text-primary)]">{t('settings.delete_title')}</p>
          <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
            {t('settings.delete_description')}
          </p>
        </div>
        <Button
          variant="destructive"
          onClick={() => setShowDeleteModal(true)}
          className="shrink-0"
        >
          {t('settings.delete_button')}
        </Button>
      </div>

      <DeleteProjectModal
        open={showDeleteModal}
        project={project}
        onCancel={() => setShowDeleteModal(false)}
      />
    </section>
  )
}

function DeleteProjectModal({
  open,
  project,
  onCancel,
}: {
  open: boolean
  project: ProjectData
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirmName, setConfirmName] = useState('')
  const [serverError, setServerError] = useState<string | null>(null)

  const deleteMutation = useMutation({
    mutationFn: () => deleteRequest(`/api/v1/projects/${project.slug}`),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      void navigate({ to: '/' })
    },
    onError: (err) => {
      setServerError(err instanceof APIError ? err.message : t('delete_project.failed'))
    },
  })

  function handleDelete(e: React.FormEvent) {
    e.preventDefault()
    if (confirmName !== project.name) return
    setServerError(null)
    deleteMutation.mutate()
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('delete_project.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-[var(--color-text-secondary)]">
          <Trans
            i18nKey="delete_project.body"
            ns="projects"
            values={{ name: project.name }}
            components={{ strong: <span className="font-semibold text-[var(--color-text-primary)]" /> }}
          />
        </p>
        <ul className="mt-2 text-sm text-[var(--color-text-secondary)] list-disc list-inside space-y-0.5">
          <li>{t('delete_project.item_flags')}</li>
          <li>{t('delete_project.item_environments')}</li>
          <li>{t('delete_project.item_members')}</li>
        </ul>

        <div className="mt-4 p-3 bg-[rgba(248,113,113,0.08)] border border-[var(--color-status-error)] rounded text-xs text-[var(--color-status-error)]">
          {t('delete_project.warning')}
        </div>

        <form onSubmit={handleDelete} className="mt-4 space-y-3">
          <div>
            <label htmlFor="confirm-name" className="block text-xs font-medium text-[var(--color-text-primary)] mb-1">
              <Trans
                i18nKey="delete_project.confirm_label"
                ns="projects"
                values={{ name: project.name }}
                components={{ mono: <span className="font-mono font-semibold" /> }}
              />
            </label>
            <Input
              id="confirm-name"
              type="text"
              value={confirmName}
              onChange={(e) => {
                setConfirmName(e.target.value)
                setServerError(null)
              }}
              autoFocus
              placeholder={project.name}
              hasError={false}
              className="focus:border-[var(--color-status-error)] focus:shadow-[0_0_0_2px_rgba(248,113,113,0.4)]"
            />
          </div>
          {serverError && <p className="text-xs text-[var(--color-status-error)]">{serverError}</p>}
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={deleteMutation.isPending}
            >
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              loading={deleteMutation.isPending}
              disabled={confirmName !== project.name}
            >
              {t('delete_project.delete_button')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SettingsSkeleton() {
  return (
    <div className="p-6 max-w-2xl">
      <div className="h-8 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-6" />
      <div className="h-4 w-16 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-4" />
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg divide-y divide-[var(--color-border)]">
        <div className="px-4 py-4">
          <div className="h-3 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-2" />
          <div className="h-8 w-48 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </div>
        <div className="px-4 py-4">
          <div className="h-3 w-20 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-2" />
          <div className="h-7 w-32 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </div>
      </div>
    </div>
  )
}
