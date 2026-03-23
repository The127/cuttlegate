export type BadgeStatus = 'enabled' | 'disabled' | 'warning' | 'unknown'

// All colour classes reference design tokens from styles.css @theme block.
// Pill backgrounds use rgba() tints per docs/ui-design.md StatusBadge spec.
// Dark is the default — no dark: variant needed.
const statusConfig: Record<
  BadgeStatus,
  { dot: string; pill: string; label: string }
> = {
  enabled: {
    dot: 'bg-[var(--color-status-enabled)]',
    pill: 'bg-[rgba(16,217,168,0.12)] text-[var(--color-status-enabled)] border-[rgba(16,217,168,0.25)]',
    label: 'Enabled',
  },
  disabled: {
    dot: 'bg-[var(--color-status-error)]',
    pill: 'bg-[rgba(248,113,113,0.12)] text-[var(--color-status-error)] border-[rgba(248,113,113,0.25)]',
    label: 'Disabled',
  },
  warning: {
    dot: 'bg-[var(--color-status-warning)]',
    pill: 'bg-[rgba(251,191,36,0.12)] text-[var(--color-status-warning)] border-[rgba(251,191,36,0.25)]',
    label: 'Warning',
  },
  unknown: {
    dot: 'bg-[var(--color-surface-elevated)]',
    pill: 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)]',
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
 * Colours come exclusively from design tokens (see web/src/styles.css @theme).
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
