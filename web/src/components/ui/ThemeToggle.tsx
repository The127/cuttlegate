import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

type ThemePreference = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'cg:theme.preference'
const VALID_VALUES = new Set<ThemePreference>(['system', 'light', 'dark'])

/** Read the stored preference, falling back to "system" on missing/invalid/error. */
function readPreference(): ThemePreference {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw !== null && VALID_VALUES.has(raw as ThemePreference)) {
      return raw as ThemePreference
    }
  } catch {
    // localStorage unavailable — graceful fallback
  }
  return 'system'
}

/** Apply the theme class to <html> and persist the preference. */
function applyPreference(pref: ThemePreference): void {
  const el = document.documentElement
  el.classList.remove('theme-light', 'theme-dark')

  if (pref === 'light') {
    el.classList.add('theme-light')
  } else if (pref === 'dark') {
    el.classList.add('theme-dark')
  }
  // "system" → no class; CSS @media governs.

  try {
    if (pref === 'system') {
      localStorage.removeItem(STORAGE_KEY)
    } else {
      localStorage.setItem(STORAGE_KEY, pref)
    }
  } catch {
    // localStorage unavailable — selection still applies for this session
  }
}

/** Inline SVG icons — 16×16, currentColor stroke, aria-hidden. */
function MonitorIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2" y="3" width="20" height="14" rx="2" />
      <path d="M8 21h8M12 17v4" />
    </svg>
  )
}

function SunIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  )
}

const OPTIONS: { value: ThemePreference; icon: () => React.JSX.Element }[] = [
  { value: 'system', icon: MonitorIcon },
  { value: 'light', icon: SunIcon },
  { value: 'dark', icon: MoonIcon },
]

export function ThemeToggle() {
  const { t } = useTranslation('common')
  const [current, setCurrent] = useState<ThemePreference>(readPreference)

  // Apply theme class whenever preference changes (including initial mount).
  useEffect(() => {
    applyPreference(current)
  }, [current])

  return (
    <div className="flex items-center gap-1" role="radiogroup" aria-label={t('theme.label')}>
      {OPTIONS.map(({ value, icon: Icon }) => (
        <button
          key={value}
          type="button"
          role="radio"
          aria-checked={current === value}
          aria-label={t(`theme.${value}`)}
          title={t(`theme.${value}`)}
          onClick={() => setCurrent(value)}
          className={`p-1.5 rounded transition-colors ${
            current === value
              ? 'bg-[var(--color-surface-elevated)] text-[var(--color-accent)]'
              : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
          }`}
        >
          <Icon />
        </button>
      ))}
    </div>
  )
}
