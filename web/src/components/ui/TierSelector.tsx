import { useTranslation } from 'react-i18next'
import type { ToolCapabilityTier } from './TierBadge'

export type { ToolCapabilityTier }

interface TierOption {
  value: ToolCapabilityTier
  labelKey: string
  descriptionKey: string
  activeClasses: string
  inactiveClasses: string
}

const TIERS: TierOption[] = [
  {
    value: 'read',
    labelKey: 'api_keys.tier_read_label',
    descriptionKey: 'api_keys.tier_read_description',
    activeClasses:
      'bg-[var(--color-accent)] text-white border-[var(--color-accent)]',
    inactiveClasses:
      'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700',
  },
  {
    value: 'write',
    labelKey: 'api_keys.tier_write_label',
    descriptionKey: 'api_keys.tier_write_description',
    activeClasses:
      'bg-blue-600 text-white border-blue-600',
    inactiveClasses:
      'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700',
  },
  {
    value: 'destructive',
    labelKey: 'api_keys.tier_destructive_label',
    descriptionKey: '',
    activeClasses:
      'bg-amber-500 text-white border-amber-500',
    inactiveClasses:
      'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700',
  },
]

export interface TierSelectorProps {
  value: ToolCapabilityTier
  onChange: (tier: ToolCapabilityTier) => void
}

/**
 * TierSelector — button-group for selecting an API key capability tier.
 *
 * Controlled component. Caller must pre-select 'read' as the default.
 * Destructive tier uses amber (not red) — amber = destructive capability,
 * red = destructive action (e.g. delete button). See docs/ui-design.md.
 */
export function TierSelector({ value, onChange }: TierSelectorProps) {
  const { t } = useTranslation('projects')
  const activeTier = TIERS.find(tier => tier.value === value)

  return (
    <div>
      <div className="flex rounded-md shadow-sm" role="group" aria-label={t('api_keys.capability_tier_label')}>
        {TIERS.map((tier, index) => {
          const isActive = value === tier.value
          const isFirst = index === 0
          const isLast = index === TIERS.length - 1
          const roundedClasses = isFirst
            ? 'rounded-l-md'
            : isLast
              ? 'rounded-r-md'
              : ''
          return (
            <button
              key={tier.value}
              type="button"
              onClick={() => onChange(tier.value)}
              aria-pressed={isActive}
              className={`flex-1 px-3 py-1.5 text-sm font-medium border focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:z-10 ${roundedClasses} ${isActive ? tier.activeClasses : tier.inactiveClasses} ${!isFirst ? '-ml-px' : ''}`}
            >
              {t(tier.labelKey)}
            </button>
          )
        })}
      </div>
      {value === 'destructive' && (
        <p className="mt-1.5 text-xs text-amber-700 dark:text-amber-400">
          {t('api_keys.tier_destructive_warning')}
        </p>
      )}
      {activeTier?.descriptionKey && (
        <p className="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {t(activeTier.descriptionKey)}
        </p>
      )}
    </div>
  )
}
