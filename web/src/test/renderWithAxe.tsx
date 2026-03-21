import { render, type RenderOptions, type RenderResult } from '@testing-library/react'
import { axe, type AxeResults } from 'jest-axe'
import type { ReactElement } from 'react'

interface RenderWithAxeResult extends RenderResult {
  axeResults: AxeResults
}

/**
 * Render a component and run an axe-core accessibility scan.
 * Returns the standard Testing Library render result plus `axeResults`.
 *
 * Usage:
 *   const { axeResults } = await renderWithAxe(<MyComponent />)
 *   expect(axeResults).toHaveNoViolations()
 */
export async function renderWithAxe(
  ui: ReactElement,
  options?: RenderOptions,
): Promise<RenderWithAxeResult> {
  const result = render(ui, options)
  const axeResults = await axe(result.container)
  return { ...result, axeResults }
}
