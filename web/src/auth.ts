import { UserManager } from 'oidc-client-ts'

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
  })
  return _userManager
}

export function getUserManager(): UserManager {
  if (_userManager === null) {
    throw new Error('UserManager not initialized — call initUserManager() first')
  }
  return _userManager
}
