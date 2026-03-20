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
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full p-8 bg-white rounded-lg shadow-sm border border-gray-200">
          <h1 className="text-lg font-semibold text-gray-900">Authentication Error</h1>
          <p className="mt-2 text-sm text-gray-600 font-mono">{error}</p>
          <a
            href="/"
            className="mt-4 inline-block text-sm text-blue-600 hover:text-blue-800"
          >
            Return to home
          </a>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <p className="text-sm text-gray-500">Completing sign-in&hellip;</p>
    </div>
  )
}
