import { Link, useLocation } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

interface SettingsTabBarProps {
  slug: string
}

export function SettingsTabBar({ slug }: SettingsTabBarProps) {
  const { t } = useTranslation('projects')
  const { pathname } = useLocation()

  const generalPath = `/projects/${slug}/settings`
  const environmentsPath = `/projects/${slug}/settings/environments`

  const tabs = [
    {
      label: t('settings.tab_general'),
      to: generalPath,
      isActive: pathname === generalPath,
    },
    {
      label: t('settings.tab_environments'),
      to: environmentsPath,
      isActive: pathname.startsWith(environmentsPath),
    },
  ]

  return (
    <nav
      className="mb-6 border-b border-[var(--color-border)]"
      aria-label={t('settings.title')}
    >
      <div className="flex gap-0">
        {tabs.map((tab) => (
          <Link
            key={tab.to}
            to={tab.to}
            className={`relative px-3 pb-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:rounded ${
              tab.isActive
                ? 'text-[var(--color-text-primary)]'
                : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            {tab.label}
            {tab.isActive && (
              <span
                className="absolute bottom-0 left-0 right-0 h-[2px]"
                style={{
                  background:
                    'linear-gradient(135deg, var(--color-accent-start), var(--color-accent-end))',
                }}
              />
            )}
          </Link>
        ))}
      </div>
    </nav>
  )
}
