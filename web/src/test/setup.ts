import '@testing-library/jest-dom/vitest'
import 'jest-axe/extend-expect'

// Radix UI components use ResizeObserver — stub it for jsdom
if (typeof window !== 'undefined' && !window.ResizeObserver) {
  window.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}
