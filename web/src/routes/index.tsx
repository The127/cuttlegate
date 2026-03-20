import { createRoute } from '@tanstack/react-router'
import { authenticatedRoute } from './_authenticated'

export const indexRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/',
  component: IndexPage,
})

function IndexPage() {
  return (
    <div className="p-8">
      <h1 className="text-2xl font-semibold text-gray-900">Cuttlegate</h1>
      <p className="mt-2 text-gray-600">Select a project to get started.</p>
    </div>
  )
}
