import { createRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, patchEmpty, postJSON, deleteRequest, APIError } from '../../api'
import { getUserManager } from '../../auth'
import { formatRelativeDate } from '../../utils/date'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Select, SelectItem } from '../../components/ui/Select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '../../components/ui/Dialog'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'

type Role = 'admin' | 'editor' | 'viewer'
const ROLES: Role[] = ['admin', 'editor', 'viewer']

const ROLE_STYLES: Record<Role, string> = {
  admin: 'bg-[rgba(79,124,255,0.1)] text-[var(--color-accent)] border-[rgba(79,124,255,0.3)]',
  editor: 'bg-[rgba(16,217,168,0.08)] text-[var(--color-status-enabled)] border-[var(--color-status-enabled)]',
  viewer: 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)]',
}

interface Member {
  project_id: string
  user_id: string
  role: Role
  name: string
  email: string
  created_at: string
}

export const memberListRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/members',
  component: MemberListPage,
})

function MemberListPage() {
  const { t } = useTranslation('projects')
  const { slug } = memberListRoute.useParams()
  const project = projectRoute.useLoaderData()
  useDocumentTitle(t('members.title'), project.name)
  const queryClient = useQueryClient()
  const queryKey = ['members', slug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ members: Member[] }>(`/api/v1/projects/${slug}/members`).then(
        (d) => d.members,
      ),
  })

  const [currentUserId, setCurrentUserId] = useState<string | null>(null)
  useEffect(() => {
    void getUserManager()
      .getUser()
      .then((u) => {
        if (u?.profile.sub) setCurrentUserId(u.profile.sub)
      })
  }, [])

  const members = data ?? []
  const myRole = members.find((m) => m.user_id === currentUserId)?.role ?? 'viewer'
  const isAdmin = myRole === 'admin'

  const addMemberSectionRef = useRef<HTMLDivElement>(null)

  const [pendingRemove, setPendingRemove] = useState<Member | null>(null)
  const [removeError, setRemoveError] = useState<string | null>(null)
  const [roleErrors, setRoleErrors] = useState<Record<string, string>>({})

  const removeMutation = useMutation({
    mutationFn: (userId: string) =>
      deleteRequest(`/api/v1/projects/${slug}/members/${userId}`),
    onSuccess: () => {
      setPendingRemove(null)
      setRemoveError(null)
      void queryClient.invalidateQueries({ queryKey })
    },
    onError: (err) => {
      const msg =
        err instanceof APIError && err.code === 'last_admin'
          ? t('members.last_admin')
          : err instanceof APIError
            ? err.message
            : t('members.remove_failed')
      setRemoveError(msg)
    },
  })

  const roleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: Role }) =>
      patchEmpty(`/api/v1/projects/${slug}/members/${userId}`, { role }),
    onSuccess: (_data, { userId }) => {
      setRoleErrors((prev) => { const next = { ...prev }; delete next[userId]; return next })
      void queryClient.invalidateQueries({ queryKey })
    },
    onError: (err, { userId }) => {
      const msg =
        err instanceof APIError ? err.message : t('members.role_update_failed')
      setRoleErrors((prev) => ({ ...prev, [userId]: msg }))
    },
  })

  if (isLoading) return <MemberListSkeleton />
  if (isError)
    return (
      <div className="p-6">
        <span className="text-sm text-[var(--color-status-error)]">{t('members.error')} </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-[var(--color-status-error)] underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-[var(--color-status-error)] rounded"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )

  return (
    <div className="p-6 max-w-4xl">
      <h1 className="text-xl font-semibold text-[var(--color-text-primary)] mb-6">{t('members.title')}</h1>

      {members.length === 0 ? (
        <MemberEmptyState
          isAdmin={isAdmin}
          onAddClick={() => {
            addMemberSectionRef.current?.scrollIntoView({ behavior: 'smooth' })
            document.getElementById('member-user-id')?.focus()
          }}
        />
      ) : (
        <div className="border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-[var(--color-surface-elevated)] border-b border-[var(--color-border)]">
              <tr>
                <th className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium w-full">
                  {t('members.column_member')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                  {t('members.column_role')}
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap">
                  {t('members.column_joined')}
                </th>
                {isAdmin && <th className="px-4 py-2" />}
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--color-border)]">
              {members.map((m) => (
                <MemberRow
                  key={m.user_id}
                  member={m}
                  isAdmin={isAdmin}
                  isSelf={m.user_id === currentUserId}
                  roleError={roleErrors[m.user_id] ?? ''}
                  rolePending={
                    roleMutation.isPending && roleMutation.variables?.userId === m.user_id
                  }
                  onRoleChange={(role) => roleMutation.mutate({ userId: m.user_id, role })}
                  onRemoveIntent={() => {
                    setPendingRemove(m)
                    setRemoveError(null)
                  }}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {isAdmin && (
        <div className="mt-6" ref={addMemberSectionRef}>
          <AddMemberForm slug={slug} />
        </div>
      )}

      <RemoveMemberDialog
        open={pendingRemove !== null}
        member={pendingRemove}
        error={removeError}
        isRemoving={removeMutation.isPending}
        onConfirm={() => pendingRemove && removeMutation.mutate(pendingRemove.user_id)}
        onCancel={() => {
          setPendingRemove(null)
          setRemoveError(null)
        }}
      />
    </div>
  )
}

function memberDisplayName(member: Member): string {
  return member.name || member.user_id
}

function MemberRow({
  member,
  isAdmin,
  isSelf,
  roleError,
  rolePending,
  onRoleChange,
  onRemoveIntent,
}: {
  member: Member
  isAdmin: boolean
  isSelf: boolean
  roleError: string
  rolePending: boolean
  onRoleChange: (role: Role) => void
  onRemoveIntent: () => void
}) {
  const { t } = useTranslation('projects')
  const displayName = memberDisplayName(member)
  return (
    <tr className="hover:bg-[var(--color-surface)]">
      <td className="px-4 py-3">
        <div className="flex flex-col gap-0.5">
          <span className="text-sm text-[var(--color-text-primary)] font-medium">
            {displayName}
            {isSelf && (
              <span className="ml-2 text-xs text-[var(--color-text-muted)] font-normal">{t('members.you')}</span>
            )}
          </span>
          {member.email && (
            <span className="text-xs text-[var(--color-text-muted)]">{member.email}</span>
          )}
          {!member.name && (
            <span className="font-mono text-xs text-[var(--color-text-secondary)]">{member.user_id}</span>
          )}
        </div>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {isAdmin ? (
          <div>
            <Select
              value={member.role}
              onValueChange={(v) => onRoleChange(v as Role)}
              disabled={rolePending}
              aria-label={t('members.role_aria', { name: displayName })}
            >
              {ROLES.map((r) => (
                <SelectItem key={r} value={r}>
                  {t(ROLE_I18N_KEYS[r])}
                </SelectItem>
              ))}
            </Select>
            {roleError && (
              <p className="mt-1 text-xs text-[var(--color-status-error)]">{roleError}</p>
            )}
          </div>
        ) : (
          <RoleBadge role={member.role} />
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <time
          dateTime={member.created_at}
          className="text-xs text-[var(--color-text-muted)]"
          title={new Date(member.created_at).toLocaleString()}
        >
          {formatRelativeDate(member.created_at)}
        </time>
      </td>
      {isAdmin && (
        <td className="px-4 py-3 whitespace-nowrap">
          <button
            onClick={onRemoveIntent}
            aria-label={t('members.remove_aria', { name: displayName })}
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
        </td>
      )}
    </tr>
  )
}

const ROLE_I18N_KEYS: Record<Role, string> = {
  admin: 'members.role_admin',
  editor: 'members.role_editor',
  viewer: 'members.role_viewer',
}

function RoleBadge({ role }: { role: Role }) {
  const { t } = useTranslation('projects')
  return (
    <span
      className={`inline-block text-xs font-medium border rounded px-2 py-0.5 ${ROLE_STYLES[role]}`}
    >
      {t(ROLE_I18N_KEYS[role])}
    </span>
  )
}

function AddMemberForm({ slug }: { slug: string }) {
  const { t } = useTranslation('projects')
  const queryClient = useQueryClient()
  const queryKey = ['members', slug]
  const [userId, setUserId] = useState('')
  const [role, setRole] = useState<Role>('viewer')
  const [error, setError] = useState<string | null>(null)

  const addMutation = useMutation({
    mutationFn: () =>
      postJSON<Member>(`/api/v1/projects/${slug}/members`, { user_id: userId.trim(), role }),
    onSuccess: () => {
      setUserId('')
      setRole('viewer')
      setError(null)
      void queryClient.invalidateQueries({ queryKey })
    },
    onError: (err) => {
      if (err instanceof APIError && err.status === 409) {
        setError(t('members.already_member'))
      } else {
        setError(err instanceof APIError ? err.message : t('members.add_failed'))
      }
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!userId.trim()) return
    setError(null)
    addMutation.mutate()
  }

  return (
    <div className="border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] p-4">
      <h2 className="text-sm font-medium text-[var(--color-text-primary)] mb-3">{t('members.add_title')}</h2>
      <form onSubmit={handleSubmit} className="flex items-start gap-3 flex-wrap">
        <div className="flex-1 min-w-48">
          <label htmlFor="member-user-id" className="sr-only">
            {t('members.user_id_label')}
          </label>
          <Input
            id="member-user-id"
            type="text"
            value={userId}
            onChange={(e) => {
              setUserId(e.target.value)
              setError(null)
            }}
            placeholder={t('members.user_id_placeholder')}
            hasError={!!error}
            aria-describedby={error ? 'add-member-error' : 'member-user-id-helper'}
            className="font-mono"
          />
          <p id="member-user-id-helper" className="mt-1 text-xs text-[var(--color-text-muted)]">
            {t('members.user_id_helper')}
          </p>
        </div>
        <div>
          <Select
            value={role}
            onValueChange={(v) => setRole(v as Role)}
            aria-label={t('members.role_label')}
          >
            {ROLES.map((r) => (
              <SelectItem key={r} value={r}>
                {t(ROLE_I18N_KEYS[r])}
              </SelectItem>
            ))}
          </Select>
        </div>
        <Button
          type="submit"
          loading={addMutation.isPending}
          disabled={!userId.trim()}
          className="whitespace-nowrap"
        >
          {t('members.add_button')}
        </Button>
      </form>
      {error && (
        <p id="add-member-error" className="mt-2 text-xs text-[var(--color-status-error)]">
          {error}
        </p>
      )}
    </div>
  )
}

function RemoveMemberDialog({
  open,
  member,
  error,
  isRemoving,
  onConfirm,
  onCancel,
}: {
  open: boolean
  member: Member | null
  error: string | null
  isRemoving: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  const displayName = member ? memberDisplayName(member) : ''

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('members.remove_dialog_title')}</DialogTitle>
          {member && (
            <DialogDescription>
              {t('members.remove_dialog_body', { name: displayName })}
              {member.email && (
                <span className="text-[var(--color-text-secondary)]"> ({member.email})</span>
              )}
            </DialogDescription>
          )}
        </DialogHeader>
        {error && (
          <p className="text-xs text-[var(--color-status-error)]">{error}</p>
        )}
        <DialogFooter>
          <Button
            autoFocus
            type="button"
            variant="secondary"
            onClick={onCancel}
            disabled={isRemoving}
          >
            {t('actions.cancel', { ns: 'common' })}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={isRemoving}
            onClick={onConfirm}
          >
            {t('members.remove')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function MemberEmptyState({
  isAdmin,
  onAddClick,
}: {
  isAdmin: boolean
  onAddClick: () => void
}) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-16 px-6 border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)]">
      {isAdmin ? (
        <>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {t('members.empty')}
          </p>
          <Button size="lg" className="mt-4" onClick={onAddClick}>
            {t('members.add_button')}
          </Button>
          <RoleExplanation />
        </>
      ) : (
        <>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {t('members.empty_nonadmin')}
          </p>
          <p className="text-sm text-[var(--color-text-muted)] mt-2">
            {t('members.empty_nonadmin_hint')}
          </p>
        </>
      )}
    </div>
  )
}

function RoleExplanation() {
  const { t } = useTranslation('projects')
  const ROLE_DESCRIPTIONS: Record<Role, string> = {
    admin: 'members.role_admin_description',
    editor: 'members.role_editor_description',
    viewer: 'members.role_viewer_description',
  }
  return (
    <div className="mt-6 text-left max-w-sm mx-auto">
      <p className="text-xs font-medium text-[var(--color-text-secondary)] mb-2">
        {t('members.roles_explanation_title')}
      </p>
      <ul className="space-y-1.5">
        {ROLES.map((r) => (
          <li key={r} className="flex items-start gap-2">
            <RoleBadge role={r} />
            <span className="text-xs text-[var(--color-text-muted)]">{t(ROLE_DESCRIPTIONS[r])}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function MemberListSkeleton() {
  return (
    <div className="p-6 max-w-4xl">
      <div className="h-6 w-24 bg-[var(--color-surface-elevated)] rounded animate-pulse mb-6" />
      <div className="border border-[var(--color-border)] rounded-lg bg-[var(--color-surface)] overflow-hidden">
        <div className="bg-[var(--color-surface-elevated)] border-b border-[var(--color-border)] px-4 py-2 flex gap-8">
          <div className="h-3 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          <div className="h-3 w-8 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          <div className="h-3 w-10 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
        </div>
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center gap-8 px-4 py-3 border-b border-[var(--color-border)] last:border-0">
            <div className="h-5 w-72 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-5 w-14 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
            <div className="h-4 w-12 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
        ))}
      </div>
    </div>
  )
}

