import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { fetchJSON, postJSON, patchJSON, deleteRequest, APIError } from '../../api'

// ── Types ──────────────────────────────────────────────────────────────────

interface Condition {
  attribute: string
  operator: string
  values: string[]
}

interface Rule {
  id: string
  priority: number
  conditions: Condition[]
  variantKey: string
  enabled: boolean
  createdAt: string
}

interface Segment {
  id: string
  slug: string
  name: string
}

// ── Route ──────────────────────────────────────────────────────────────────

export const flagRulesRoute = createRoute({
  getParentRoute: () => projectEnvRoute,
  path: '/flags/$key/rules',
  component: RulesPage,
})

// ── Operator helpers ───────────────────────────────────────────────────────

const OPERATOR_LABELS: Record<string, string> = {
  eq: 'equals',
  neq: 'not equals',
  lt: 'less than',
  lte: 'less than or equal',
  gt: 'greater than',
  gte: 'greater than or equal',
  in: 'in',
  not_in: 'not in',
  contains: 'contains',
  not_contains: 'does not contain',
  in_segment: 'in segment',
  not_in_segment: 'not in segment',
}

const SEGMENT_OPERATORS = new Set(['in_segment', 'not_in_segment'])

function isSegmentOperator(op: string): boolean {
  return SEGMENT_OPERATORS.has(op)
}

// ── Page ───────────────────────────────────────────────────────────────────

function RulesPage() {
  const { slug, envSlug, key } = flagRulesRoute.useParams()
  const queryClient = useQueryClient()
  const rulesKey = ['rules', slug, envSlug, key]
  const segmentsKey = ['segments', slug]

  const rulesQuery = useQuery({
    queryKey: rulesKey,
    queryFn: () =>
      fetchJSON<{ rules: Rule[] }>(
        `/api/v1/projects/${slug}/flags/${key}/environments/${envSlug}/rules`,
      ).then((r) => r.rules),
  })

  const segmentsQuery = useQuery({
    queryKey: segmentsKey,
    queryFn: () =>
      fetchJSON<{ segments: Segment[] }>(`/api/v1/projects/${slug}/segments`).then(
        (r) => r.segments,
      ),
  })

  // Flag variants needed for the variant dropdown
  const flagQuery = useQuery({
    queryKey: ['flag', slug, envSlug, key],
    queryFn: () =>
      fetchJSON<{ variants: { key: string; name: string }[] }>(
        `/api/v1/projects/${slug}/environments/${envSlug}/flags/${key}`,
      ),
  })

  const createMutation = useMutation({
    mutationFn: (body: Omit<Rule, 'id' | 'createdAt'>) =>
      postJSON<Rule>(
        `/api/v1/projects/${slug}/flags/${key}/environments/${envSlug}/rules`,
        body,
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: rulesKey }),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: Omit<Rule, 'createdAt'>) =>
      patchJSON<Rule>(
        `/api/v1/projects/${slug}/flags/${key}/environments/${envSlug}/rules/${id}`,
        body,
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: rulesKey }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      deleteRequest(
        `/api/v1/projects/${slug}/flags/${key}/environments/${envSlug}/rules/${id}`,
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: rulesKey }),
  })

  const [addingNew, setAddingNew] = useState(false)

  const variants = flagQuery.data?.variants ?? []
  const segments = segmentsQuery.data ?? []
  const rules = rulesQuery.data ?? []

  if (rulesQuery.isLoading) return <RulesSkeleton />

  if (rulesQuery.isError) {
    const is404 = rulesQuery.error instanceof APIError && rulesQuery.error.status === 404
    return (
      <div className="p-6">
        <p className="text-sm text-red-600">
          {is404 ? 'Rules not found.' : 'Failed to load targeting rules.'}
        </p>
        <button
          onClick={() => void rulesQuery.refetch()}
          className="mt-2 text-sm text-blue-600 underline hover:no-underline"
        >
          Retry
        </button>
      </div>
    )
  }

  function handleAdd(rule: Omit<Rule, 'id' | 'createdAt'>) {
    createMutation.mutate(rule, { onSuccess: () => setAddingNew(false) })
  }

  function handleUpdate(rule: Omit<Rule, 'createdAt'>) {
    updateMutation.mutate(rule)
  }

  function handleDelete(id: string) {
    deleteMutation.mutate(id)
  }

  return (
    <div className="p-6 max-w-2xl">
      <div className="mb-4">
        <Link
          to="/projects/$slug/environments/$envSlug/flags/$key"
          params={{ slug, envSlug, key }}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          ← Flag detail
        </Link>
      </div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-700">Targeting Rules</h2>
        {!addingNew && (
          <button
            onClick={() => setAddingNew(true)}
            className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Add rule
          </button>
        )}
      </div>

      {rules.length === 0 && !addingNew ? (
        <EmptyState onAdd={() => setAddingNew(true)} />
      ) : (
        <div className="space-y-3">
          {rules.map((rule) => (
            <RuleRow
              key={rule.id}
              rule={rule}
              variants={variants}
              segments={segments}
              onSave={handleUpdate}
              onDelete={handleDelete}
              isSaving={updateMutation.isPending && updateMutation.variables?.id === rule.id}
              isDeleting={deleteMutation.isPending && deleteMutation.variables === rule.id}
            />
          ))}

          {addingNew && (
            <NewRuleRow
              variants={variants}
              segments={segments}
              nextPriority={rules.length > 0 ? Math.max(...rules.map((r) => r.priority)) + 1 : 1}
              onSave={handleAdd}
              onCancel={() => setAddingNew(false)}
              isSaving={createMutation.isPending}
            />
          )}
        </div>
      )}
    </div>
  )
}

