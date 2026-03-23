import { createRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectEnvRoute } from './$slug.environments.$envSlug'
import { fetchJSON, postJSON, patchJSON, deleteRequest, APIError } from '../../api'
import { Button, Input, Select, SelectItem } from '../../components/ui'

// ── Types ──────────────────────────────────────────────────────────────────

interface Condition {
  attribute: string
  operator: string
  values: string[]
}

interface Rule {
  id: string
  name: string
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

const OPERATOR_KEYS: Record<string, string> = {
  eq: 'operators.eq',
  neq: 'operators.neq',
  lt: 'operators.lt',
  lte: 'operators.lte',
  gt: 'operators.gt',
  gte: 'operators.gte',
  in: 'operators.in',
  not_in: 'operators.not_in',
  contains: 'operators.contains',
  not_contains: 'operators.not_contains',
  in_segment: 'operators.in_segment',
  not_in_segment: 'operators.not_in_segment',
}

const SEGMENT_OPERATORS = new Set(['in_segment', 'not_in_segment'])

function isSegmentOperator(op: string): boolean {
  return SEGMENT_OPERATORS.has(op)
}

// ── Page ───────────────────────────────────────────────────────────────────

function RulesPage() {
  const { t } = useTranslation('rules')
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
        <p className="text-sm text-red-600 dark:text-red-400">
          {is404 ? t('not_found') : t('error')}
        </p>
        <button
          onClick={() => void rulesQuery.refetch()}
          className="mt-2 text-sm text-[var(--color-accent)] underline hover:no-underline"
        >
          {t('actions.retry', { ns: 'common' })}
        </button>
      </div>
    )
  }

  function handleAdd(
    rule: Omit<Rule, 'id' | 'createdAt'>,
    opts?: { onSuccess?: () => void; onError?: (err: Error) => void },
  ) {
    createMutation.mutate(rule, {
      onSuccess: () => { setAddingNew(false); opts?.onSuccess?.() },
      onError: opts?.onError,
    })
  }

  function handleUpdate(
    rule: Omit<Rule, 'createdAt'>,
    opts?: { onSuccess?: () => void; onError?: (err: Error) => void },
  ) {
    updateMutation.mutate(rule, opts)
  }

  function handleDelete(id: string) {
    deleteMutation.mutate(id)
  }

  return (
    <div className="p-6 max-w-2xl">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-700 dark:text-gray-300">{t('title')}</h2>
        {!addingNew && (
          <Button onClick={() => setAddingNew(true)}>{t('add_rule')}</Button>
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
              onSave={(r, opts) => handleUpdate(r, opts)}
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
              onSave={(r, opts) => handleAdd(r, opts)}
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
  const { t } = useTranslation('rules')
  return (
    <div className="border border-dashed border-gray-200 dark:border-gray-600 rounded-lg px-6 py-10 text-center">
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t('empty_state')}
      </p>
      <Button onClick={onAdd} className="mt-3">{t('add_rule')}</Button>
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
  onSave: (rule: Omit<Rule, 'createdAt'>, opts?: { onSuccess?: () => void; onError?: (err: Error) => void }) => void
  onDelete: (id: string) => void
  isSaving: boolean
  isDeleting: boolean
}) {
  const { t } = useTranslation('rules')
  const [editing, setEditing] = useState(false)
  const [pendingDelete, setPendingDelete] = useState(false)
  const [draft, setDraft] = useState<Omit<Rule, 'id' | 'createdAt'>>({
    name: rule.name,
    priority: rule.priority,
    conditions: rule.conditions,
    variantKey: rule.variantKey,
    enabled: rule.enabled,
  })
  const [saveError, setSaveError] = useState<string | null>(null)

  function startEdit() {
    setDraft({
      name: rule.name,
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
    onSave({ id: rule.id, ...draft }, {
      onSuccess: () => setEditing(false),
      onError: (err) => setSaveError(err instanceof APIError ? err.message : t('save_failed')),
    })
  }

  function confirmDelete() {
    onDelete(rule.id)
    setPendingDelete(false)
  }

  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
      {/* Summary row */}
      <div className="flex items-start gap-3 px-4 py-3">
        <span
          className="mt-0.5 shrink-0 w-6 h-6 flex items-center justify-center rounded-full bg-gray-100 dark:bg-gray-700 text-xs font-mono text-gray-600 dark:text-gray-300"
          title="Priority"
        >
          {rule.priority}
        </span>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-800 dark:text-gray-200 mb-1">
            {rule.name ? rule.name : t('fallback_name', { priority: rule.priority })}
          </p>
          {rule.conditions.length === 0 ? (
            <p className="text-sm text-gray-400 dark:text-gray-500 italic">{t('no_conditions_display')}</p>
          ) : (
            <ul className="space-y-0.5">
              {rule.conditions.map((c, i) => (
                <li key={i} className="text-sm text-gray-700 dark:text-gray-300">
                  <span className="font-mono text-gray-800 dark:text-gray-200">{c.attribute}</span>{' '}
                  <span className="text-gray-500 dark:text-gray-400">{OPERATOR_KEYS[c.operator] ? t(OPERATOR_KEYS[c.operator]) : c.operator}</span>{' '}
                  {isSegmentOperator(c.operator) ? (
                    <span className="font-mono text-gray-800 dark:text-gray-200">{c.values[0]}</span>
                  ) : (
                    <span className="font-mono text-gray-800 dark:text-gray-200">{c.values.join(', ')}</span>
                  )}
                </li>
              ))}
            </ul>
          )}
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            → <span className="font-mono">{rule.variantKey}</span>
          </p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {!editing && !pendingDelete && (
            <>
              <Button variant="secondary" size="sm" onClick={startEdit} aria-label={t('actions.edit', { ns: 'common' })}>
                {t('actions.edit', { ns: 'common' })}
              </Button>
              <Button
                variant="danger-outline"
                size="sm"
                onClick={() => setPendingDelete(true)}
                disabled={isDeleting}
                aria-label={t('actions.delete', { ns: 'common' })}
              >
                {isDeleting ? '…' : '✕'}
              </Button>
            </>
          )}
          {pendingDelete && (
            <span className="flex items-center gap-2 text-sm">
              <span className="text-gray-600 dark:text-gray-400">{t('delete_confirm')}</span>
              <Button variant="danger-outline" size="sm" onClick={confirmDelete}>
                {t('delete_yes')}
              </Button>
              <Button variant="secondary" size="sm" onClick={() => setPendingDelete(false)}>
                {t('delete_no')}
              </Button>
            </span>
          )}
        </div>
      </div>

      {/* Inline editor */}
      {editing && (
        <div className="border-t border-gray-100 dark:border-gray-700 px-4 py-4 space-y-4">
          <RuleEditor
            draft={draft}
            segments={segments}
            variants={variants}
            onChange={setDraft}
          />
          <div className="flex items-center gap-3">
            <Button onClick={save} disabled={isSaving}>
              {isSaving ? t('states.saving', { ns: 'common' }) : t('actions.save', { ns: 'common' })}
            </Button>
            <Button variant="secondary" onClick={cancelEdit} disabled={isSaving}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            {saveError && <p className="text-xs text-red-600 dark:text-red-400">{saveError}</p>}
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
  onSave: (rule: Omit<Rule, 'id' | 'createdAt'>, opts?: { onSuccess?: () => void; onError?: (err: Error) => void }) => void
  onCancel: () => void
  isSaving: boolean
}) {
  const { t } = useTranslation('rules')
  const [draft, setDraft] = useState<Omit<Rule, 'id' | 'createdAt'>>({
    name: '',
    priority: nextPriority,
    conditions: [],
    variantKey: variants[0]?.key ?? '',
    enabled: true,
  })
  const [saveError, setSaveError] = useState<string | null>(null)

  return (
    <div className="bg-white dark:bg-gray-800 border border-blue-200 dark:border-blue-800 rounded-lg px-4 py-4 space-y-4">
      <p className="text-xs font-medium text-gray-500 dark:text-gray-400">{t('new_rule')}</p>
      <RuleEditor draft={draft} segments={segments} variants={variants} onChange={setDraft} />
      <div className="flex items-center gap-3">
        <Button
          onClick={() => {
            setSaveError(null)
            onSave(draft, {
              onError: (err) => setSaveError(err instanceof APIError ? err.message : t('save_failed')),
            })
          }}
          disabled={isSaving}
        >
          {isSaving ? t('states.saving', { ns: 'common' }) : t('actions.save', { ns: 'common' })}
        </Button>
        <Button variant="secondary" onClick={onCancel} disabled={isSaving}>
          {t('actions.cancel', { ns: 'common' })}
        </Button>
        {saveError && <p className="text-xs text-red-600">{saveError}</p>}
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
  const { t } = useTranslation('rules')

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
      {/* Name */}
      <div>
        <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{t('name_label')}</label>
        <Input
          type="text"
          value={draft.name}
          onChange={(e) => onChange({ ...draft, name: e.target.value })}
          placeholder={t('name_placeholder')}
          aria-label={t('name_label')}
          className="py-1.5 px-2"
        />
      </div>

      {/* Conditions */}
      <div>
        <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">{t('conditions_label')}</p>
        {draft.conditions.length === 0 && (
          <p className="text-xs text-gray-400 dark:text-gray-500 italic mb-2">{t('no_conditions_editor')}</p>
        )}
        <div className="space-y-2">
          {draft.conditions.map((c, i) => (
            <div key={i} className="flex items-start gap-2">
              {/* Attribute */}
              <Input
                type="text"
                value={c.attribute}
                onChange={(e) => updateCondition(i, { attribute: e.target.value })}
                placeholder={t('attribute_placeholder')}
                aria-label={t('condition_attribute_aria', { n: i + 1 })}
                className="w-32 font-mono py-1.5 px-2"
              />
              {/* Operator */}
              <Select
                value={c.operator}
                onValueChange={(val) => updateCondition(i, { operator: val })}
                aria-label={t('condition_operator_aria', { n: i + 1 })}
              >
                {Object.entries(OPERATOR_KEYS).map(([val, key]) => (
                  <SelectItem key={val} value={val}>
                    {t(key)}
                  </SelectItem>
                ))}
              </Select>
              {/* Value — segment dropdown or text input */}
              {isSegmentOperator(c.operator) ? (
                <Select
                  value={c.values[0] ?? ''}
                  onValueChange={(val) => updateCondition(i, { values: [val] })}
                  aria-label={t('condition_segment_aria', { n: i + 1 })}
                  placeholder={t('select_segment')}
                  className="flex-1"
                >
                  {segments.map((s) => (
                    <SelectItem key={s.slug} value={s.slug}>
                      {s.name}
                    </SelectItem>
                  ))}
                </Select>
              ) : (
                <Input
                  type="text"
                  value={c.values.join(', ')}
                  onChange={(e) =>
                    updateCondition(i, {
                      values: e.target.value.split(',').map((v) => v.trim()).filter(Boolean),
                    })
                  }
                  placeholder={t('value_placeholder')}
                  aria-label={t('condition_value_aria', { n: i + 1 })}
                  className="flex-1 font-mono py-1.5 px-2"
                />
              )}
              <button
                onClick={() => removeCondition(i)}
                aria-label={t('condition_remove_aria', { n: i + 1 })}
                className="mt-1 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
              >
                ✕
              </button>
            </div>
          ))}
        </div>
        <button
          onClick={addCondition}
          className="mt-2 text-xs text-[var(--color-accent)] hover:underline focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
        >
          {t('add_condition')}
        </button>
      </div>

      {/* Variant */}
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">{t('serve_variant_label')}</label>
        <Select
          value={draft.variantKey}
          onValueChange={(val) => onChange({ ...draft, variantKey: val })}
          aria-label={t('serve_variant_aria')}
        >
          {variants.map((v) => (
            <SelectItem key={v.key} value={v.key}>
              {v.key}
            </SelectItem>
          ))}
        </Select>
      </div>

      {/* Priority */}
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">
          {t('priority_label')} <span className="text-gray-400 font-normal">{t('priority_hint')}</span>
        </label>
        <Input
          type="number"
          min={0}
          value={draft.priority}
          onChange={(e) => onChange({ ...draft, priority: parseInt(e.target.value, 10) || 0 })}
          className="w-24 py-1.5 px-2"
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
        <div className="h-4 w-32 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        <div className="h-8 w-20 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
      </div>
      {[1, 2, 3].map((i) => (
        <div key={i} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg px-4 py-3">
          <div className="flex items-center gap-3">
            <div className="w-6 h-6 bg-gray-100 dark:bg-gray-700 rounded-full animate-pulse" />
            <div className="flex-1 space-y-1.5">
              <div className="h-3 w-48 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
              <div className="h-3 w-24 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
