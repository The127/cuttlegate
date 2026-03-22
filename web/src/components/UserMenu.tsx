import { useEffect, useState } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { getUserManager } from '../auth'

interface UserIdentity {
  name: string
  email: string
}

function getInitial(name: string): string {
  return name.charAt(0).toUpperCase()
}

function extractIdentity(profile: { name?: string; email?: string }): UserIdentity {
  const name = profile.name ?? profile.email ?? 'User'
  const email = profile.email ?? ''
  return { name, email }
}

function useUserIdentity(): UserIdentity | null {
  const [identity, setIdentity] = useState<UserIdentity | null>(null)

  useEffect(() => {
    let cancelled = false
    const mgr = getUserManager()

    mgr.getUser().then((user) => {
      if (cancelled || !user) return
      setIdentity(extractIdentity(user.profile))
    }).catch(() => {
      // getUser failure — identity stays null, badge hidden
    })

    const onUserLoaded = (user: { profile: { name?: string; email?: string } }) => {
      if (!cancelled) setIdentity(extractIdentity(user.profile))
    }
    mgr.events.addUserLoaded(onUserLoaded)

    return () => {
      cancelled = true
      mgr.events.removeUserLoaded(onUserLoaded)
    }
  }, [])

  return identity
}

let loggingOut = false

async function handleLogout(): Promise<void> {
  if (loggingOut) return
  loggingOut = true
  const mgr = getUserManager()
  try {
    await mgr.signoutRedirect()
  } catch {
    await mgr.removeUser()
    window.location.href = '/'
  } finally {
    loggingOut = false
  }
}

export function UserMenu() {
  const identity = useUserIdentity()

  if (!identity) return null

  const initial = getInitial(identity.name)

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          className="flex items-center justify-center w-8 h-8 rounded-full bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          aria-label="User menu"
        >
          {initial}
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={4}
          className="min-w-[200px] bg-white rounded-md border border-gray-200 shadow-lg py-1 z-50"
        >
          <DropdownMenu.Label className="px-3 py-2 text-xs text-gray-500">
            Logged in as
          </DropdownMenu.Label>
          <DropdownMenu.Label className="px-3 pb-2 text-sm text-gray-900 font-medium truncate">
            {identity.email}
          </DropdownMenu.Label>

          <DropdownMenu.Separator className="h-px bg-gray-200 my-1" />

          <DropdownMenu.Item
            onSelect={() => void handleLogout()}
            className="px-3 py-2 text-sm text-gray-700 cursor-pointer outline-none hover:bg-gray-100 focus:bg-gray-100"
          >
            Log out
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
