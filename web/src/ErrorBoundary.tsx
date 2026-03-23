import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

/**
 * Top-level error boundary wrapping <RouterProvider> in main.tsx.
 *
 * Catches uncaught render errors across the entire React tree.
 *
 * Strings are hardcoded English — this is the last-resort fallback UI and
 * useTranslation() is unavailable in class components without a HOC wrapper
 * that could itself fail. Approved exception to the i18n rule for this
 * component only.
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(_error: unknown): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('ErrorBoundary caught an error:', error, info)
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div
          className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]"
          role="alert"
        >
          <div className="text-center max-w-sm">
            <h1 className="text-2xl font-semibold text-[var(--color-text-primary)]">
              Something went wrong
            </h1>
            <p className="mt-2 text-[var(--color-text-secondary)]">
              An unexpected error occurred. Please reload the page.
            </p>
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="mt-6 inline-flex items-center justify-center gap-1.5 font-medium rounded-[var(--radius-md)] px-4 py-2 text-sm bg-[linear-gradient(135deg,var(--color-accent-start),var(--color-accent-end))] text-white hover:shadow-[0_0_16px_rgba(0,212,170,0.25)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            >
              Reload
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
