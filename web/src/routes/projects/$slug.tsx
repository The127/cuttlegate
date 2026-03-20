import { createRoute, notFound, Outlet } from '@tanstack/react-router'
import { authenticatedRoute } from '../_authenticated'
import { fetchJSON, APIError } from '../../api'

interface ProjectDetail {
  id: string
  name: string
  slug: string
  created_at: string
}

export const projectRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/projects/$slug',
  loader: async ({ params }) => {
    try {
      return await fetchJSON<ProjectDetail>(`/api/v1/projects/${params.slug}`)
    } catch (err) {
      if (err instanceof APIError && err.status === 404) {
        throw notFound()
      }
      throw err
    }
  },
  component: ProjectPage,
})

function ProjectPage() {
  return <Outlet />
}
