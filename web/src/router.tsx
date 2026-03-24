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
import { flagEvaluationsRoute } from './routes/projects/$slug.environments.$envSlug.flags.$key.evaluations'
import { environmentsOverviewRoute } from './routes/projects/$slug.environments'
import { segmentListRoute } from './routes/projects/$slug.segments'
import { environmentSettingsRoute } from './routes/projects/$slug.settings.environments'
import { compareRoute } from './routes/projects/$slug.compare'
import { projectSettingsRoute } from './routes/projects/$slug.settings'
import { apiKeyListRoute } from './routes/projects/$slug.api-keys'
import { memberListRoute } from './routes/projects/$slug.members'
import { auditRoute } from './routes/projects/$slug.audit'
import { projectFlagListRoute } from './routes/projects/$slug.flags'

const routeTree = rootRoute.addChildren([
  authenticatedRoute.addChildren([
    indexRoute,
    projectRoute.addChildren([
      projectIndexRoute,
      environmentsOverviewRoute,
      segmentListRoute,
      projectSettingsRoute,
      environmentSettingsRoute,
      compareRoute,
      apiKeyListRoute,
      memberListRoute,
      auditRoute,
      projectFlagListRoute,
      projectEnvRoute.addChildren([flagListRoute, flagDetailRoute, flagRulesRoute, flagEvaluationsRoute]),
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
