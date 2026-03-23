import { useState, useEffect, useRef } from 'react'
import { createRoute, useNavigate } from '@tanstack/react-router'
import { rootRoute } from '../__root'
import { getUserManager } from '../../auth'

export const callbackRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/callback',
  component: CallbackPage,
})

function CallbackPage() {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)
  const handled = useRef(false)

  useEffect(() => {
    if (handled.current) return
    handled.current = true

    getUserManager()
      .signinRedirectCallback()
      .then((user) => {
        const target = typeof user.state === 'string' && user.state ? user.state : '/'
        navigate({ to: target, replace: true })
      })
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Authentication failed'
        setError(message)
      })
  }, [navigate])

  if (error !== null) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="max-w-md w-full p-8 bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Authentication Error</h1>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400 font-mono">{error}</p>
          <a
            href="/"
            className="mt-4 inline-block text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
          >
            Return to home
          </a>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <p className="text-sm text-gray-500 dark:text-gray-400">Completing sign-in&hellip;</p>
    </div>
  )
}
