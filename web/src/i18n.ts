import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import HttpBackend from 'i18next-http-backend'

/**
 * Initialise i18next with the HTTP backend.
 *
 * Locale files are served as static assets from /locales/<lang>/<ns>.json.
 * Adding a new locale requires only a new web/public/locales/<lang>/ directory
 * with translated JSON files — no code changes needed.
 *
 * TODO(#194): Once BrandContext ships, override the `common:app_name` key at
 * runtime using the `app_name` value from GET /api/v1/config. The i18n
 * instance is exported below so callers can call i18n.addResource() or
 * i18n.changeLanguage() as needed.
 */
export function initI18n(): Promise<void> {
  return i18n
    .use(HttpBackend)
    .use(initReactI18next)
    .init({
      lng: 'en',
      fallbackLng: 'en',
      ns: ['common', 'flags', 'segments', 'rules', 'projects'],
      defaultNS: 'common',
      backend: {
        loadPath: '/locales/{{lng}}/{{ns}}.json',
      },
      interpolation: {
        escapeValue: false,
      },
    })
    .then(() => undefined)
}

export { i18n }
