import { Fragment } from 'react'
import { Link, useMatches } from '@tanstack/react-router'

interface Crumb {
  label: string
  to?: string
}

function useBreadcrumbs(): Crumb[] {
  const matches = useMatches()
  const crumbs: Crumb[] = [{ label: 'Home', to: '/' }]

  for (const match of matches) {
    const routeId = match.routeId

    if (routeId === '/_authenticated/projects/$slug') {
      const params = match.params as { slug: string }
      const loaderData = match.loaderData as { name: string } | undefined
      crumbs.push({
        label: loaderData?.name ?? params.slug,
        to: `/projects/${params.slug}`,
      })
    } else if (routeId === '/_authenticated/projects/$slug/segments') {
      const params = match.params as { slug: string }
      crumbs.push({
        label: 'Segments',
        to: `/projects/${params.slug}/segments`,
      })
    } else if (routeId === '/_authenticated/projects/$slug/environments/$envSlug') {
      const params = match.params as { slug: string; envSlug: string }
      crumbs.push({
        label: params.envSlug,
        to: `/projects/${params.slug}/environments/${params.envSlug}/flags`,
      })
    } else if (routeId === '/_authenticated/projects/$slug/environments/$envSlug/flags/$key') {
      const params = match.params as { slug: string; envSlug: string; key: string }
      crumbs.push({
        label: params.key,
        to: `/projects/${params.slug}/environments/${params.envSlug}/flags/${params.key}`,
      })
    } else if (routeId === '/_authenticated/projects/$slug/environments/$envSlug/flags/$key/rules') {
      crumbs.push({ label: 'Rules' })
    }
  }

  // Last crumb is the current page — strip its link
  if (crumbs.length > 1) {
    const last = crumbs[crumbs.length - 1]
    crumbs[crumbs.length - 1] = { label: last.label }
  }

  return crumbs
}

export function Breadcrumbs() {
  const crumbs = useBreadcrumbs()

  if (crumbs.length <= 1) return null

  // When >3 crumbs, collapse middle ones on narrow viewports
  const shouldCollapse = crumbs.length > 3

  return (
    <nav aria-label="Breadcrumb" className="px-4 py-2 bg-gray-50 border-b border-gray-100">
      <ol className="flex items-center gap-1.5 text-sm text-gray-500 min-w-0">
        {crumbs.map((crumb, i) => {
          const isFirst = i === 0
          const isLast = i === crumbs.length - 1
          // Collapsible on narrow: middle segments not adjacent to last
          const isCollapsible = shouldCollapse && !isFirst && !isLast && i < crumbs.length - 2

          return (
            <Fragment key={crumb.to ?? crumb.label}>
              {/* Ellipsis after first crumb — only visible on narrow viewports */}
              {i === 1 && shouldCollapse && (
                <li className="flex items-center gap-1.5 sm:hidden" aria-hidden="true">
                  <Separator />
                  <span className="text-gray-400">…</span>
                </li>
              )}
              <li
                className={`items-center gap-1.5 min-w-0 ${isCollapsible ? 'hidden sm:flex' : 'flex'}`}
              >
                {!isFirst && <Separator />}
                {crumb.to ? (
                  <Link
                    to={crumb.to}
                    className="truncate hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
                  >
                    {crumb.label}
                  </Link>
                ) : (
                  <span className="truncate text-gray-900 font-medium">{crumb.label}</span>
                )}
              </li>
            </Fragment>
          )
        })}
      </ol>
    </nav>
  )
}

function Separator() {
  return <span className="text-gray-300 shrink-0" aria-hidden="true">/</span>
}
