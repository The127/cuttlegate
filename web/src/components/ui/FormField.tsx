import { createContext, useContext, useId, type ReactNode } from 'react'
import { Label } from './Label'

export interface FormFieldContextValue {
  fieldId: string
  errorId: string | undefined
}

export const FormFieldContext = createContext<FormFieldContextValue | null>(null)

export function useFormFieldId(): FormFieldContextValue | null {
  return useContext(FormFieldContext)
}

export interface FormFieldProps {
  label: string
  htmlFor?: string
  error?: string
  children: ReactNode
  className?: string
}

export function FormField({ label, htmlFor, error, children, className = '' }: FormFieldProps) {
  const generatedId = useId()
  const errorId = error ? `${generatedId}-error` : undefined
  const fieldId = htmlFor ?? generatedId

  return (
    <FormFieldContext.Provider value={{ fieldId, errorId }}>
      <div className={className}>
        <Label htmlFor={fieldId}>{label}</Label>
        <div className="mt-1">
          {children}
        </div>
        {error && (
          <p id={errorId} className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">
            {error}
          </p>
        )}
      </div>
    </FormFieldContext.Provider>
  )
}
