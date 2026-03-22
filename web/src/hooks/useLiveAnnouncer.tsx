import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from 'react'

interface LiveAnnouncerContextValue {
  announce: (message: string) => void
}

const LiveAnnouncerContext = createContext<LiveAnnouncerContextValue>({
  announce: () => {},
})

/**
 * Provides a visually-hidden aria-live="polite" region for screen reader
 * announcements. Wrap the application shell with this provider.
 *
 * Use `useLiveAnnouncer()` to get the `announce(message)` function anywhere
 * in the tree.
 */
export function LiveAnnouncerProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState('')
  // Toggle an invisible character so repeated identical messages re-trigger
  // the live region even when the string content hasn't changed.
  const toggleRef = useRef(false)

  const announce = useCallback((msg: string) => {
    toggleRef.current = !toggleRef.current
    setMessage(msg + (toggleRef.current ? '\u200B' : '\u200C'))
  }, [])

  return (
    <LiveAnnouncerContext.Provider value={{ announce }}>
      {children}
      <div
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {message}
      </div>
    </LiveAnnouncerContext.Provider>
  )
}

export function useLiveAnnouncer(): (message: string) => void {
  return useContext(LiveAnnouncerContext).announce
}
