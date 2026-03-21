import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { fetchJSON, patchJSON, deleteRequest } from '../../api'

interface Variant {
  key: string
  name: string
}

interface FlagItem {
  id: string
  key: string
  name: string
  type: string
  variants: Variant[]
  default_variant_key: string
  enabled: boolean
}

export const flagListRoute = createRoute({
  getParentRoute: () => projectEnvRoute,
  path: '/flags',
  component: FlagListPage,
})

function FlagListPage() {
  const { slug, envSlug } = flagListRoute.useParams()
  const queryClient = useQueryClient()
  const queryKey = ['flags', slug, envSlug]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      fetchJSON<{ flags: FlagItem[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags`,
      ).then((d) => d.flags),
  })

  const [toggleErrorKey, setToggleErrorKey] = useState<string | null>(null)

  const toggleMutation = useMutation({
    mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) =>
      patchJSON(`/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}`, { enabled }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
    onError: (_err, variables) => {
      setToggleErrorKey(variables.key)
      setTimeout(() => setToggleErrorKey(null), 3000)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (key: string) => deleteRequest(`/api/v1/projects/${slug}/flags/${key}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey }),
  })

  const [pendingDelete, setPendingDelete] = useState<string | null>(null)

  if (isLoading) return <FlagListSkeleton />
  if (isError) return <FlagListError onRetry={() => void refetch()} />

  const flags = data ?? []

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-semibold text-gray-900">Feature Flags</h1>
        {/* TODO: navigate to flag creation once route exists */}
        <button
          disabled
          className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          title="Flag creation coming soon"
        >
          New flag
        </button>
      </div>

      {flags.length === 0 ? (
        <EmptyState />
      ) : (
        <ul className="divide-y divide-gray-100 border border-gray-200 rounded-lg bg-white">
          {flags.map((flag) => (
            <FlagRow
              key={flag.id}
              flag={flag}
              slug={slug}
              envSlug={envSlug}
              onToggle={(enabled) => toggleMutation.mutate({ key: flag.key, enabled })}
              onDeleteIntent={() => setPendingDelete(flag.key)}
              isToggling={
                toggleMutation.isPending && toggleMutation.variables?.key === flag.key
              }
              isToggleError={toggleErrorKey === flag.key}
            />
          ))}
        </ul>
      )}

      {pendingDelete && (
        <DeleteConfirmModal
          flagKey={pendingDelete}
          isDeleting={deleteMutation.isPending}
          deleteFailed={deleteMutation.isError}
          onConfirm={() => {
            deleteMutation.mutate(pendingDelete, {
              onSuccess: () => setPendingDelete(null),
            })
          }}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </div>
  )
}

