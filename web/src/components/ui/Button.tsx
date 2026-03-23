import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'

export type ButtonVariant =
  | 'primary'
  | 'secondary'
  | 'ghost'
  | 'danger'
  | 'danger-outline'
  | 'destructive'
export type ButtonSize = 'sm' | 'md' | 'lg'

// Destructive/danger: solid red per docs/ui-design.md — no gradient, no subtlety.
const DANGER_CLASSES =
  'bg-[#f87171] text-white hover:bg-[#ef4444] focus:ring-[var(--color-status-error)] disabled:opacity-50 disabled:cursor-not-allowed'

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-[linear-gradient(135deg,var(--color-accent-start),var(--color-accent-end))] text-white hover:shadow-[0_0_16px_rgba(0,212,170,0.25)] focus:ring-[var(--color-accent)] disabled:opacity-50 disabled:cursor-not-allowed',
  secondary:
    'bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)] border border-[var(--color-border)] hover:border-[var(--color-border-hover)] focus:ring-[var(--color-accent)] disabled:opacity-50 disabled:cursor-not-allowed',
  ghost:
    'bg-transparent text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-elevated)] hover:text-[var(--color-text-primary)] focus:ring-[var(--color-accent)] disabled:opacity-50 disabled:cursor-not-allowed',
  danger: DANGER_CLASSES,
  'danger-outline':
    'bg-transparent text-[#f87171] border border-[var(--color-border)] hover:border-[#f87171] focus:ring-[var(--color-status-error)] disabled:opacity-50 disabled:cursor-not-allowed',
  // destructive is a semantic alias for danger — same visual style
  destructive: DANGER_CLASSES,
}

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'px-2.5 py-1 text-sm',
  md: 'px-3 py-1.5 text-sm',
  lg: 'px-4 py-2 text-sm',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  /** Shows an inline spinner and disables the button while true */
  loading?: boolean
  children?: ReactNode
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    { variant = 'primary', size = 'md', loading = false, className = '', children, disabled, ...props },
    ref,
  ) => (
    <button
      ref={ref}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`inline-flex items-center justify-center gap-1.5 font-medium rounded-[var(--radius-md)] focus:outline-none focus:ring-2 ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
      {...props}
    >
      {loading && (
        <svg
          className="animate-spin w-3.5 h-3.5 shrink-0"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      )}
      {children}
    </button>
  ),
)
Button.displayName = 'Button'
