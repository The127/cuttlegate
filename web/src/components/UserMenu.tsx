import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { getUserManager } from '../auth'
import { queryClient } from '../queryClient'

/** Derive up to two initials from a full name string. Returns "?" if name is absent. */
function deriveInitials(name: string | undefined): string {
  if (!name) return '?'
  const words = name.trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) return '?'
  if (words.length === 1) return words[0].charAt(0).toUpperCase()
  return (words[0].charAt(0) + words[words.length - 1].charAt(0)).toUpperCase()
}

export function UserMenu() {
  const { t } = useTranslation('common')
  const [initials, setInitials] = useState<string>('?')
  const [email, setEmail] = useState<string>('')
  const loggingOut = useRef(false)

  useEffect(() => {
    void getUserManager()
      .getUser()
      .then((user) => {
        if (user) {
          setInitials(deriveInitials(user.profile.name))
          setEmail((user.profile.email as string) ?? '')
        }
      })
  }, [])

  const handleLogout = useCallback(async () => {
    if (loggingOut.current) return
    loggingOut.current = true

    queryClient.clear()

    try {
      await getUserManager().signoutRedirect({
        post_logout_redirect_uri: window.location.origin,
      })
    } catch {
      // Fallback: OIDC provider may not support end_session_endpoint
      await getUserManager().removeUser()
      window.location.href = '/'
    }
  }, [])

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          className="h-8 w-8 rounded-full flex items-center justify-center text-xs font-semibold text-white select-none shrink-0 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-[rgba(79,124,255,0.35)]"
          style={{
            background:
              'linear-gradient(135deg, var(--color-accent-start), var(--color-accent-end))',
          }}
          aria-label={`User menu: ${initials}`}
        >
          {initials}
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={6}
          className="min-w-[200px] py-1 rounded-[var(--radius-md)] border border-[var(--color-border-hover)] bg-[var(--color-surface-elevated)] z-50"
          style={{ boxShadow: '0 24px 48px rgba(0,0,0,0.5)' }}
        >
          <DropdownMenu.Label className="px-3 py-2 text-xs text-[var(--color-text-muted)]">
            {t('user_menu.logged_in_as')}
          </DropdownMenu.Label>

          <div
            className="px-3 pb-2 text-sm text-[var(--color-text-primary)] truncate"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            {email}
          </div>

          <DropdownMenu.Separator className="h-px my-1 bg-[var(--color-border)]" />

          <DropdownMenu.Item
            className="px-3 py-2 text-sm text-[var(--color-text-primary)] rounded-[var(--radius-sm)] mx-1 cursor-pointer outline-none select-none data-[highlighted]:bg-[rgba(248,113,113,0.12)] data-[highlighted]:text-[var(--color-status-error)]"
            onSelect={() => void handleLogout()}
          >
            {t('user_menu.log_out')}
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
