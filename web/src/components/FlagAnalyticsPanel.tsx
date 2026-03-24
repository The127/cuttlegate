import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchJSON } from '../api'

// ---------------------------------------------------------------------------
// Types — derived from evaluation_stats_handler.go bucketsResponse shape
// ---------------------------------------------------------------------------

interface Bucket {
  ts: string
  total: number
  variants: Record<string, number>
}

interface BucketsResponse {
  flag_key: string
  environment: string
  window: string
  bucket_size: string
  buckets: Bucket[]
}

// ---------------------------------------------------------------------------
// Colour palette
// ---------------------------------------------------------------------------

const BOOL_TRUE_COLOR = '#22c55e'  // green-500
const BOOL_FALSE_COLOR = '#f87171' // red-400

// 5-colour design-system palette for multivariate flags
const MV_PALETTE = [
  '#6366f1', // indigo-500
  '#f59e0b', // amber-500
  '#06b6d4', // cyan-500
  '#ec4899', // pink-500
  '#84cc16', // lime-500
]

// ---------------------------------------------------------------------------
// Sparse bucket normalisation
// ---------------------------------------------------------------------------

/**
 * Fill zero-value days for every day in the requested window that has no
 * bucket entry. The API may return sparse buckets (only days with events).
 */
function normalizeBuckets(buckets: Bucket[], windowDays: number): Bucket[] {
  // Build a map keyed by date string YYYY-MM-DD
  const byDate = new Map<string, Bucket>()
  for (const b of buckets) {
    const dateKey = b.ts.slice(0, 10)
    byDate.set(dateKey, b)
  }

  const filled: Bucket[] = []
  const now = new Date()
  for (let i = windowDays - 1; i >= 0; i--) {
    const d = new Date(now)
    d.setUTCDate(d.getUTCDate() - i)
    const dateKey = d.toISOString().slice(0, 10)
    if (byDate.has(dateKey)) {
      filled.push(byDate.get(dateKey)!)
    } else {
      filled.push({ ts: dateKey + 'T00:00:00Z', total: 0, variants: {} })
    }
  }
  return filled
}

// ---------------------------------------------------------------------------
// Helper: collect all variant keys across all buckets
// ---------------------------------------------------------------------------

function collectVariantKeys(buckets: Bucket[]): string[] {
  const seen = new Set<string>()
  for (const b of buckets) {
    for (const k of Object.keys(b.variants)) {
      seen.add(k)
    }
  }
  return Array.from(seen)
}

// ---------------------------------------------------------------------------
// SVG bar chart
// ---------------------------------------------------------------------------

const CHART_CONTENT_WIDTH = 440
const CHART_CONTENT_HEIGHT = 120
const MARGIN_LEFT = 40
const MARGIN_BOTTOM = 20
const CHART_WIDTH = CHART_CONTENT_WIDTH + MARGIN_LEFT
const CHART_HEIGHT = CHART_CONTENT_HEIGHT + MARGIN_BOTTOM
const BAR_GAP = 2

interface BarChartProps {
  buckets: Bucket[]
  flagType: string
  variantKeys: string[]
  ariaLabel: string
}

function formatShortDate(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' })
}

