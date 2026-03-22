import '@testing-library/jest-dom/vitest'
import 'jest-axe/extend-expect'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import commonEn from '../../public/locales/en/common.json'
import projectsEn from '../../public/locales/en/projects.json'
import flagsEn from '../../public/locales/en/flags.json'
import segmentsEn from '../../public/locales/en/segments.json'
import rulesEn from '../../public/locales/en/rules.json'

// Initialize i18next synchronously for tests — loads actual English translations
// so test assertions against rendered strings continue to work after i18n extraction.
if (!i18n.isInitialized) {
  void i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    resources: {
      en: {
        common: commonEn,
        projects: projectsEn,
        flags: flagsEn,
        segments: segmentsEn,
        rules: rulesEn,
      },
    },
    interpolation: { escapeValue: false },
  })
}

// Radix UI components use ResizeObserver — stub it for jsdom
if (typeof window !== 'undefined' && !window.ResizeObserver) {
  window.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

// Clipboard API — jsdom does not implement navigator.clipboard.
// Define a stub so components using navigator.clipboard.writeText do not throw.
// Individual tests that assert clipboard behaviour can spy on navigator.clipboard.writeText.
if (typeof window !== 'undefined' && !navigator.clipboard) {
  Object.defineProperty(window.navigator, 'clipboard', {
    value: {
      writeText: () => Promise.resolve(),
      readText: () => Promise.resolve(''),
    },
    configurable: true,
    writable: true,
  })
}
