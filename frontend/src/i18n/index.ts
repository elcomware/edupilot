import { en } from './en'
import { fr } from './fr'

export type Locale = 'en' | 'fr'
export type Messages = Record<string, string>

const catalogues: Record<Locale, Messages> = { en, fr }

let currentLocale: Locale = 'en'

export function setLocale(locale: Locale): void {
  currentLocale = locale
}

export function getLocale(): Locale {
  return currentLocale
}

export function availableLocales(): readonly Locale[] {
  return ['en', 'fr']
}

export type TranslationParams = Readonly<Record<string, string | number>>

/**
 * Resolves dotted keys such as `payments.actions.pay`. User-facing text is
 * never hard-coded inside components.
 */
export function t(key: string, params?: TranslationParams): string {
  const template = catalogues[currentLocale][key] ?? catalogues.en[key]

  if (template === undefined) {
    return key
  }

  if (params === undefined) {
    return template
  }

  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name]
    return value === undefined ? match : String(value)
  })
}
