export type BadgeStatus = 'enabled' | 'disabled' | 'warning' | 'unknown'

// All colour classes reference design tokens from styles.css / tokens.md.
// Status colours are used on white backgrounds with a text label — per the
// WCAG 2.1 AA usage rule in tokens.md, standalone colour text on white is
// not permitted; the label is always present here.
const statusConfig: Record<
  BadgeStatus,
  { dot: string; pill: string; label: string }
> = {
  enabled: {
    dot: 'bg-[var(--color-status-enabled)]',
    pill: 'bg-green-50 text-green-700 border-green-200',
    label: 'Enabled',
  },
  disabled: {
    dot: 'bg-[var(--color-status-error)]',
    pill: 'bg-red-50 text-red-700 border-red-200',
    label: 'Disabled',
  },
  warning: {
    dot: 'bg-[var(--color-status-warning)]',
    pill: 'bg-amber-50 text-amber-700 border-amber-200',
    label: 'Warning',
  },
  unknown: {
    dot: 'bg-gray-400',
    pill: 'bg-gray-100 text-gray-500 border-gray-200',
    label: 'Unknown',
  },
}

export interface StatusBadgeProps {
  status: BadgeStatus
  /** Override the displayed text label. Defaults to the capitalised status name. */
  label?: string
  className?: string
}

/**
 * StatusBadge — coloured dot + text label for flag/environment state.
 *
 * Colours come exclusively from design tokens (see web/src/design/tokens.md).
 * The dot is always accompanied by a visible text label for colour-blind safety.
 */
export function StatusBadge({ status, label, className = '' }: StatusBadgeProps) {
  const config = statusConfig[status]
  const displayLabel = label ?? config.label

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium border ${config.pill} ${className}`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full shrink-0 ${config.dot}`}
        aria-hidden="true"
      />
      {displayLabel}
    </span>
  )
}
