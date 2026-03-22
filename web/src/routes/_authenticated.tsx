import { Outlet, createRoute } from '@tanstack/react-router'
import { rootRoute } from './__root'
import { getUserManager } from '../auth'
import { ProjectSwitcher } from '../components/ProjectSwitcher'
import { UserMenu } from '../components/UserMenu'
import { APIError } from '../api'

export const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: '_authenticated',
  beforeLoad: async ({ location }) => {
    const user = await getUserManager().getUser()
    if (!user || user.expired) {
      await getUserManager().signinRedirect({
        state: location.pathname + location.searchStr,
      })
      // Hang until the browser navigation completes — prevents rendering.
      await new Promise<never>(() => {})
    }
  },
  errorComponent: RouteError,
  component: AppShell,
})

function RouteError({ error }: { error: unknown }) {
  if (error instanceof APIError && error.status === 403) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full p-8 bg-white rounded-lg shadow-sm border border-gray-200">
          <h1 className="text-lg font-semibold text-gray-900">Access Denied</h1>
          <p className="mt-2 text-sm text-gray-600">
            You do not have permission to access this resource.
          </p>
          <a href="/" className="mt-4 inline-block text-sm text-blue-600 hover:text-blue-800">
            Return to home
          </a>
        </div>
      </div>
    )
  }
  const message = error instanceof Error ? error.message : 'An unexpected error occurred.'
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full p-8 bg-white rounded-lg shadow-sm border border-gray-200">
        <h1 className="text-lg font-semibold text-gray-900">Something went wrong</h1>
        <p className="mt-2 text-sm text-gray-600 font-mono">{message}</p>
      </div>
    </div>
  )
}

function AppShell() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="flex items-center justify-between border-b border-gray-200 bg-white">
        <ProjectSwitcher />
        <div className="px-4">
          <UserMenu />
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
