import { getUserManager } from './auth'

export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly code: string = '',
  ) {
    super(message)
    this.name = 'APIError'
  }
}

// authedFetch injects Authorization and Content-Type headers, overriding any
// same-named headers in init. Callers must not set those headers themselves.
async function authedFetch(path: string, init?: RequestInit): Promise<Response> {
  const user = await getUserManager().getUser()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (user?.access_token) {
    headers['Authorization'] = `Bearer ${user.access_token}`
  }
  return fetch(path, { ...init, headers })
}

async function throwIfNotOk(res: Response): Promise<void> {
  if (res.ok) return
  let message = `${res.status} ${res.statusText}`
  let code = ''
  try {
    const body = JSON.parse(await res.text())
    if (typeof body.message === 'string') {
      message = body.message
    }
    if (typeof body.error === 'string') {
      code = body.error
    }
  } catch {
    // Response is not valid JSON — keep the status text fallback.
  }
  throw new APIError(res.status, message, code)
}

export async function fetchJSON<T>(path: string): Promise<T> {
  const res = await authedFetch(path)
  await throwIfNotOk(res)
  return res.json() as Promise<T>
}

export async function patchJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await authedFetch(path, { method: 'PATCH', body: JSON.stringify(body) })
  await throwIfNotOk(res)
  return res.json() as Promise<T>
}

export async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await authedFetch(path, { method: 'POST', body: JSON.stringify(body) })
  await throwIfNotOk(res)
  return res.json() as Promise<T>
}

export async function putJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await authedFetch(path, { method: 'PUT', body: JSON.stringify(body) })
  await throwIfNotOk(res)
  return res.json() as Promise<T>
}

export async function patchEmpty(path: string, body: unknown): Promise<void> {
  const res = await authedFetch(path, { method: 'PATCH', body: JSON.stringify(body) })
  await throwIfNotOk(res)
}

export async function deleteRequest(path: string): Promise<void> {
  const res = await authedFetch(path, { method: 'DELETE' })
  await throwIfNotOk(res)
}