function BarChart({ buckets, flagType, variantKeys, ariaLabel }: BarChartProps) {
  const { t } = useTranslation('flags')
  const isBool = flagType === 'bool'
  const n = buckets.length
  if (n === 0) return null

  const maxTotal = Math.max(...buckets.map((b) => b.total), 1)
  const barWidth = (CHART_CONTENT_WIDTH - BAR_GAP * (n - 1)) / n

  function getSegments(b: Bucket): Array<{ color: string; value: number; label: string }> {
    if (isBool) {
      const trueVal = b.variants['true'] ?? 0
      const falseVal = b.variants['false'] ?? 0
      return [
        { color: BOOL_FALSE_COLOR, value: falseVal, label: 'false' },
        { color: BOOL_TRUE_COLOR, value: trueVal, label: 'true' },
      ]
    }
    // Multivariate: stack all known variant keys
    return variantKeys.map((k, i) => ({
      color: MV_PALETTE[i % MV_PALETTE.length],
      value: b.variants[k] ?? 0,
      label: k,
    }))
  }

  const bars = buckets.map((b, i) => {
    const x = MARGIN_LEFT + i * (barWidth + BAR_GAP)
    const segments = getSegments(b)
    const shapes: React.ReactNode[] = []
    let accum = 0
    for (const seg of segments) {
      if (seg.value <= 0) continue
      const segH = (seg.value / maxTotal) * CHART_CONTENT_HEIGHT
      const y = CHART_CONTENT_HEIGHT - accum - segH
      shapes.push(
        <rect
          key={seg.label}
          x={x}
          y={y}
          width={barWidth}
          height={segH}
          fill={seg.color}
          role="presentation"
        />
      )
      accum += segH
    }
    // Zero-height bar: render a thin placeholder
    if (b.total === 0) {
      shapes.push(
        <rect
          key="zero"
          x={x}
          y={CHART_CONTENT_HEIGHT - 2}
          width={barWidth}
          height={2}
          fill="#1c1f35"
          role="presentation"
        />
      )
    }
    return <g key={i}>{shapes}</g>
  })

  // X-axis date labels: first and last bucket
  const firstDate = formatShortDate(buckets[0].ts)
  const lastDate = formatShortDate(buckets[n - 1].ts)

  return (
    <svg
      width={CHART_WIDTH}
      height={CHART_HEIGHT}
      viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
      aria-label={ariaLabel}
      role="img"
      className="w-full"
      data-testid="analytics-chart"
    >
      {/* Y-axis max count label */}
      <text
        x={MARGIN_LEFT - 4}
        y={10}
        textAnchor="end"
        className="text-[10px] fill-[var(--color-text-muted)]"
        aria-label={t('analytics.y_axis_label', { count: maxTotal })}
      >
        {maxTotal}
      </text>
      {bars}
      {/* X-axis date labels */}
      <text
        x={MARGIN_LEFT}
        y={CHART_CONTENT_HEIGHT + 14}
        textAnchor="start"
        className="text-[10px] fill-[var(--color-text-muted)]"
      >
        {firstDate}
      </text>
      <text
        x={CHART_WIDTH}
        y={CHART_CONTENT_HEIGHT + 14}
        textAnchor="end"
        className="text-[10px] fill-[var(--color-text-muted)]"
      >
        {lastDate}
      </text>
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Legend
// ---------------------------------------------------------------------------

interface LegendProps {
  flagType: string
  variantKeys: string[]
}

function Legend({ flagType, variantKeys }: LegendProps) {
  const { t } = useTranslation('flags')
  if (flagType === 'bool') {
    return (
      <div className="flex gap-4 mt-2">
        <LegendItem color={BOOL_TRUE_COLOR} label={t('analytics.true_label')} />
        <LegendItem color={BOOL_FALSE_COLOR} label={t('analytics.false_label')} />
      </div>
    )
  }
  return (
    <div className="flex flex-wrap gap-3 mt-2">
      {variantKeys.map((k, i) => (
        <LegendItem
          key={k}
          color={MV_PALETTE[i % MV_PALETTE.length]}
          label={t('analytics.variant_label', { key: k })}
        />
      ))}
    </div>
  )
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
      <span
        style={{ backgroundColor: color }}
        className="inline-block w-3 h-3 rounded-sm shrink-0"
        aria-hidden="true"
      />
      {label}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Window selector
// ---------------------------------------------------------------------------

type WindowOption = '7d' | '30d'

interface WindowSelectorProps {
  active: WindowOption
  onChange: (w: WindowOption) => void
}

function WindowSelector({ active, onChange }: WindowSelectorProps) {
  const { t } = useTranslation('flags')
  const options: WindowOption[] = ['7d', '30d']
  const labelKey: Record<WindowOption, string> = {
    '7d': 'analytics.window_7d',
    '30d': 'analytics.window_30d',
  }
  return (
    <div className="flex gap-1" role="group">
      {options.map((w) => (
        <button
          key={w}
          onClick={() => onChange(w)}
          aria-pressed={active === w}
          className={`px-2 py-0.5 text-xs rounded border transition-colors focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-blue-500 ${
            active === w
              ? 'bg-[var(--color-accent)] text-white border-[var(--color-accent)]'
              : 'bg-[var(--color-surface)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:bg-[var(--color-surface)]'
          }`}
        >
          {t(labelKey[w])}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main panel
// ---------------------------------------------------------------------------

export interface FlagAnalyticsPanelProps {
  slug: string
  envSlug: string
  flagKey: string
  flagType: string
}

export function FlagAnalyticsPanel({ slug, envSlug, flagKey, flagType }: FlagAnalyticsPanelProps) {
  const { t } = useTranslation('flags')
  const [selectedWindow, setSelectedWindow] = useState<WindowOption>('7d')
  const windowDays = selectedWindow === '7d' ? 7 : 30

  const { data, isError, isLoading } = useQuery<BucketsResponse>({
    queryKey: ['flag-analytics-buckets', slug, envSlug, flagKey, selectedWindow],
    queryFn: () =>
      fetchJSON<BucketsResponse>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags/${flagKey}/stats/buckets?window=${selectedWindow}&bucket=day`,
      ),
    refetchInterval: 60_000,
  })

  const normalizedBuckets = useMemo(() => {
    if (!data) return []
    return normalizeBuckets(data.buckets, windowDays)
  }, [data, windowDays])

  const variantKeys = useMemo(() => collectVariantKeys(normalizedBuckets), [normalizedBuckets])

  const totalCount = useMemo(
    () => normalizedBuckets.reduce((sum, b) => sum + b.total, 0),
    [normalizedBuckets],
  )

  const isEmpty = !isLoading && !isError && totalCount === 0

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg mt-4">
      <div className="px-5 py-3 border-b border-[var(--color-border)] flex items-center justify-between">
        <h2 className="text-xs font-semibold text-[var(--color-text-secondary)] font-medium">
          {t('analytics.panel_title')}
        </h2>
        <WindowSelector active={selectedWindow} onChange={setSelectedWindow} />
      </div>

      <div className="px-5 py-4">
        {isError ? (
          <p className="text-sm text-[var(--color-status-error)]" role="status">
            {t('analytics.load_error')}
          </p>
        ) : isLoading ? (
          <div className="h-20 flex items-center">
            <div className="h-4 w-48 bg-[var(--color-surface-elevated)] rounded animate-pulse" />
          </div>
        ) : isEmpty ? (
          <p className="text-sm text-[var(--color-text-secondary)]" data-testid="analytics-empty">
            {t('analytics.empty_state')}
          </p>
        ) : (
          <>
            <BarChart
              buckets={normalizedBuckets}
              flagType={flagType}
              variantKeys={variantKeys}
              ariaLabel={t('analytics.chart_aria_label')}
            />
            <Legend flagType={flagType} variantKeys={variantKeys} />
            <p className="mt-3 text-xs text-[var(--color-text-secondary)]">
              {t('analytics.total_evaluations', { count: totalCount, window: selectedWindow })}
            </p>
          </>
        )}
      </div>
    </div>
  )
}
