import { createRoute, Link } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { projectRoute } from './$slug'
import { fetchJSON, postJSON, APIError } from '../../api'
import { formatRelativeDate } from '../../utils/date'
import { Button, Input, Label, Select, SelectItem, CopyableCode } from '../../components/ui'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogCloseButton,
} from '../../components/ui/Dialog'

interface Environment {
  id: string
  name: string
  slug: string
}

interface EnvFlag {
  id: string
  key: string
  name: string
  enabled: boolean
}

interface ProjectFlag {
  id: string
  key: string
  name: string
  created_at: string
}

export const projectIndexRoute = createRoute({
  getParentRoute: () => projectRoute,
  path: '/',
  component: ProjectDashboard,
})

function ProjectDashboard() {
  const { t } = useTranslation('projects')
  const project = projectRoute.useLoaderData()
  const queryClient = useQueryClient()

  const envsQuery = useQuery({
    queryKey: ['environments', project.slug],
    queryFn: () =>
      fetchJSON<{ environments: Environment[] }>(
        `/api/v1/projects/${project.slug}/environments`,
      ).then((d) => d.environments),
  })

  const flagsQuery = useQuery({
    queryKey: ['flags', project.slug],
    queryFn: () =>
      fetchJSON<{ flags: ProjectFlag[] }>(
        `/api/v1/projects/${project.slug}/flags`,
      ).then((d) => d.flags),
  })

  const [showCreateEnv, setShowCreateEnv] = useState(false)
  const [showCreateFlag, setShowCreateFlag] = useState(false)

  return (
    <div className="p-6 max-w-5xl">
      <ProjectHeader name={project.name} slug={project.slug} />

      <section className="mt-6">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.environments_section')}
        </h2>
        {envsQuery.isLoading ? (
          <EnvironmentCardsSkeleton />
        ) : envsQuery.isError ? (
          <SectionError label="environments" onRetry={() => void envsQuery.refetch()} />
        ) : envsQuery.data!.length === 0 ? (
          <EnvironmentsEmptyState onCreateClick={() => setShowCreateEnv(true)} />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {envsQuery.data!.map((env) => (
              <EnvironmentCard key={env.id} env={env} projectSlug={project.slug} />
            ))}
          </div>
        )}
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.quick_links_section')}
        </h2>
        <div className="flex gap-3 flex-wrap">
          <QuickLink label={t('dashboard.compare')} to={`/projects/${project.slug}/compare`} />
          <QuickLink label={t('dashboard.segments')} to={`/projects/${project.slug}/segments`} />
          <QuickLink label={t('dashboard.settings')} to={`/projects/${project.slug}/settings`} />
          <QuickLink label={t('dashboard.members')} to={`/projects/${project.slug}/members`} />
          <QuickLink label={t('dashboard.api_keys')} to={`/projects/${project.slug}/api-keys`} />
          <QuickLink label={t('dashboard.audit_log')} to={`/projects/${project.slug}/audit`} />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('dashboard.recent_flags_section')}
        </h2>
        {flagsQuery.isLoading ? (
          <RecentFlagsSkeleton />
        ) : flagsQuery.isError ? (
          <SectionError label="flags" onRetry={() => void flagsQuery.refetch()} />
        ) : flagsQuery.data!.length === 0 ? (
          <FlagsEmptyState onCreateClick={() => setShowCreateFlag(true)} />
        ) : (
          <RecentFlagsList flags={flagsQuery.data!} />
        )}
      </section>

      <CreateEnvironmentModal
        open={showCreateEnv}
        slug={project.slug}
        onCreated={() => {
          setShowCreateEnv(false)
          void queryClient.invalidateQueries({ queryKey: ['environments', project.slug] })
        }}
        onCancel={() => setShowCreateEnv(false)}
      />

      {showCreateFlag && (
        <CreateFlagModal
          slug={project.slug}
          onCreated={() => {
            setShowCreateFlag(false)
            void queryClient.invalidateQueries({ queryKey: ['flags', project.slug] })
          }}
          onCancel={() => setShowCreateFlag(false)}
        />
      )}
    </div>
  )
}

// --- Empty states ---

function EnvironmentsEmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-12 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('dashboard.no_environments')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('dashboard.create_environment_cta')}
      </Button>
    </div>
  )
}

function FlagsEmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div className="text-center py-12 px-6">
      <p className="text-sm text-[var(--color-text-secondary)]">
        {t('dashboard.no_flags')}
      </p>
      <Button size="lg" className="mt-4" onClick={onCreateClick}>
        {t('dashboard.create_flag_cta')}
      </Button>
    </div>
  )
}

