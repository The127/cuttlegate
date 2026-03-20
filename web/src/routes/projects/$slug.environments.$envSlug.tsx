import { createRoute, Outlet } from '@tanstack/react-router'
import { projectRoute } from './$slug'

export const projectEnvRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/environments/$envSlug',
  component: ProjectEnvPage,
})

function ProjectEnvPage() {
  return <Outlet />
}
