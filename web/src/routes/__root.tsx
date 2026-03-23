import { Outlet, createRootRoute, useLocation } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

export const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFoundPage,
})

export function NotFoundPage() {
  const { t } = useTranslation('common')
  const location = useLocation()
  const url = location.pathname

  useEffect(() => {
    document.title = `${t('not_found.title')} — ${t('app_name')}`
  }, [t])

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-surface-elevated)]">
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
