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

const OPTIONS: ThemePreference[] = ['system', 'light', 'dark']

export function ThemeToggle() {
  const { t } = useTranslation('common')
  const [current, setCurrent] = useState<ThemePreference>(readPreference)

  // Apply theme class whenever preference changes (including initial mount).
  useEffect(() => {
    applyPreference(current)
  }, [current])

  return (
    <div className="flex items-center gap-1" role="radiogroup" aria-label={t('theme.label')}>
      {OPTIONS.map((opt) => (
        <button
          key={opt}
          type="button"
          role="radio"
          aria-checked={current === opt}
          onClick={() => setCurrent(opt)}
          className={`px-2 py-1 text-xs rounded transition-colors ${
            current === opt
              ? 'bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)] font-medium'
              : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
          }`}
        >
          {t(`theme.${opt}`)}
        </button>
      ))}
    </div>
  )
}
