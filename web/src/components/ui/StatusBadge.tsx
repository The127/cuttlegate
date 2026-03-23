export type BadgeStatus = 'enabled' | 'disabled' | 'warning' | 'unknown'

// All colour classes reference design tokens from styles.css / tokens.md.
// Status colours are used on white/dark backgrounds with a text label — per the
// WCAG 2.1 AA usage rule in tokens.md, standalone colour text is not permitted;
// the label is always present here.
// Dark mode dot colours adapt via the CSS custom property override in styles.css.
// Dark mode pill colours: chosen for WCAG AA at gray-900 background (issue #241 spec).
const statusConfig: Record<
  BadgeStatus,
  { dot: string; pill: string; label: string }
> = {
  enabled: {
    dot: 'bg-[var(--color-status-enabled)]',
    pill: 'bg-green-50 text-green-700 border-green-200 dark:bg-green-950 dark:text-green-300 dark:border-green-800',
    label: 'Enabled',
  },
  disabled: {
    dot: 'bg-[var(--color-status-error)]',
    pill: 'bg-red-50 text-red-700 border-red-200 dark:bg-red-950 dark:text-red-300 dark:border-red-800',
    label: 'Disabled',
  },
  warning: {
    dot: 'bg-[var(--color-status-warning)]',
    pill: 'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-950 dark:text-amber-300 dark:border-amber-800',
    label: 'Warning',
  },
  unknown: {
    // bg-gray-400 dot is mode-agnostic — visible on both light and dark surfaces
    dot: 'bg-gray-400',
    pill: 'bg-gray-100 text-gray-500 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700',
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
