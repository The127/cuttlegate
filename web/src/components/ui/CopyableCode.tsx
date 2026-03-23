import { useState, useEffect, useRef } from 'react'

export interface CopyableCodeProps {
  /** The value to display and copy to clipboard */
  value: string
  /** Accessible label for the copy button — defaults to "Copy <value>" */
  'aria-label'?: string
  className?: string
}

/**
 * CopyableCode — renders a value in monospace with a click-to-copy affordance.
 *
 * On click: writes value to clipboard, flips icon to checkmark for 1 second,
 * then reverts. Silent failure if clipboard API is unavailable (non-HTTPS / permission denied).
 *
 * Uses only inline SVG — no icon library dependency.
 */
export function CopyableCode({ value, 'aria-label': ariaLabel, className = '' }: CopyableCodeProps) {
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current)
    }
  }, [])

  function handleCopy() {
    if (!navigator.clipboard) return
    void navigator.clipboard
      .writeText(value)
      .then(() => {
        setCopied(true)
        timerRef.current = setTimeout(() => setCopied(false), 1000)
      })
      .catch(() => {
        // clipboard write unavailable (non-HTTPS or permission denied) — silent
      })
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={ariaLabel ?? `Copy ${value}`}
      className={`inline-flex items-center gap-1.5 font-mono text-sm text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-2 py-0.5 hover:text-[var(--color-accent)] hover:border-[var(--color-accent)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] ${className}`}
    >
      <span>{value}</span>
      {copied ? <CheckIcon /> : <ClipboardIcon />}
    </button>
  )
}

function ClipboardIcon() {
  return (
    <svg
      className="w-3.5 h-3.5 shrink-0 text-[var(--color-text-muted)]"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
      <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
    </svg>
  )
}

function CheckIcon() {
  return (
    <svg
      className="w-3.5 h-3.5 shrink-0 text-[var(--color-status-enabled)]"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
        clipRule="evenodd"
      />
    </svg>
  )
}
