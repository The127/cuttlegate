import React from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { initUserManager, type OIDCConfig } from './auth'
import { createAppRouter } from './router'
import { queryClient } from './queryClient'
import { initI18n, i18n } from './i18n'
import './styles.css'

async function fetchOIDCConfig(): Promise<OIDCConfig> {
  const res = await fetch('/api/v1/config')
  if (!res.ok) {
    throw new Error(`Failed to load OIDC config: ${res.status}`)
  }
  return (await res.json()) as OIDCConfig
}

async function bootstrap() {
  const [config] = await Promise.all([fetchOIDCConfig(), initI18n()])

  document.title = i18n.t('app_name')

  initUserManager(config)

  const router = createAppRouter()

  const rootEl = document.getElementById('root')
  if (rootEl === null) throw new Error('Missing #root element')

  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </React.StrictMode>,
  )
}

bootstrap().catch((err: unknown) => {
  const message = err instanceof Error ? err.message : String(err)
  console.error('Bootstrap failed:', message)
  const rootEl = document.getElementById('root')
  if (rootEl !== null) {
    rootEl.textContent = `Failed to start application: ${message}`
  }
})
