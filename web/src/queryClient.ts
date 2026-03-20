import { QueryCache, QueryClient } from '@tanstack/react-query'
import { getUserManager } from './auth'
import { APIError } from './api'

export function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error, query) => {
        if (!(error instanceof APIError) || error.status !== 401) return

        // 401 received — attempt silent renewal then re-run the query.
        getUserManager()
          .signinSilent()
          .then(() => {
            // Renewal succeeded: invalidate so TanStack Query re-fetches.
            void queryClient.invalidateQueries({ queryKey: query.queryKey })
          })
          .catch(() => {
            // Renewal failed — redirect to login.
            void getUserManager().signinRedirect({
              state: window.location.pathname + window.location.search,
            })
          })
      },
    }),
    defaultOptions: {
      queries: {
        // Disable automatic retry — our 401 handler above takes over.
        retry: false,
      },
    },
  })
}

// Singleton — created once and shared across the app.
export const queryClient = createQueryClient()
