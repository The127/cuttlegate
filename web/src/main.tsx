import React from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { initUserManager, type OIDCConfig } from './auth'
import { BrandProvider, type BrandConfig } from './brand'
import { createAppRouter } from './router'
import { queryClient } from './queryClient'
import { initI18n } from './i18n'
import { ErrorBoundary } from './ErrorBoundary'
import './styles.css'

interface AppConfig extends OIDCConfig, BrandConfig {}

async function bootstrap() {
  const res = await fetch('/api/v1/config')
  if (!res.ok) {
    throw new Error(`Failed to load config: ${res.status}`)
  }
  const [config] = await Promise.all([(res.json() as Promise<AppConfig>), initI18n()])

  // Apply branding before first render to avoid flash of default styles.
  // The single --color-accent token is the only accent custom property in @theme.
  // Verified post-redesign (Sprint 25, #317): no gradient tokens exist — this
  // override covers all accent usages (Button bg, focus rings, hover states).
  document.documentElement.style.setProperty('--color-accent', config.accent_colour)
  document.title = config.app_name

  initUserManager(config)

  const router = createAppRouter()

  const rootEl = document.getElementById('root')
  if (rootEl === null) throw new Error('Missing #root element')

  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <ErrorBoundary>
        <BrandProvider config={config}>
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </BrandProvider>
      </ErrorBoundary>
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