// ── Empty state ────────────────────────────────────────────────────────────

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="border border-dashed border-gray-200 rounded-lg px-6 py-10 text-center">
      <p className="text-sm text-gray-500">
        No targeting rules. Add a rule to start targeting specific users.
      </p>
      <button
        onClick={onAdd}
        className="mt-3 px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        Add rule
      </button>
    </div>
  )
}

// ── Existing rule row ──────────────────────────────────────────────────────

function RuleRow({
  rule,
  variants,
  segments,
  onSave,
  onDelete,
  isSaving,
  isDeleting,
}: {
  rule: Rule
  variants: { key: string; name: string }[]
  segments: Segment[]
  onSave: (rule: Omit<Rule, 'createdAt'>) => void
  onDelete: (id: string) => void
  isSaving: boolean
  isDeleting: boolean
}) {
  const [editing, setEditing] = useState(false)
  const [pendingDelete, setPendingDelete] = useState(false)
  const [draft, setDraft] = useState<Omit<Rule, 'id' | 'createdAt'>>({
    priority: rule.priority,
    conditions: rule.conditions,
    variantKey: rule.variantKey,
    enabled: rule.enabled,
  })
  const [saveError, setSaveError] = useState<string | null>(null)

  function startEdit() {
    setDraft({
      priority: rule.priority,
      conditions: rule.conditions,
      variantKey: rule.variantKey,
      enabled: rule.enabled,
    })
    setSaveError(null)
    setEditing(true)
  }

  function cancelEdit() {
    setSaveError(null)
    setEditing(false)
  }

  function save() {
    setSaveError(null)
    onSave(
      { id: rule.id, ...draft },
    )
    // optimistically close — parent invalidates cache; error surfaced via mutation
    setEditing(false)
  }

  function confirmDelete() {
    onDelete(rule.id)
    setPendingDelete(false)
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      {/* Summary row */}
      <div className="flex items-start gap-3 px-4 py-3">
        <span
          className="mt-0.5 shrink-0 w-6 h-6 flex items-center justify-center rounded-full bg-gray-100 text-xs font-mono text-gray-600"
          title="Priority"
        >
          {rule.priority}
        </span>
        <div className="flex-1 min-w-0">
          {rule.conditions.length === 0 ? (
            <p className="text-sm text-gray-400 italic">No conditions — matches all users</p>
          ) : (
            <ul className="space-y-0.5">
              {rule.conditions.map((c, i) => (
                <li key={i} className="text-sm text-gray-700">
                  <span className="font-mono text-gray-800">{c.attribute}</span>{' '}
                  <span className="text-gray-500">{OPERATOR_LABELS[c.operator] ?? c.operator}</span>{' '}
                  {isSegmentOperator(c.operator) ? (
                    <span className="font-mono text-gray-800">{c.values[0]}</span>
                  ) : (
                    <span className="font-mono text-gray-800">{c.values.join(', ')}</span>
                  )}
                </li>
              ))}
            </ul>
          )}
          <p className="mt-1 text-xs text-gray-500">
            → <span className="font-mono">{rule.variantKey}</span>
          </p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {!editing && !pendingDelete && (
            <>
              <button
                onClick={startEdit}
                className="px-2.5 py-1 text-sm text-gray-600 border border-gray-200 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
                aria-label="Edit rule"
              >
                Edit
              </button>
              <button
                onClick={() => setPendingDelete(true)}
                disabled={isDeleting}
                className="px-2.5 py-1 text-sm text-red-600 border border-red-200 rounded hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50"
                aria-label="Delete rule"
              >
                {isDeleting ? '…' : '✕'}
              </button>
            </>
          )}
          {pendingDelete && (
            <span className="flex items-center gap-2 text-sm">
              <span className="text-gray-600">Delete?</span>
              <button
                onClick={confirmDelete}
                className="px-2 py-0.5 text-sm font-medium text-red-600 border border-red-300 rounded hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500"
              >
                Yes
              </button>
              <button
                onClick={() => setPendingDelete(false)}
                className="px-2 py-0.5 text-sm text-gray-600 border border-gray-200 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
              >
                No
              </button>
            </span>
          )}
        </div>
      </div>

      {/* Inline editor */}
      {editing && (
        <div className="border-t border-gray-100 px-4 py-4 space-y-4">
          <RuleEditor
            draft={draft}
            segments={segments}
            variants={variants}
            onChange={setDraft}
          />
          <div className="flex items-center gap-3">
            <button
              onClick={save}
              disabled={isSaving}
              className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {isSaving ? 'Saving…' : 'Save'}
            </button>
            <button
              onClick={cancelEdit}
              disabled={isSaving}
              className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
            >
              Cancel
            </button>
            {saveError && <p className="text-xs text-red-600">{saveError}</p>}
          </div>
        </div>
      )}
    </div>
  )
}

