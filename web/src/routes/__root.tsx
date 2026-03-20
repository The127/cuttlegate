import { Outlet, createRootRoute } from '@tanstack/react-router'

export const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFoundPage,
})

function NotFoundPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <h1 className="text-2xl font-semibold text-gray-900">404</h1>
        <p className="mt-2 text-gray-600">Page not found.</p>
        <a href="/" className="mt-4 inline-block text-sm text-blue-600 hover:text-blue-800">
          Return to home
        </a>
      </div>
    </div>
  )
}
