import type { ReactNode } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

// ── Nav icons (16px inline SVG) ───────────────────────────────────────────────

function IconFlag() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
      <line x1="4" y1="22" x2="4" y2="15" />
    </svg>
  )
}

function IconUsers() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  )
}

function IconKey() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="7.5" cy="15.5" r="5.5" />
      <path d="M21 2l-9.6 9.6" />
      <path d="M15.5 7.5L17 6l3 3-1.5 1.5" />
    </svg>
  )
}

function IconUserCircle() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="10" />
      <circle cx="12" cy="10" r="3" />
      <path d="M7 20.662V19a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v1.662" />
    </svg>
  )
}

function IconScrollText() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 21h12a2 2 0 0 0 2-2v-2H10v2a2 2 0 1 1-4 0V5a2 2 0 1 0-4 0v3h4" />
      <path d="M19 17V5a2 2 0 0 0-2-2H4" />
      <line x1="13" y1="11" x2="17" y2="11" />
      <line x1="13" y1="15" x2="17" y2="15" />
    </svg>
  )
}

function IconSettings() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  )
}

// ── Nav item ──────────────────────────────────────────────────────────────────

interface NavItemProps {
  to: string
  icon: ReactNode
  label: string
  isActive: boolean
}

function NavItem({ to, icon, label, isActive }: NavItemProps) {
  return (
    <li>
      <Link
        to={to}
        aria-current={isActive ? 'page' : undefined}
        className={[
          'relative flex items-center gap-2.5 w-full px-3 py-2 text-sm rounded-[var(--radius-sm)] transition-colors',
          isActive
            ? 'bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)]'
            : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-elevated)]',
        ].join(' ')}
      >
        {isActive && (
          <span
            className="absolute left-0 top-0 bottom-0 w-[3px] rounded-l-[var(--radius-sm)]"
            style={{ background: 'linear-gradient(to bottom, var(--color-accent-start), var(--color-accent-end))' }}
            aria-hidden="true"
          />
        )}
        {icon}
        {label}
      </Link>
    </li>
  )
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

interface SidebarProps {
  projectSlug: string
  envSlug: string | null
}

export function Sidebar({ projectSlug, envSlug }: SidebarProps) {
  const { t } = useTranslation('common')
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const flagsPath = envSlug
    ? `/projects/${projectSlug}/environments/${envSlug}/flags`
    : `/projects/${projectSlug}`

  const envScopedItems: NavItemProps[] = [
    {
      to: flagsPath,
      icon: <IconFlag />,
      label: t('nav.flags'),
      isActive: pathname.includes('/flags'),
    },
    {
      to: `/projects/${projectSlug}/segments`,
      icon: <IconUsers />,
      label: t('nav.segments'),
      isActive: pathname.includes('/segments'),
    },
  ]

  const projectScopedItems: NavItemProps[] = [
    {
      to: `/projects/${projectSlug}/api-keys`,
      icon: <IconKey />,
      label: t('nav.api_keys'),
      isActive: pathname.includes('/api-keys'),
    },
    {
      to: `/projects/${projectSlug}/members`,
      icon: <IconUserCircle />,
      label: t('nav.members'),
      isActive: pathname.includes('/members'),
    },
    {
      to: `/projects/${projectSlug}/audit`,
      icon: <IconScrollText />,
      label: t('nav.audit'),
      isActive: pathname.includes('/audit'),
    },
    {
      to: `/projects/${projectSlug}/settings`,
      icon: <IconSettings />,
      label: t('nav.settings'),
      isActive: pathname.includes('/settings'),
    },
  ]

  return (
    <aside
      className="w-56 shrink-0 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col"
      aria-label="Project navigation"
    >
      <nav className="flex-1 px-2 py-4">
        <ul className="flex flex-col gap-0.5">
          {envScopedItems.map((item) => (
            <NavItem key={item.to} {...item} />
          ))}
        </ul>
        <div
          className="my-2 mx-3 h-px bg-[var(--color-border)]"
          aria-hidden="true"
        />
        <ul className="flex flex-col gap-0.5">
          {projectScopedItems.map((item) => (
            <NavItem key={item.to} {...item} />
          ))}
        </ul>
      </nav>
    </aside>
  )
}
