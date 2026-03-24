import * as RadixSelect from '@radix-ui/react-select'
import { forwardRef, type ReactNode } from 'react'

// ── SelectItem ──────────────────────────────────────────────────────────────

export interface SelectItemProps {
  value: string
  children: ReactNode
  disabled?: boolean
}

export const SelectItem = forwardRef<HTMLDivElement, SelectItemProps>(
  ({ value, children, disabled }, ref) => (
    <RadixSelect.Item
      ref={ref}
      value={value}
      disabled={disabled}
      className="relative flex items-center pl-6 pr-3 py-1.5 text-sm text-[var(--color-text-primary)] rounded cursor-default select-none data-[highlighted]:bg-[var(--color-surface-elevated)] data-[highlighted]:outline-none data-[disabled]:opacity-50"
    >
      <RadixSelect.ItemText>{children}</RadixSelect.ItemText>
      <RadixSelect.ItemIndicator className="absolute left-1 text-[var(--color-accent)]">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true">
          <path d="M10.28 2.28L3.989 8.575 1.695 6.28A1 1 0 00.28 7.695l3 3a1 1 0 001.414 0l7-7A1 1 0 0010.28 2.28z" />
        </svg>
      </RadixSelect.ItemIndicator>
    </RadixSelect.Item>
  ),
)
SelectItem.displayName = 'SelectItem'

// ── Select ──────────────────────────────────────────────────────────────────

export interface SelectProps {
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  children: ReactNode
  'aria-label'?: string
  'aria-labelledby'?: string
}

export function Select({
  value,
  onValueChange,
  placeholder,
  disabled,
  className = '',
  children,
  'aria-label': ariaLabel,
  'aria-labelledby': ariaLabelledby,
}: SelectProps) {
  return (
    <RadixSelect.Root value={value} onValueChange={onValueChange} disabled={disabled}>
      <RadixSelect.Trigger
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledby}
        className={`inline-flex items-center justify-between gap-1 text-sm border border-[var(--color-border)] rounded-[var(--radius-md)] px-2 py-1.5 bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-accent)] focus:shadow-[0_0_0_2px_rgba(79,124,255,0.4)] disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      >
        <RadixSelect.Value placeholder={placeholder} />
        <RadixSelect.Icon className="text-[var(--color-text-muted)] ml-1">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true">
            <path d="M6 8L1 3h10z" />
          </svg>
        </RadixSelect.Icon>
      </RadixSelect.Trigger>
      <RadixSelect.Portal>
        <RadixSelect.Content
          position="popper"
          sideOffset={4}
          className="z-50 min-w-[8rem] overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] shadow-md"
        >
          <RadixSelect.Viewport className="p-1">{children}</RadixSelect.Viewport>
        </RadixSelect.Content>
      </RadixSelect.Portal>
    </RadixSelect.Root>
  )
}
