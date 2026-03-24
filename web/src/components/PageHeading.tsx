import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export interface Ancestor {
  label: string
  to: string
}

interface PageHeadingProps {
  ancestors: Ancestor[]
  current: string
}

export function PageHeading({ ancestors, current }: PageHeadingProps) {
  const { t } = useTranslation('common')

  return (
    <nav aria-label="Page context" className="mb-4 flex items-center gap-1.5 text-sm flex-wrap">
      {ancestors.map((ancestor, i) => (
        <span key={i} className="flex items-center gap-1.5">
          <Link
            to={ancestor.to}
            className="font-mono text-[var(--color-accent)] hover:underline focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
          >
            {ancestor.label}
          </Link>
          <span className="text-[var(--color-text-muted)]" aria-hidden="true">
            {t('nav.heading_separator')}
          </span>
        </span>
      ))}
      <span className="font-mono text-[var(--color-text-primary)] font-medium">
        {current}
      </span>
    </nav>
  )
}
