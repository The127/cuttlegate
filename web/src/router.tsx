import { createRouter } from '@tanstack/react-router'
import { rootRoute } from './routes/__root'
import { authenticatedRoute } from './routes/_authenticated'
import { indexRoute } from './routes/index'
import { callbackRoute } from './routes/auth/callback'
import { projectRoute } from './routes/projects/$slug'
import { projectIndexRoute } from './routes/projects/$slug.index'
import { projectEnvRoute } from './routes/projects/$slug.environments.$envSlug'
import { flagListRoute } from './routes/projects/$slug.environments.$envSlug.flags'
import { flagDetailRoute } from './routes/projects/$slug.environments.$envSlug.flags.$key'
import { flagRulesRoute } from './routes/projects/$slug.environments.$envSlug.flags.$key.rules'
import { segmentListRoute } from './routes/projects/$slug.segments'
import { memberListRoute } from './routes/projects/$slug.members'

const routeTree = rootRoute.addChildren([
  authenticatedRoute.addChildren([
    indexRoute,
    projectRoute.addChildren([
      projectIndexRoute,
      segmentListRoute,
      memberListRoute,
      projectEnvRoute.addChildren([flagListRoute, flagDetailRoute, flagRulesRoute]),
    ]),
  ]),
  callbackRoute,
])

export function createAppRouter() {
  return createRouter({ routeTree })
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>
  }
}
