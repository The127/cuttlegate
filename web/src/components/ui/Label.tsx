import { forwardRef, type LabelHTMLAttributes } from 'react'

export type LabelProps = LabelHTMLAttributes<HTMLLabelElement>

export const Label = forwardRef<HTMLLabelElement, LabelProps>(
  ({ className = '', ...props }, ref) => (
    <label
      ref={ref}
      className={`block text-sm font-medium text-[var(--color-text-primary)] ${className}`}
      {...props}
    />
  ),
)
Label.displayName = 'Label'
