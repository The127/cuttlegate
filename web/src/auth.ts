import { UserManager, WebStorageStateStore } from 'oidc-client-ts'

/**
 * In-memory storage adapter for OIDC tokens.
 * Tokens must never touch sessionStorage or localStorage (XSS risk).
 * When the tab closes, the user re-authenticates — this is acceptable.
 */
class InMemoryStorage implements Storage {
  private data = new Map<string, string>()

  get length(): number {
    return this.data.size
  }

  clear(): void {
    this.data.clear()
  }

  getItem(key: string): string | null {
    return this.data.get(key) ?? null
  }

  key(index: number): string | null {
    const keys = Array.from(this.data.keys())
    return keys[index] ?? null
  }

  removeItem(key: string): void {
    this.data.delete(key)
  }

  setItem(key: string, value: string): void {
    this.data.set(key, value)
  }
}

let _userManager: UserManager | null = null

export interface OIDCConfig {
  authority: string
  client_id: string
  redirect_uri: string
}

export function initUserManager(config: OIDCConfig): UserManager {
  _userManager = new UserManager({
    authority: config.authority,
    client_id: config.client_id,
    redirect_uri: config.redirect_uri,
    response_type: 'code',
    scope: 'openid profile email offline_access',
    automaticSilentRenew: true,
    userStore: new WebStorageStateStore({ store: new InMemoryStorage() }),
  })
  return _userManager
}

export function getUserManager(): UserManager {
  if (_userManager === null) {
    throw new Error('UserManager not initialized — call initUserManager() first')
  }
  return _userManager
}
