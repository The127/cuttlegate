import { useEffect } from 'react'
import { useBrand } from '../brand'

/**
 * Sets `document.title` from the provided segments.
 *
 * Defined segments are joined with ` — ` (em dash) and the brand app name
 * is appended after a ` | ` separator.  Undefined segments (still loading)
 * are silently omitted.  If every segment is undefined the title falls back
 * to the bare app name.
 *
 * @example
 *   useDocumentTitle('Flags', envSlug, project?.name)
 *   // → "Flags — staging — Acme Corp | Cuttlegate"
 */
export function useDocumentTitle(...segments: (string | undefined)[]) {
  const { app_name } = useBrand()

  useEffect(() => {
    const defined = segments.filter((s): s is string => s !== undefined)
    document.title = defined.length > 0
      ? `${defined.join(' \u2014 ')} | ${app_name}`
      : app_name
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...segments, app_name])
}
