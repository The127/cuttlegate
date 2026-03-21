import { getUserManager } from './auth'

export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'APIError'
  }
}

async function authedFetch(path: string, init?: RequestInit): Promise<Response> {
  const user = await getUserManager().getUser()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (user?.access_token) {
    headers['Authorization'] = `Bearer ${user.access_token}`
  }
  return fetch(path, { ...init, headers })
}

export async function fetchJSON<T>(path: string): Promise<T> {
  const res = await authedFetch(path)
  if (!res.ok) {
    throw new APIError(res.status, `${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function patchJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await authedFetch(path, { method: 'PATCH', body: JSON.stringify(body) })
  if (!res.ok) {
    throw new APIError(res.status, `${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function deleteRequest(path: string): Promise<void> {
  const res = await authedFetch(path, { method: 'DELETE' })
  if (!res.ok) {
    throw new APIError(res.status, `${res.status} ${res.statusText}`)
  }
}
