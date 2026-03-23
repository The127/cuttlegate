import * as RadixDialog from '@radix-ui/react-dialog'
import { forwardRef, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// ── Re-exports of unstyled Radix primitives ──────────────────────────────────

/**
 * Root dialog controller — accepts `open` and `onOpenChange` for controlled usage.
 * Always use the controlled pattern: `<Dialog open={...} onOpenChange={...}>`.
 */
export const Dialog = RadixDialog.Root
Dialog.displayName = 'Dialog'

/**
 * Element that opens the dialog when clicked.
 * Wrap your trigger button with this — Radix handles aria-expanded and aria-haspopup.
 */
export const DialogTrigger = RadixDialog.Trigger
DialogTrigger.displayName = 'DialogTrigger'

// ── Styled sub-components ────────────────────────────────────────────────────

interface DialogContentProps {
  children: ReactNode
  /** Additional class names applied to the content panel. */
  className?: string
}

/**
 * Overlay + content panel. Renders into a Portal so it is always on top of the page.
 * Provides focus trapping, Escape-to-close, and overlay-click-to-close via Radix.
 */
export const DialogContent = forwardRef<HTMLDivElement, DialogContentProps>(
  ({ children, className = '' }, ref) => (
    <RadixDialog.Portal>
      <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/40 dark:bg-black/60" />
      <RadixDialog.Content
        ref={ref}
        className={`fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-xl p-6 focus:outline-none ${className}`}
      >
        {children}
      </RadixDialog.Content>
    </RadixDialog.Portal>
  ),
)
DialogContent.displayName = 'DialogContent'

// ── Layout helpers ───────────────────────────────────────────────────────────

export function DialogHeader({ children }: { children: ReactNode }) {
  return <div className="mb-4">{children}</div>
}
DialogHeader.displayName = 'DialogHeader'

/**
 * Styled dialog title — wired to aria-labelledby on the content panel by Radix.
 */
export function DialogTitle({
  children,
  className = '',
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <RadixDialog.Title
      className={`text-base font-semibold text-gray-900 dark:text-gray-100 ${className}`}
    >
      {children}
    </RadixDialog.Title>
  )
}
DialogTitle.displayName = 'DialogTitle'

/**
 * Styled dialog description — wired to aria-describedby on the content panel by Radix.
 * Omit if there is no description; Radix will not set aria-describedby when this is absent.
 */
export function DialogDescription({
  children,
  className = '',
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <RadixDialog.Description
      className={`mt-1 text-sm text-gray-500 dark:text-gray-400 ${className}`}
    >
      {children}
    </RadixDialog.Description>
  )
}
DialogDescription.displayName = 'DialogDescription'

export function DialogFooter({ children }: { children: ReactNode }) {
  return <div className="mt-6 flex justify-end gap-3">{children}</div>
}
DialogFooter.displayName = 'DialogFooter'

/**
 * Unstyled Radix close trigger — wrap any element (e.g. an icon button) with this
 * to wire up Escape-and-click close via Radix. For a labelled close button, use
 * the `DialogCloseButton` convenience component instead.
 *
 * This is a thin wrapper over `@radix-ui/react-dialog` Close — it does not add styling.
 */
export const DialogClose = RadixDialog.Close
DialogClose.displayName = 'DialogClose'

// ── Convenience components ───────────────────────────────────────────────────

/**
 * A styled close button that renders in the top-right corner of the dialog.
 * Uses t('actions.close', { ns: 'common' }) for its accessible label.
 * Compose into DialogContent when a visible dismiss affordance is needed.
 */
export function DialogCloseButton() {
  const { t } = useTranslation('common')
  return (
    <RadixDialog.Close
      aria-label={t('actions.close')}
      className="absolute right-4 top-4 rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M12.207 3.793a1 1 0 0 0-1.414 0L8 6.586 5.207 3.793a1 1 0 0 0-1.414 1.414L6.586 8l-2.793 2.793a1 1 0 1 0 1.414 1.414L8 9.414l2.793 2.793a1 1 0 0 0 1.414-1.414L9.414 8l2.793-2.793a1 1 0 0 0 0-1.414z" />
      </svg>
    </RadixDialog.Close>
  )
}
DialogCloseButton.displayName = 'DialogCloseButton'
