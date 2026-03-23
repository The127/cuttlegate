import { Outlet, createRoute, useRouterState } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { rootRoute } from './__root'
import { getUserManager } from '../auth'
import { ProjectSwitcher } from '../components/ProjectSwitcher'
import { CreateProjectDialogProvider } from '../components/CreateProjectDialog'
import { Breadcrumbs } from '../components/Breadcrumbs'
import { LiveAnnouncerProvider } from '../hooks/useLiveAnnouncer'
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
  const { t } = useTranslation('common')
  if (error instanceof APIError && error.status === 403) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="max-w-md w-full p-8 bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('errors.access_denied_title')}</h1>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            {t('errors.access_denied_body')}
          </p>
          <a href="/" className="mt-4 inline-block text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300">
            {t('errors.return_to_home')}
          </a>
        </div>
      </div>
    )
  }
  const message = error instanceof Error ? error.message : t('errors.unexpected')
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="max-w-md w-full p-8 bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('errors.something_went_wrong')}</h1>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400 font-mono">{message}</p>
      </div>
    </div>
  )
}

function AppShell() {
  const mainRef = useRef<HTMLElement>(null)
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const prevPathname = useRef<string | null>(null)

  useEffect(() => {
    if (prevPathname.current !== null && prevPathname.current !== pathname) {
      mainRef.current?.focus()
    }
    prevPathname.current = pathname
  }, [pathname])

  return (
    <LiveAnnouncerProvider>
      <CreateProjectDialogProvider>
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
          <ProjectSwitcher />
          <Breadcrumbs />
          <main id="main-content" ref={mainRef} tabIndex={-1} className="outline-none">
            <Outlet />
          </main>
        </div>
      </CreateProjectDialogProvider>
    </LiveAnnouncerProvider>
  )
}
