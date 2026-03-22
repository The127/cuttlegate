import { useId, type ReactNode } from 'react'
import { Label } from './Label'

export interface FormFieldProps {
  label: string
  htmlFor?: string
  error?: string
  children: ReactNode
  className?: string
}

export function FormField({ label, htmlFor, error, children, className = '' }: FormFieldProps) {
  const generatedId = useId()
  const errorId = `${generatedId}-error`
  const fieldId = htmlFor ?? generatedId

  return (
    <div className={className}>
      <Label htmlFor={fieldId}>{label}</Label>
      <div
        className="mt-1"
        // Pass aria-describedby context down via data attribute so consumers can wire it
        data-error-id={error ? errorId : undefined}
      >
        {children}
      </div>
      {error && (
        <p id={errorId} className="mt-1 text-xs text-red-600" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
