import { Outlet, createRootRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFoundPage,
})

function NotFoundPage() {
  const { t } = useTranslation('common')
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="text-center">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{t('not_found.title')}</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">{t('not_found.body')}</p>
        <a href="/" className="mt-4 inline-block text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300">
          {t('not_found.return_to_home')}
        </a>
      </div>
    </div>
  )
}