// ── New rule row ───────────────────────────────────────────────────────────

function NewRuleRow({
  variants,
  segments,
  nextPriority,
  onSave,
  onCancel,
  isSaving,
}: {
  variants: { key: string; name: string }[]
  segments: Segment[]
  nextPriority: number
  onSave: (rule: Omit<Rule, 'id' | 'createdAt'>) => void
  onCancel: () => void
  isSaving: boolean
}) {
  const [draft, setDraft] = useState<Omit<Rule, 'id' | 'createdAt'>>({
    priority: nextPriority,
    conditions: [],
    variantKey: variants[0]?.key ?? '',
    enabled: true,
  })

  return (
    <div className="bg-white border border-blue-200 rounded-lg px-4 py-4 space-y-4">
      <p className="text-xs font-medium text-gray-500">New rule</p>
      <RuleEditor draft={draft} segments={segments} variants={variants} onChange={setDraft} />
      <div className="flex items-center gap-3">
        <button
          onClick={() => onSave(draft)}
          disabled={isSaving}
          className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {isSaving ? 'Saving…' : 'Save'}
        </button>
        <button
          onClick={onCancel}
          disabled={isSaving}
          className="px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}

// ── Rule editor form (shared by new and edit) ──────────────────────────────

function RuleEditor({
  draft,
  segments,
  variants,
  onChange,
}: {
  draft: Omit<Rule, 'id' | 'createdAt'>
  segments: Segment[]
  variants: { key: string; name: string }[]
  onChange: (draft: Omit<Rule, 'id' | 'createdAt'>) => void
}) {
  function updateCondition(i: number, patch: Partial<Condition>) {
    const updated = draft.conditions.map((c, idx) => {
      if (idx !== i) return c
      const next = { ...c, ...patch }
      // Reset values when switching to/from segment operators
      if (patch.operator !== undefined && patch.operator !== c.operator) {
        next.values = []
      }
      return next
    })
    onChange({ ...draft, conditions: updated })
  }

  function addCondition() {
    onChange({
      ...draft,
      conditions: [...draft.conditions, { attribute: '', operator: 'eq', values: [''] }],
    })
  }

  function removeCondition(i: number) {
    onChange({ ...draft, conditions: draft.conditions.filter((_, idx) => idx !== i) })
  }

  return (
    <div className="space-y-3">
      {/* Conditions */}
      <div>
        <p className="text-xs font-medium text-gray-500 mb-2">Conditions</p>
        {draft.conditions.length === 0 && (
          <p className="text-xs text-gray-400 italic mb-2">No conditions — rule matches all users.</p>
        )}
        <div className="space-y-2">
          {draft.conditions.map((c, i) => (
            <div key={i} className="flex items-start gap-2">
              {/* Attribute */}
              <input
                type="text"
                value={c.attribute}
                onChange={(e) => updateCondition(i, { attribute: e.target.value })}
                placeholder="attribute"
                aria-label={`Condition ${i + 1} attribute`}
                className="w-32 text-sm font-mono border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              {/* Operator */}
              <select
                value={c.operator}
                onChange={(e) => updateCondition(i, { operator: e.target.value })}
                aria-label={`Condition ${i + 1} operator`}
                className="text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {Object.entries(OPERATOR_LABELS).map(([val, label]) => (
                  <option key={val} value={val}>
                    {label}
                  </option>
                ))}
              </select>
              {/* Value — segment dropdown or text input */}
              {isSegmentOperator(c.operator) ? (
                <select
                  value={c.values[0] ?? ''}
                  onChange={(e) => updateCondition(i, { values: [e.target.value] })}
                  aria-label={`Condition ${i + 1} segment`}
                  className="flex-1 text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="" disabled>
                    Select segment…
                  </option>
                  {segments.map((s) => (
                    <option key={s.slug} value={s.slug}>
                      {s.name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  type="text"
                  value={c.values.join(', ')}
                  onChange={(e) =>
                    updateCondition(i, {
                      values: e.target.value.split(',').map((v) => v.trim()).filter(Boolean),
                    })
                  }
                  placeholder="value(s), comma-separated"
                  aria-label={`Condition ${i + 1} value`}
                  className="flex-1 text-sm font-mono border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              )}
              <button
                onClick={() => removeCondition(i)}
                aria-label={`Remove condition ${i + 1}`}
                className="mt-1 text-gray-400 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
              >
                ✕
              </button>
            </div>
          ))}
        </div>
        <button
          onClick={addCondition}
          className="mt-2 text-xs text-blue-600 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
        >
          + Add condition
        </button>
      </div>

      {/* Variant */}
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Serve variant</label>
        <select
          value={draft.variantKey}
          onChange={(e) => onChange({ ...draft, variantKey: e.target.value })}
          className="text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {variants.map((v) => (
            <option key={v.key} value={v.key}>
              {v.key}
            </option>
          ))}
        </select>
      </div>

      {/* Priority */}
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">
          Priority <span className="text-gray-400 font-normal">(lower = evaluated first)</span>
        </label>
        <input
          type="number"
          min={0}
          value={draft.priority}
          onChange={(e) => onChange({ ...draft, priority: parseInt(e.target.value, 10) || 0 })}
          className="w-24 text-sm border border-gray-300 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
    </div>
  )
}

// ── Skeleton ───────────────────────────────────────────────────────────────

function RulesSkeleton() {
  return (
    <div className="p-6 max-w-2xl space-y-3">
      <div className="flex items-center justify-between mb-4">
        <div className="h-4 w-32 bg-gray-100 rounded animate-pulse" />
        <div className="h-8 w-20 bg-gray-100 rounded animate-pulse" />
      </div>
      {[1, 2, 3].map((i) => (
        <div key={i} className="bg-white border border-gray-200 rounded-lg px-4 py-3">
          <div className="flex items-center gap-3">
            <div className="w-6 h-6 bg-gray-100 rounded-full animate-pulse" />
            <div className="flex-1 space-y-1.5">
              <div className="h-3 w-48 bg-gray-100 rounded animate-pulse" />
              <div className="h-3 w-24 bg-gray-100 rounded animate-pulse" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
