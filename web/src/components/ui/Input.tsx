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
        className={`block w-full rounded border px-3 py-2 text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-[var(--color-accent)] ${
          invalid ? 'border-red-400 dark:border-red-500' : 'border-gray-300 dark:border-gray-600'
        } ${className}`}
        {...props}
      />
    )
  },
)
Input.displayName = 'Input'
