import { forwardRef, type InputHTMLAttributes } from 'react'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  hasError?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ hasError = false, className = '', 'aria-invalid': ariaInvalid, ...props }, ref) => {
    const invalid = ariaInvalid === true || ariaInvalid === 'true' || hasError
    return (
      <input
        ref={ref}
        aria-invalid={invalid || undefined}
        className={`block w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-[var(--color-accent)] ${
          invalid ? 'border-red-400' : 'border-gray-300'
        } ${className}`}
        {...props}
      />
    )
  },
)
Input.displayName = 'Input'
