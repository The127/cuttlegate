# Frontend Index

_Generated 2026-03-23 13:50 UTC. Read this at session start for frontend orientation._

## Routes (`web/src/routes/`)

| Route file | URL pattern | Params | Description |
|---|---|---|---|
| `auth/callback.tsx` | `/auth/callback` | — | — |
| `index.tsx` | `/` | — | — |
| `projects/$slug.api-keys.tsx` | `/projects/:slug/api-keys` | $slug | — |
| `projects/$slug.audit.tsx` | `/projects/:slug/audit` | $slug | — |
| `projects/$slug.compare.tsx` | `/projects/:slug/compare` | $slug | — |
| `projects/$slug.environments.$envSlug.flags.$key.evaluations.tsx` | `/projects/:slug/environments/:envSlug/flags/:key/evaluations` | $slug, $envSlug, $key | — |
| `projects/$slug.environments.$envSlug.flags.$key.rules.tsx` | `/projects/:slug/environments/:envSlug/flags/:key/rules` | $slug, $envSlug, $key | — |
| `projects/$slug.environments.$envSlug.flags.$key.tsx` | `/projects/:slug/environments/:envSlug/flags/:key` | $slug, $envSlug, $key | — |
| `projects/$slug.environments.$envSlug.flags.tsx` | `/projects/:slug/environments/:envSlug/flags` | $slug, $envSlug | — |
| `projects/$slug.environments.$envSlug.tsx` | `/projects/:slug/environments/:envSlug` | $slug, $envSlug | — |
| `projects/$slug.index.tsx` | `/projects/:slug` | $slug | — |
| `projects/$slug.members.tsx` | `/projects/:slug/members` | $slug | — |
| `projects/$slug.segments.tsx` | `/projects/:slug/segments` | $slug | — |
| `projects/$slug.settings.environments.tsx` | `/projects/:slug/settings/environments` | $slug | — |
| `projects/$slug.settings.tsx` | `/projects/:slug/settings` | $slug | — |
| `projects/$slug.tsx` | `/projects/:slug` | $slug | — |

## Components (`web/src/components/`)

| Component | File | Description |
|---|---|---|
| `Breadcrumbs` | `Breadcrumbs.tsx` | — |
| `useOpenCreateProjectDialog` | `CreateProjectDialog.tsx` | — |
| `CreateProjectDialogProvider` | `CreateProjectDialog.tsx` | — |
| `FlagAnalyticsPanel` | `FlagAnalyticsPanel.tsx` | — |
| `ProjectSwitcher` | `ProjectSwitcher.tsx` | — |
| `PromoteDialog` | `PromoteDialog.tsx` | — |
| `Button` | `ui/Button.tsx` | — |
| `CopyableCode` | `ui/CopyableCode.tsx` | — |
| `DataTable` | `ui/DataTable.tsx` | — |
| `Dialog` | `ui/Dialog.tsx` | — |
| `DialogTrigger` | `ui/Dialog.tsx` | — |
| `DialogContent` | `ui/Dialog.tsx` | — |
| `DialogHeader` | `ui/Dialog.tsx` | — |
| `DialogTitle` | `ui/Dialog.tsx` | — |
| `DialogDescription` | `ui/Dialog.tsx` | — |
| `DialogFooter` | `ui/Dialog.tsx` | — |
| `DialogClose` | `ui/Dialog.tsx` | — |
| `DialogCloseButton` | `ui/Dialog.tsx` | — |
| `FormFieldContext` | `ui/FormField.tsx` | — |
| `useFormFieldId` | `ui/FormField.tsx` | — |
| `FormField` | `ui/FormField.tsx` | — |
| `Input` | `ui/Input.tsx` | — |
| `Label` | `ui/Label.tsx` | — |
| `SelectItem` | `ui/Select.tsx` | — |
| `Select` | `ui/Select.tsx` | — |
| `StatusBadge` | `ui/StatusBadge.tsx` | — |

## Hooks (`web/src/hooks/`)

| Hook | File | Description |
|---|---|---|
| `useFlagSSE` | `useFlagSSE.ts` | — |
| `useProjectRole` | `useProjectRole.ts` | — |

## API surface (`web/src/api.ts`)

```
export async function fetchJSON<T>(path: string): Promise<T>
export async function patchJSON<T>(path: string, body: unknown): Promise<T>
export async function postJSON<T>(path: string, body: unknown): Promise<T>
export async function putJSON<T>(path: string, body: unknown): Promise<T>
export async function patchEmpty(path: string, body: unknown): Promise<void>
export async function deleteRequest(path: string): Promise<void>
export class APIError
```

