import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getUserManager } from '../auth'

/**
 * Subscribes to the SSE flag state stream for a given project/environment.
 * When a `flag.state_changed` event arrives, invalidates the relevant
 * TanStack Query caches so the UI reflects the change without a page refresh.
 */
export function useFlagSSE(projectSlug: string, envSlug: string) {
  const queryClient = useQueryClient()

  useEffect(() => {
    const controller = new AbortController()

    async function connect() {
      let token: string | undefined
      try {
        const user = await getUserManager().getUser()
        token = user?.access_token
      } catch {
        return
      }
      if (!token || controller.signal.aborted) return

      const url = `/api/v1/projects/${encodeURIComponent(projectSlug)}/environments/${encodeURIComponent(envSlug)}/flags/stream`

      let res: Response
      try {
        res = await fetch(url, {
          headers: {
            Authorization: `Bearer ${token}`,
            Accept: 'text/event-stream',
          },
          signal: controller.signal,
        })
      } catch {
        return
      }

      if (!res.ok || !res.body) return

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      const onAbort = () => reader.cancel()
      controller.signal.addEventListener('abort', onAbort)

      try {
        while (!controller.signal.aborted) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })

          let boundary: number
          while ((boundary = buffer.indexOf('\n\n')) !== -1) {
            const block = buffer.slice(0, boundary)
            buffer = buffer.slice(boundary + 2)

            for (const line of block.split('\n')) {
              if (!line.startsWith('data:')) continue
              const json = line.slice(5).trim()
              if (!json) continue

              try {
                const event = JSON.parse(json)
                if (event.type === 'flag.state_changed') {
                  void queryClient.invalidateQueries({
                    queryKey: ['flag', projectSlug, envSlug, event.flag_key],
                  })
                  void queryClient.invalidateQueries({
                    queryKey: ['flag-env-state', projectSlug],
                  })
                  void queryClient.invalidateQueries({
                    queryKey: ['flags', projectSlug, envSlug],
                  })
                }
              } catch {
                // Ignore malformed SSE data.
              }
            }
          }
        }
      } finally {
        controller.signal.removeEventListener('abort', onAbort)
        reader.releaseLock()
      }
    }

    void connect()

    return () => controller.abort()
  }, [projectSlug, envSlug, queryClient])
}
