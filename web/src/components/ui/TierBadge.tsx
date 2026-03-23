import { useTranslation } from 'react-i18next'

export type ToolCapabilityTier = 'read' | 'write' | 'destructive'

const tierConfig: Record<ToolCapabilityTier, { classes: string; labelKey: string }> = {
  read: {
    classes:
      'bg-neutral-100 text-neutral-600 border border-neutral-200 dark:bg-neutral-800 dark:text-neutral-400 dark:border-neutral-700',
    labelKey: 'api_keys.tier_read_label',
  },
  write: {
    classes:
      'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-950 dark:text-blue-300 dark:border-blue-800',
    labelKey: 'api_keys.tier_write_label',
  },
  destructive: {
    classes:
      'bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950 dark:text-amber-300 dark:border-amber-800',
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
 * read: grey, write: blue, destructive: amber.
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
