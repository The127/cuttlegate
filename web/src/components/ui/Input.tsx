import { forwardRef, type InputHTMLAttributes } from 'react'
import { useFormFieldId } from './FormField'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  hasError?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      hasError = false,
      className = '',
      'aria-invalid': ariaInvalid,
      id: idProp,
      'aria-describedby': ariaDescribedByProp,
      ...props
    },
    ref,
  ) => {
    const fieldCtx = useFormFieldId()
    const invalid = ariaInvalid === true || ariaInvalid === 'true' || hasError

    // Caller-supplied props win over context-derived values.
    const id = idProp ?? fieldCtx?.fieldId
    const ariaDescribedBy =
      ariaDescribedByProp ?? (fieldCtx?.errorId ? fieldCtx.errorId : undefined)

    return (
      <input
        ref={ref}
        id={id}
        aria-invalid={invalid || undefined}
        aria-describedby={ariaDescribedBy}
        className={`block w-full rounded-[var(--radius-md)] border px-3 py-2 text-sm bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-accent)] focus:shadow-[0_0_0_2px_rgba(79,124,255,0.4)] ${
          invalid
            ? 'border-[#f87171] focus:shadow-[0_0_0_2px_rgba(248,113,113,0.4)]'
            : 'border-[var(--color-border)]'
        } ${className}`}
        {...props}
      />
    )
  },
)
Input.displayName = 'Input'
