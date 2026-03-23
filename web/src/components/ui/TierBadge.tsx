import { useTranslation } from 'react-i18next'

export type ToolCapabilityTier = 'read' | 'write' | 'destructive'

// Dark-by-default tier badge colours per docs/ui-design.md TierBadge spec.
// Uses rgba() values so no dark: variant prefix is needed.
const tierConfig: Record<ToolCapabilityTier, { classes: string; labelKey: string }> = {
  read: {
    classes:
      'bg-[rgba(255,255,255,0.06)] text-[var(--color-text-secondary)] border border-[var(--color-border)]',
    labelKey: 'api_keys.tier_read_label',
  },
  write: {
    classes:
      'bg-[rgba(79,124,255,0.15)] text-[#818cf8] border border-[rgba(79,124,255,0.3)]',
    labelKey: 'api_keys.tier_write_label',
  },
  destructive: {
    classes:
      'bg-[rgba(251,191,36,0.12)] text-[#fbbf24] border border-[rgba(251,191,36,0.25)]',
    labelKey: 'api_keys.tier_destructive_label',
  },
}

export interface TierBadgeProps {
  tier: ToolCapabilityTier
  className?: string
}

/**
 * TierBadge — coloured pill for API key capability tier.
 *
 * read: grey, write: blue-purple, destructive: amber.
 * See docs/ui-design.md for the tier badge colour convention.
 */
export function TierBadge({ tier, className = '' }: TierBadgeProps) {
  const { t } = useTranslation('projects')
  const config = tierConfig[tier] ?? tierConfig.read

  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${config.classes} ${className}`}
    >
      {t(config.labelKey)}
    </span>
  )
}