// --- Create Environment Modal ---

const ENV_SLUG_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_ENV_SLUG_LENGTH = 128

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/^-+|-+$/g, '')
}

function validateEnvSlug(
  slug: string,
  t: (k: string, opts?: Record<string, unknown>) => string,
): string | null {
  if (slug.length === 0) return null
  if (slug.length > MAX_ENV_SLUG_LENGTH)
    return t('environments.slug_too_long', { max: MAX_ENV_SLUG_LENGTH })
  if (!ENV_SLUG_RE.test(slug)) return t('environments.slug_invalid')
  return null
}

function CreateEnvironmentModal({
  open,
  slug,
  onCreated,
  onCancel,
}: {
  open: boolean
  slug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('projects')
  const [name, setName] = useState('')
  const [envSlug, setEnvSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [slugError, setSlugError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () =>
      postJSON(`/api/v1/projects/${slug}/environments`, { name, slug: envSlug }),
    onSuccess: () => onCreated(),
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setSlugError(t('environments.slug_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setSlugError(err.message)
          return
        }
      }
      setServerError(
        err instanceof APIError ? err.message : t('environments.server_error'),
      )
    },
  })

  function handleNameChange(value: string) {
    setName(value)
    if (!slugTouched) {
      const generated = slugify(value)
      setEnvSlug(generated)
      setSlugError(validateEnvSlug(generated, t))
    }
  }

  function handleSlugChange(value: string) {
    setEnvSlug(value)
    setSlugTouched(true)
    setSlugError(null)
    setServerError(null)
  }

  function handleSlugBlur() {
    setSlugError(validateEnvSlug(envSlug, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const err = validateEnvSlug(envSlug, t)
    if (err) {
      setSlugError(err)
      return
    }
    if (!envSlug) {
      setSlugError(t('environments.slug_required'))
      return
    }
    setServerError(null)
    createMutation.mutate()
  }

  function handleOpenChange(isOpen: boolean) {
    if (!isOpen) onCancel()
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('environments.create_title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="env-name-dashboard"
              className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1"
            >
              {t('environments.name_label')}
            </label>
            <input
              id="env-name-dashboard"
              type="text"
              autoFocus
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder={t('environments.name_placeholder')}
              className="w-full text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 rounded px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            />
          </div>
          <div>
            <label
              htmlFor="env-slug-dashboard"
              className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1"
            >
              {t('environments.slug_label')}
            </label>
            <input
              id="env-slug-dashboard"
              type="text"
              value={envSlug}
              onChange={(e) => handleSlugChange(e.target.value)}
              onBlur={handleSlugBlur}
              placeholder={t('environments.slug_placeholder')}
              aria-invalid={!!slugError}
              aria-describedby={slugError ? 'env-slug-dashboard-error' : undefined}
              className={`w-full font-mono text-sm bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 border rounded px-2 py-1.5 focus:outline-none focus:ring-2 ${
                slugError
                  ? 'border-red-300 focus:ring-red-500'
                  : 'border-gray-300 dark:border-gray-600 focus:ring-[var(--color-accent)]'
              }`}
            />
            {slugError && (
              <p id="env-slug-dashboard-error" className="mt-1 text-xs text-red-600 dark:text-red-400">
                {slugError}
              </p>
            )}
          </div>
          {serverError && (
            <p className="text-xs text-red-600 dark:text-red-400">{serverError}</p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={createMutation.isPending}
            >
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button
              type="submit"
              loading={createMutation.isPending}
              disabled={!!slugError || !name.trim() || !envSlug}
            >
              {t('environments.create_button')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// --- Create Flag Modal ---

const FLAG_KEY_RE = /^[a-z0-9][a-z0-9-]*$/
const MAX_FLAG_KEY_LENGTH = 128

function validateFlagKey(
  key: string,
  t: (k: string, opts?: Record<string, unknown>) => string,
): string | null {
  if (key.length === 0) return null
  if (key.length > MAX_FLAG_KEY_LENGTH) return t('create.key_too_long', { max: MAX_FLAG_KEY_LENGTH })
  if (!FLAG_KEY_RE.test(key)) return t('create.key_invalid')
  return null
}

function CreateFlagModal({
  slug,
  onCreated,
  onCancel,
}: {
  slug: string
  onCreated: () => void
  onCancel: () => void
}) {
  const { t } = useTranslation('flags')
  const [key, setKey] = useState('')
  const [name, setName] = useState('')
  const [type, setType] = useState('bool')
  const [keyError, setKeyError] = useState<string | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)
  const [keyTouched, setKeyTouched] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () => {
      const variants =
        type === 'bool'
          ? [{ key: 'true', name: 'On' }, { key: 'false', name: 'Off' }]
          : [{ key: 'default', name: 'Default' }]
      const default_variant_key = type === 'bool' ? 'false' : 'default'
      return postJSON(`/api/v1/projects/${slug}/flags`, {
        key,
        name,
        type,
        variants,
        default_variant_key,
      })
    },
    onSuccess: () => setCreatedKey(key),
    onError: (err) => {
      if (err instanceof APIError) {
        if (err.status === 409 || err.code === 'conflict') {
          setKeyError(t('create.key_conflict'))
          return
        }
        if (err.status === 400 && err.code === 'validation_error') {
          setKeyError(err.message)
          return
        }
      }
      setServerError(err instanceof APIError ? err.message : t('create.server_error'))
    },
  })

  function handleKeyChange(value: string) {
    setKey(value)
    setKeyError(null)
    setServerError(null)
    if (keyTouched) {
      setKeyError(validateFlagKey(value, t))
    }
  }

  function handleKeyBlur() {
    setKeyTouched(true)
    setKeyError(validateFlagKey(key, t))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const localError = validateFlagKey(key, t)
    if (localError) {
      setKeyError(localError)
      return
    }
    if (key.length === 0) {
      setKeyError(t('create.key_required'))
      return
    }
    setServerError(null)
    createMutation.mutate()
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) {
          if (createdKey) onCreated()
          else onCancel()
        }
      }}
    >
      <DialogContent>
        <DialogCloseButton />
        {createdKey ? (
          <>
            <DialogHeader>
              <DialogTitle>{t('create.success_title')}</DialogTitle>
            </DialogHeader>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              {t('create.success_body')}
            </p>
            <CopyableCode
              value={createdKey}
              aria-label={t('create.success_copy_aria', { key: createdKey })}
              className="w-full justify-between"
            />
            <DialogFooter>
              <Button variant="primary" onClick={onCreated}>
                {t('create.success_done')}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>{t('create.title')}</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="flag-key-dashboard" className="text-xs text-gray-500 mb-1">
                  {t('create.key_label')}
                </Label>
                <Input
                  id="flag-key-dashboard"
                  type="text"
                  autoFocus
                  value={key}
                  onChange={(e) => handleKeyChange(e.target.value)}
                  onBlur={handleKeyBlur}
                  placeholder={t('create.key_placeholder')}
                  aria-invalid={!!keyError}
                  aria-describedby={keyError ? 'flag-key-dashboard-error' : undefined}
                  hasError={!!keyError}
                  className="font-mono py-1.5 px-2"
                />
                {keyError && (
                  <p id="flag-key-dashboard-error" className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {keyError}
                  </p>
                )}
              </div>

              <div>
                <Label htmlFor="flag-name-dashboard" className="text-xs text-gray-500 mb-1">
                  {t('create.name_label')}
                </Label>
                <Input
                  id="flag-name-dashboard"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t('create.name_placeholder')}
                  className="py-1.5 px-2"
                />
              </div>

              <div>
                <Label htmlFor="flag-type-dashboard" className="text-xs text-gray-500 mb-1">
                  {t('create.type_label')}
                </Label>
                <Select
                  value={type}
                  onValueChange={setType}
                  aria-label={t('create.type_aria')}
                  className="w-full"
                >
                  <SelectItem value="bool">{t('create.type_bool')}</SelectItem>
                  <SelectItem value="string">{t('create.type_string')}</SelectItem>
                  <SelectItem value="number">{t('create.type_number')}</SelectItem>
                  <SelectItem value="json">{t('create.type_json')}</SelectItem>
                </Select>
              </div>

              {serverError && (
                <p className="text-xs text-red-600 dark:text-red-400">{serverError}</p>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={onCancel}
                  disabled={createMutation.isPending}
                >
                  {t('actions.cancel', { ns: 'common' })}
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  loading={createMutation.isPending}
                  disabled={!!keyError}
                >
                  {createMutation.isPending
                    ? t('states.creating', { ns: 'common' })
                    : t('actions.create', { ns: 'common' })}
                </Button>
              </div>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

// --- Existing components (unchanged) ---

function ProjectHeader({ name, slug }: { name: string; slug: string }) {
  const { t } = useTranslation('projects')
  const [copied, setCopied] = useState(false)

  function copySlug() {
    void navigator.clipboard
      .writeText(slug)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 1500)
      })
      .catch(() => {})
  }

  return (
    <>
      <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{name}</h1>
      <div className="mt-1 flex items-center gap-2">
        <button
          onClick={copySlug}
          className="relative font-mono text-sm text-gray-500 dark:text-gray-400 hover:text-[var(--color-accent)] bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded px-2 py-0.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
          aria-label={t('dashboard.copy_slug_aria', { slug })}
        >
          {slug}
          {copied && (
            <span className="absolute -top-7 left-1/2 -translate-x-1/2 text-xs bg-gray-800 dark:bg-gray-600 text-white rounded px-2 py-0.5 whitespace-nowrap pointer-events-none">
              {t('settings.copied')}
            </span>
          )}
        </button>
      </div>
    </>
  )
}

function EnvironmentCard({ env, projectSlug }: { env: Environment; projectSlug: string }) {
  const { t } = useTranslation('projects')
  const flagsQuery = useQuery({
    queryKey: ['flags', projectSlug, env.slug],
    queryFn: () =>
      fetchJSON<{ flags: EnvFlag[] }>(
        `/api/v1/projects/${projectSlug}/environments/${env.slug}/flags`,
      ).then((d) => d.flags),
  })

  const total = flagsQuery.data?.length ?? 0
  const enabled = flagsQuery.data?.filter((f) => f.enabled).length ?? 0

  return (
    <Link
      to="/projects/$slug/environments/$envSlug/flags"
      params={{ slug: projectSlug, envSlug: env.slug }}
      className="block border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 p-4 hover:border-[var(--color-accent)] hover:shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      <h3 className="font-mono text-sm font-medium text-gray-900 dark:text-gray-100">{env.name}</h3>
      {flagsQuery.isLoading ? (
        <div className="mt-2 h-4 w-20 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
      ) : flagsQuery.isError ? (
        <p className="mt-2 text-xs text-red-600 dark:text-red-400">{t('dashboard.failed_to_load_env')}</p>
      ) : (
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          <span className="font-medium text-gray-800 dark:text-gray-200">{enabled}</span>
          <span className="text-gray-400 dark:text-gray-500"> / </span>
          <span>{total}</span>
          <span className="text-gray-400 dark:text-gray-500"> {t('toggle.enabled', { ns: 'flags' }).toLowerCase()}</span>
        </p>
      )}
    </Link>
  )
}

function QuickLink({ label, to }: { label: string; to: string }) {
  return (
    <Link
      to={to}
      className="px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-700 rounded hover:bg-gray-50 dark:hover:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600 transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
    >
      {label}
    </Link>
  )
}

function RecentFlagsList({ flags }: { flags: ProjectFlag[] }) {
  const recent = [...flags]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 5)

  return (
    <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
      {recent.map((flag) => (
        <li key={flag.id} className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3 min-w-0">
            <span className="font-mono text-sm text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded px-2 py-0.5">
              {flag.key}
            </span>
            <span className="text-sm text-gray-600 dark:text-gray-400 truncate">{flag.name}</span>
          </div>
          <time
            dateTime={flag.created_at}
            className="text-xs text-gray-400 dark:text-gray-500 shrink-0 ml-4"
            title={new Date(flag.created_at).toLocaleString()}
          >
            {formatRelativeDate(flag.created_at)}
          </time>
        </li>
      ))}
    </ul>
  )
}

function EnvironmentCardsSkeleton() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      {[1, 2, 3].map((i) => (
        <div key={i} className="border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 p-4">
          <div className="h-4 w-20 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
          <div className="mt-2 h-4 w-24 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        </div>
      ))}
    </div>
  )
}

function RecentFlagsSkeleton() {
  return (
    <ul className="divide-y divide-gray-100 dark:divide-gray-700 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800">
      {[1, 2, 3].map((i) => (
        <li key={i} className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            <div className="h-6 w-28 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            <div className="h-4 w-40 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
          </div>
          <div className="h-3 w-12 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
        </li>
      ))}
    </ul>
  )
}

function SectionError({ label, onRetry }: { label: string; onRetry: () => void }) {
  const { t } = useTranslation('projects')
  return (
    <div>
      <span className="text-sm text-red-600 dark:text-red-400">{t('dashboard.failed_to_load', { resource: label })} </span>
      <button
        onClick={onRetry}
        className="text-sm text-red-600 dark:text-red-400 underline hover:no-underline focus:outline-none focus:ring-2 focus:ring-red-500 rounded"
      >
        {t('actions.retry', { ns: 'common' })}
      </button>
    </div>
  )
}
