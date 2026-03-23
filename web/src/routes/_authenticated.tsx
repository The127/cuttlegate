import { Outlet, createRoute, useLocation, useRouterState } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { rootRoute } from './__root'
import { getUserManager } from '../auth'
import { ProjectSwitcher } from '../components/ProjectSwitcher'
import { Sidebar } from '../components/Sidebar'
import { CreateProjectDialogProvider } from '../components/CreateProjectDialog'
import { LiveAnnouncerProvider } from '../hooks/useLiveAnnouncer'
import { APIError } from '../api'
import { useDocumentTitle } from '../hooks/useDocumentTitle'

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
  notFoundComponent: InShellNotFoundPage,
  component: AppShell,
})

function RouteError({ error }: { error: unknown }) {
  const { t } = useTranslation('common')
  if (error instanceof APIError && error.status === 403) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
        <div className="max-w-md w-full p-8 bg-[var(--color-surface)] rounded-lg shadow-sm border border-[var(--color-border)]">
          <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('errors.access_denied_title')}</h1>
          <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
            {t('errors.access_denied_body')}
          </p>
          <a href="/" className="mt-4 inline-block text-sm text-[var(--color-accent)] hover:text-[var(--color-accent)]">
            {t('errors.return_to_home')}
          </a>
        </div>
      </div>
    )
  }
  const message = error instanceof Error ? error.message : t('errors.unexpected')
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
      <div className="max-w-md w-full p-8 bg-[var(--color-surface)] rounded-lg shadow-sm border border-[var(--color-border)]">
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">{t('errors.something_went_wrong')}</h1>
        <p className="mt-2 text-sm text-[var(--color-text-secondary)] font-mono">{message}</p>
      </div>
    </div>
  )
}

function InShellNotFoundPage() {
  const { t } = useTranslation('common')
  const location = useLocation()
  const url = location.pathname
  useDocumentTitle(`${t('not_found.title')} \u2014 ${t('not_found.page_title')}`)

  return (
    <div className="flex flex-1 items-center justify-center p-8">
      <div className="text-center">
        <h1 className="text-2xl font-semibold text-[var(--color-text-primary)]">{t('not_found.title')}</h1>
        <p className="mt-2 text-[var(--color-text-secondary)]">{t('not_found.body', { url })}</p>
        <a href="/" className="mt-4 inline-block text-sm text-[var(--color-accent)] hover:underline">
          {t('not_found.return_to_home')}
        </a>
      </div>
    </div>
  )
}

function useActiveProjectParams() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const projectMatch = /^\/projects\/([^/]+)/.exec(pathname)
  const envMatch = /^\/projects\/[^/]+\/environments\/([^/]+)/.exec(pathname)
  return {
    pathname,
    projectSlug: projectMatch?.[1] ?? null,
    envSlug: envMatch?.[1] ?? null,
  }
}

function AppShell() {
  const mainRef = useRef<HTMLElement>(null)
  const prevPathname = useRef<string | null>(null)
  const { pathname, projectSlug, envSlug } = useActiveProjectParams()
  const isProjectRoute = pathname.startsWith('/projects/')

  useEffect(() => {
    if (prevPathname.current !== null && prevPathname.current !== pathname) {
      mainRef.current?.focus()
    }
    prevPathname.current = pathname
  }, [pathname])

  return (
    <LiveAnnouncerProvider>
      <CreateProjectDialogProvider>
        <div className="min-h-screen flex flex-col bg-[var(--color-bg)]">
          <ProjectSwitcher />
          <div className="flex flex-1 min-h-0">
            {isProjectRoute && projectSlug !== null && (
              <Sidebar projectSlug={projectSlug} envSlug={envSlug} />
            )}
            <main
              id="main-content"
              ref={mainRef}
              tabIndex={-1}
              className="flex-1 min-w-0 outline-none"
            >
              <Outlet />
            </main>
          </div>
        </div>
      </CreateProjectDialogProvider>
    </LiveAnnouncerProvider>
  )
}
