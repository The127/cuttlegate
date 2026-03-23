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
      'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:border-[var(--color-border-hover)]',
  },
  {
    value: 'write',
    labelKey: 'api_keys.tier_write_label',
    descriptionKey: 'api_keys.tier_write_description',
    activeClasses:
      'bg-[rgba(79,124,255,0.3)] text-[#818cf8] border-[rgba(79,124,255,0.5)]',
    inactiveClasses:
      'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:border-[var(--color-border-hover)]',
  },
  {
    value: 'destructive',
    labelKey: 'api_keys.tier_destructive_label',
    descriptionKey: '',
    activeClasses:
      'bg-[rgba(251,191,36,0.25)] text-[#fbbf24] border-[rgba(251,191,36,0.5)]',
    inactiveClasses:
      'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:border-[var(--color-border-hover)]',
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
        <p className="mt-1.5 text-xs text-[var(--color-status-warning)]">
          {t('api_keys.tier_destructive_warning')}
        </p>
      )}
      {activeTier?.descriptionKey && (
        <p className="mt-1.5 text-xs text-[var(--color-text-secondary)]">
          {t(activeTier.descriptionKey)}
        </p>
      )}
    </div>
  )
}
