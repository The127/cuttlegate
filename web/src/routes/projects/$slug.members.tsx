import { createRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect, useRef } from 'react'
import { projectRoute } from './$slug'
import { fetchJSON, patchEmpty, postJSON, deleteRequest, APIError } from '../../api'
import { getUserManager } from '../../auth'
import { formatRelativeDate } from '../../utils/date'

type Role = 'admin' | 'editor' | 'viewer'
const ROLES: Role[] = ['admin', 'editor', 'viewer']

const ROLE_STYLES: Record<Role, string> = {
  admin: 'bg-blue-50 text-blue-700 border-blue-200',
  editor: 'bg-green-50 text-green-700 border-green-200',
  viewer: 'bg-gray-50 text-gray-600 border-gray-200',
}

interface Member {
  project_id: string
  user_id: string
  role: Role
  created_at: string
}

export const memberListRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/members',
  component: MemberListPage,
})

function MemberListPage() {
  const { slug } = memberListRoute.useParams()
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
          ? 'Cannot remove the last admin of this project.'
          : err instanceof APIError
            ? err.message
            : 'Failed to remove member. Please try again.'
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
        err instanceof APIError ? err.message : 'Failed to update role. Please try again.'
      setRoleErrors((prev) => ({ ...prev, [userId]: msg }))
    },
  })

  if (isLoading) return <MemberListSkeleton />
  if (isError)
    return (
      <div className="p-6">
        <span className="text-sm text-red-600">Failed to load members. </span>
        <button
          onClick={() => void refetch()}
          className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
        >
          Retry
        </button>
      </div>
    )

  return (
    <div className="p-6 max-w-4xl">
      <h1 className="text-lg font-semibold text-gray-900 mb-6">Members</h1>

      {members.length === 0 ? (
        <MemberEmptyState />
      ) : (
        <div className="border border-gray-200 rounded-lg bg-white overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide w-full">
                  User ID
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  Role
                </th>
                <th className="text-left px-4 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide whitespace-nowrap">
                  Joined
                </th>
                {isAdmin && <th className="px-4 py-2" />}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
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
        <div className="mt-6">
          <AddMemberForm slug={slug} />
        </div>
      )}

      {pendingRemove && (
        <RemoveMemberDialog
          member={pendingRemove}
          error={removeError}
          isRemoving={removeMutation.isPending}
          onConfirm={() => removeMutation.mutate(pendingRemove.user_id)}
          onCancel={() => {
            setPendingRemove(null)
            setRemoveError(null)
          }}
        />
      )}
    </div>
  )
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
  return (
    <tr className="hover:bg-gray-50">
      <td className="px-4 py-3">
        <span className="font-mono text-xs text-gray-700 bg-gray-50 border border-gray-200 rounded px-1.5 py-0.5 select-all">
          {member.user_id}
        </span>
        {isSelf && (
          <span className="ml-2 text-xs text-gray-400 font-normal">(you)</span>
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {isAdmin ? (
          <div>
            <select
              value={member.role}
              disabled={rolePending}
              onChange={(e) => onRoleChange(e.target.value as Role)}
              aria-label={`Role for ${member.user_id}`}
              className="text-xs border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 bg-white"
            >
              {ROLES.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
            {roleError && (
              <p className="mt-1 text-xs text-red-600">{roleError}</p>
            )}
          </div>
        ) : (
          <RoleBadge role={member.role} />
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <time
          dateTime={member.created_at}
          className="text-xs text-gray-400"
          title={new Date(member.created_at).toLocaleString()}
        >
          {formatRelativeDate(member.created_at)}
        </time>
      </td>
      {isAdmin && (
        <td className="px-4 py-3 whitespace-nowrap">
          <button
            onClick={onRemoveIntent}
            aria-label={`Remove member ${member.user_id}`}
            className="text-gray-400 hover:text-red-600 transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 rounded p-0.5"
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

function RoleBadge({ role }: { role: Role }) {
  return (
    <span
      className={`inline-block text-xs font-medium border rounded px-2 py-0.5 ${ROLE_STYLES[role]}`}
    >
      {role}
    </span>
  )
}

function AddMemberForm({ slug }: { slug: string }) {
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
        setError('This user is already a member of this project.')
      } else {
        setError(err instanceof APIError ? err.message : 'Failed to add member. Please try again.')
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
    <div className="border border-gray-200 rounded-lg bg-white p-4">
      <h2 className="text-sm font-medium text-gray-700 mb-3">Add member</h2>
      <form onSubmit={handleSubmit} className="flex items-start gap-3 flex-wrap">
        <div className="flex-1 min-w-48">
          <label htmlFor="member-user-id" className="sr-only">
            User ID
          </label>
          <input
            id="member-user-id"
            type="text"
            value={userId}
            onChange={(e) => {
              setUserId(e.target.value)
              setError(null)
            }}
            placeholder="User ID (UUID)"
            aria-invalid={!!error}
            aria-describedby={error ? 'add-member-error' : undefined}
            className={`w-full font-mono text-sm border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
              error
                ? 'border-red-300 focus:ring-red-500'
                : 'border-gray-300 focus:ring-blue-500'
            }`}
          />
        </div>
        <div>
          <label htmlFor="member-role" className="sr-only">
            Role
          </label>
          <select
            id="member-role"
            value={role}
            onChange={(e) => setRole(e.target.value as Role)}
            className="text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          >
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </div>
        <button
          type="submit"
          disabled={addMutation.isPending || !userId.trim()}
          className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500 whitespace-nowrap"
        >
          {addMutation.isPending ? 'Adding\u2026' : 'Add member'}
        </button>
      </form>
      {error && (
        <p id="add-member-error" className="mt-2 text-xs text-red-600">
          {error}
        </p>
      )}
    </div>
  )
}

function RemoveMemberDialog({
  member,
  error,
  isRemoving,
  onConfirm,
  onCancel,
}: {
  member: Member
  error: string | null
  isRemoving: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  useEscapeKey(onCancel)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="remove-member-title"
    >
      <div className="absolute inset-0 bg-black/30" onClick={onCancel} aria-hidden="true" />
      <div className="relative bg-white rounded-lg shadow-lg max-w-sm w-full mx-4 p-6">
        <h2 id="remove-member-title" className="text-base font-semibold text-gray-900">
          Remove member?
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          This will remove{' '}
          <span className="font-mono text-xs text-gray-800 bg-gray-50 border border-gray-200 rounded px-1.5 py-0.5">
            {member.user_id}
          </span>{' '}
          from the project.
        </p>
        {error && (
          <p className="mt-3 text-xs text-red-600">{error}</p>
        )}
        <div className="mt-5 flex justify-end gap-3">
          <button
            autoFocus
            onClick={onCancel}
            disabled={isRemoving}
            className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isRemoving}
            className="px-3 py-1.5 text-sm font-medium bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-red-500"
          >
            {isRemoving ? 'Removing\u2026' : 'Remove'}
          </button>
        </div>
      </div>
    </div>
  )
}

function MemberEmptyState() {
  return (
    <div className="text-center py-16 px-6 border border-gray-200 rounded-lg bg-white">
      <p className="text-sm text-gray-500">
        Add team members to collaborate on this project.
      </p>
    </div>
  )
}

function MemberListSkeleton() {
  return (
    <div className="p-6 max-w-4xl">
      <div className="h-6 w-24 bg-gray-100 rounded animate-pulse mb-6" />
      <div className="border border-gray-200 rounded-lg bg-white overflow-hidden">
        <div className="bg-gray-50 border-b border-gray-200 px-4 py-2 flex gap-8">
          <div className="h-3 w-12 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-8 bg-gray-200 rounded animate-pulse" />
          <div className="h-3 w-10 bg-gray-200 rounded animate-pulse" />
        </div>
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center gap-8 px-4 py-3 border-b border-gray-100 last:border-0">
            <div className="h-5 w-72 bg-gray-100 rounded animate-pulse" />
            <div className="h-5 w-14 bg-gray-100 rounded animate-pulse" />
            <div className="h-4 w-12 bg-gray-100 rounded animate-pulse" />
          </div>
        ))}
      </div>
    </div>
  )
}

function useEscapeKey(handler: () => void) {
  const handlerRef = useRef(handler)
  useEffect(() => { handlerRef.current = handler }, [handler])
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') handlerRef.current()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])
}