function FlagRow({
  flag,
  slug,
  envSlug,
  onToggle,
  onDeleteIntent,
  isToggling,
  isToggleError,
}: {
  flag: FlagItem
  slug: string
  envSlug: string
  onToggle: (enabled: boolean) => void
  onDeleteIntent: () => void
  isToggling: boolean
  isToggleError: boolean
}) {
  const [copied, setCopied] = useState(false)

  function copyKey() {
    void navigator.clipboard
      .writeText(flag.key)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {
        // clipboard write unavailable (non-HTTPS or permission denied)
      })
  }

  return (
    <li className="flex items-center justify-between px-4 py-3 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        {/* Flag key — click to copy */}
        <div className="relative">
          <button
            onClick={copyKey}
            className="font-mono text-sm text-gray-800 hover:text-blue-600 bg-gray-50 border border-gray-200 rounded px-2 py-0.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label={`Copy flag key ${flag.key}`}
          >
            {flag.key}
          </button>
          {copied && (
            <span className="absolute -top-7 left-1/2 -translate-x-1/2 text-xs bg-gray-800 text-white rounded px-2 py-0.5 whitespace-nowrap pointer-events-none">
              Copied!
            </span>
          )}
        </div>

        {/* Flag name — links to detail view */}
        <Link
          to="/projects/$slug/environments/$envSlug/flags/$key"
          params={{ slug, envSlug, key: flag.key }}
          className="text-sm text-gray-700 truncate hover:text-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
        >
          {flag.name}
        </Link>

        {/* Default variant badge */}
        <span className="font-mono text-xs text-gray-500 bg-gray-100 border border-gray-200 rounded px-1.5 py-0.5 shrink-0">
          {flag.default_variant_key}
        </span>
      </div>

      <div className="flex items-center gap-3 shrink-0">
        {/* Enabled/disabled pill toggle */}
        <button
          onClick={() => onToggle(!flag.enabled)}
          disabled={isToggling}
          aria-pressed={flag.enabled}
          aria-label={flag.enabled ? 'Disable flag' : 'Enable flag'}
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border transition-colors focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:opacity-60 ${
            isToggleError
              ? 'bg-red-50 text-red-700 border-red-200 focus:ring-red-500'
              : flag.enabled
                ? 'bg-green-50 text-green-700 border-green-200 hover:bg-green-100 focus:ring-green-500'
                : 'bg-gray-100 text-gray-600 border-gray-200 hover:bg-gray-200 focus:ring-gray-400'
          }`}
        >
          <span
            className={`w-1.5 h-1.5 rounded-full ${isToggleError ? 'bg-red-500' : flag.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
            aria-hidden="true"
          />
          {isToggleError ? 'Failed' : flag.enabled ? 'Enabled' : 'Disabled'}
        </button>

        {/* Delete */}
        <button
          onClick={onDeleteIntent}
          aria-label={`Delete flag ${flag.key}`}
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
      </div>
    </li>
  )
}

function EmptyState() {
  return (
    <div className="text-center py-16 px-6">
      <p className="text-sm text-gray-500">
        No flags yet. Create your first flag to start targeting users.
      </p>
      {/* TODO: navigate to flag creation once route exists */}
      <button
        disabled
        className="mt-4 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
        title="Flag creation coming soon"
      >
        New flag
      </button>
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
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-dialog-title"
    >
      <div className="absolute inset-0 bg-black/30" onClick={onCancel} aria-hidden="true" />
      <div className="relative bg-white rounded-lg shadow-lg max-w-sm w-full mx-4 p-6">
        <h2 id="delete-dialog-title" className="text-base font-semibold text-gray-900">
          Delete flag?
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          This will permanently delete{' '}
          <span className="font-mono text-gray-800">{flagKey}</span> from all environments.
          This action cannot be undone.
        </p>
        {deleteFailed && (
          <p className="mt-3 text-xs text-red-600">Failed to delete. Please try again.</p>
        )}
        <div className="mt-5 flex justify-end gap-3">
          <button
            autoFocus
            onClick={onCancel}
            disabled={isDeleting}
            className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isDeleting}
            className="px-3 py-1.5 text-sm font-medium bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-red-500"
          >
            {isDeleting ? 'Deleting…' : 'Delete'}
          </button>
        </div>
      </div>
    </div>
  )
}

function FlagListSkeleton() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="h-6 w-32 bg-gray-100 rounded animate-pulse" />
        <div className="h-8 w-24 bg-gray-100 rounded animate-pulse" />
      </div>
      <ul className="divide-y divide-gray-100 border border-gray-200 rounded-lg bg-white">
        {[1, 2, 3].map((i) => (
          <li key={i} className="flex items-center justify-between px-4 py-3 gap-4">
            <div className="flex items-center gap-3">
              <div className="h-6 w-28 bg-gray-100 rounded animate-pulse" />
              <div className="h-4 w-40 bg-gray-100 rounded animate-pulse" />
              <div className="h-5 w-16 bg-gray-100 rounded animate-pulse" />
            </div>
            <div className="flex items-center gap-3">
              <div className="h-6 w-20 bg-gray-100 rounded-full animate-pulse" />
              <div className="h-4 w-4 bg-gray-100 rounded animate-pulse" />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

function FlagListError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="p-6">
      <span className="text-sm text-red-600">Failed to load flags. </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        Retry
      </button>
    </div>
  )
}
