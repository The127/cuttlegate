import { createContext, useContext, type ReactNode } from 'react'

export interface BrandConfig {
  app_name: string
  logo_url: string | null
  accent_colour: string
}

const defaultBrand: BrandConfig = {
  app_name: 'Cuttlegate',
  logo_url: null,
  accent_colour: '#2563eb',
}

const BrandContext = createContext<BrandConfig>(defaultBrand)

export function BrandProvider({ config, children }: { config: BrandConfig; children: ReactNode }) {
  return <BrandContext.Provider value={config}>{children}</BrandContext.Provider>
}

export function useBrand(): BrandConfig {
  return useContext(BrandContext)
}
