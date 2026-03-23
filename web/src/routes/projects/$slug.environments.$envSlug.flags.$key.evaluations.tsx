import { createRoute } from '@tanstack/react-router'
import { useInfiniteQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { flagDetailRoute } from './$slug.environments.$envSlug.flags.$key'
import { fetchJSON, APIError } from '../../api'
import { Button } from '../../components/ui'

interface MatchedRule {
  id: string
  name: string
}

interface EvaluationEvent {
  id: string
  occurred_at: string
  flag_key: string
  user_id: string
  input_context: Record<string, unknown>
  matched_rule: MatchedRule | null
  variant_key: string
  reason: string
}

interface EvaluationListResponse {
  items: EvaluationEvent[]
  next_cursor: string | null
}

const REASON_LABEL_MAP: Record<string, string> = {
  flag_disabled: 'audit.reason_flag_disabled',
  no_match: 'audit.reason_no_match',
  rule_match: 'audit.reason_rule_match',
  override: 'audit.reason_override',
}

const REASON_BADGE_CLASS: Record<string, string> = {
  flag_disabled: 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border-[var(--color-border)]',
  no_match: 'bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800',
  rule_match: 'bg-[rgba(16,217,168,0.08)] text-[var(--color-status-enabled)] border-[var(--color-status-enabled)]',
  override: 'bg-[rgba(79,124,255,0.1)] text-[var(--color-accent)] border-[rgba(79,124,255,0.3)]',
}

export const flagEvaluationsRoute = createRoute({
  getParentRoute: () => flagDetailRoute,
  path: '/evaluations',
  component: FlagEvaluationsPage,
})

function FlagEvaluationsPage() {
  const { t } = useTranslation('flags')
  const { slug, envSlug, key } = flagDetailRoute.useParams()

  const { data, isLoading, error, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery({
      queryKey: ['evaluations', slug, envSlug, key],
      queryFn: ({ pageParam }: { pageParam: string | undefined }) => {
        const url = `/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}/evaluations`
        const full = pageParam ? `${url}?before=${encodeURIComponent(pageParam)}` : url
        return fetchJSON<EvaluationListResponse>(full)
      },
      initialPageParam: undefined as string | undefined,
      getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    })

  if (isLoading) {
    return (
      <div className="px-5 py-4">
        <p className="text-sm text-[var(--color-text-secondary)]">{t('audit.loading')}</p>
      </div>
    )
  }

  if (error) {
    const msg = error instanceof APIError ? error.message : t('audit.load_error')
    return (
      <div className="px-5 py-4">
        <p className="text-sm text-[var(--color-status-error)]">{msg}</p>
      </div>
    )
  }

  const allItems = data?.pages.flatMap((p) => p.items) ?? []

  return (
    <div className="mt-4 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--color-border)]">
        <h2 className="text-xs font-semibold text-[var(--color-text-secondary)] font-medium">
          {t('audit.tab_title')}
        </h2>
      </div>

      {allItems.length === 0 ? (
        <div className="px-5 py-6 text-center">
          <p className="text-sm text-[var(--color-text-secondary)]">{t('audit.empty_state')}</p>
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--color-border)] bg-[var(--color-surface-elevated)]">
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_timestamp')}</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_user')}</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_rule')}</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_variant')}</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_reason')}</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-[var(--color-text-secondary)]">{t('audit.col_context')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--color-border)]">
                {allItems.map((item) => (
                  <EvaluationRow key={item.id} item={item} />
                ))}
              </tbody>
            </table>
          </div>

          {hasNextPage && (
            <div className="px-5 py-3 border-t border-[var(--color-border)] flex justify-center">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => void fetchNextPage()}
                disabled={isFetchingNextPage}
              >
                {t('audit.load_more')}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function EvaluationRow({ item }: { item: EvaluationEvent }) {
  const { t } = useTranslation('flags')
  const [expanded, setExpanded] = useState(false)

  const reasonLabelKey = REASON_LABEL_MAP[item.reason]
  const reasonLabel = reasonLabelKey ? t(reasonLabelKey) : item.reason
  const badgeClass = REASON_BADGE_CLASS[item.reason] ?? 'bg-[var(--color-surface)] text-[var(--color-text-secondary)] border-[var(--color-border)]'

  return (
    <>
      <tr className="hover:bg-[var(--color-surface)]
        <td className="px-4 py-2 text-xs font-mono text-[var(--color-text-secondary)] whitespace-nowrap">
          {new Date(item.occurred_at).toLocaleString()}
        </td>
        <td className="px-4 py-2 text-xs font-mono text-[var(--color-text-primary)]">
          {item.user_id || '—'}
        </td>
        <td className="px-4 py-2 text-xs font-mono text-[var(--color-text-secondary)]">
          {item.matched_rule ? (
            <span title={`${t('audit.rule_id_label')}: ${item.matched_rule.id}`}>
              {item.matched_rule.name || item.matched_rule.id}
            </span>
          ) : (
            <span className="text-[var(--color-text-muted)]">{t('audit.no_rule_matched')}</span>
          )}
        </td>
        <td className="px-4 py-2">
          <span className="font-mono text-xs text-[var(--color-text-primary)] bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-1.5 py-0.5">
            {item.variant_key}
          </span>
        </td>
        <td className="px-4 py-2">
          <span className={`inline-block text-xs font-medium border rounded-full px-2 py-0.5 ${badgeClass}`}>
            {reasonLabel}
          </span>
        </td>
        <td className="px-4 py-2">
          <button
            onClick={() => setExpanded((v) => !v)}
            aria-expanded={expanded}
            aria-label={t('audit.context_expand_aria', { id: item.id })}
            className="text-xs text-[var(--color-accent)] hover:underline focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
          >
            {expanded ? '▲' : '▼'}
          </button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={6} className="px-4 pb-3 pt-0">
            <pre className="text-xs font-mono bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded p-3 overflow-x-auto whitespace-pre-wrap break-all">
              {JSON.stringify(item.input_context, null, 2)}
            </pre>
          </td>
        </tr>
      )}
    </>
  )
}
