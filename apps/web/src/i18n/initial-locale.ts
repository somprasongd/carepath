import type { Locale } from '.'

/**
 * The #94 default-locale chain: a stored choice beats everything, then the
 * LINE app's language (the best signal for the foreign patient US-11
 * describes — they should get English on the very first screen), then Thai.
 * The preference is device-local: it is not a property of the visit and
 * must never ride the session token (ADR-0010).
 */

export const LOCALE_STORAGE_KEY = 'carepath.locale'

/**
 * The user's remembered choice, or undefined. Every read is guarded: some
 * in-app webviews ship with storage disabled and throw on access, and a
 * locale preference is never worth crashing the app over.
 */
export function readStoredLocale(): Locale | undefined {
  try {
    const value = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    return value === 'th' || value === 'en' ? value : undefined
  } catch {
    return undefined
  }
}

/** Remember the choice; storage being unavailable just means no memory. */
export function storeLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // Best effort by design.
  }
}

/**
 * LINE's language (`liff.getLanguage()`, e.g. "en" / "th" / "ja") mapped to
 * a supported app locale. Unsupported languages fall back to Thai by
 * returning undefined, and only 'en' actually changes anything — 'th' is
 * already the default.
 */
export function lineLocale(language: string | undefined): Locale | undefined {
  if (!language) return undefined
  const normalized = language.toLowerCase()
  if (normalized.startsWith('en')) return 'en'
  if (normalized.startsWith('th')) return 'th'
  return undefined
}

/**
 * The locale the app boots with. `lineLanguage` comes from
 * `liff.getLanguage()` after `main.tsx`'s bootstrap has awaited
 * `liff.init()` — so in LINE mode the very first paint already carries the
 * resolved locale, no flash of Thai first.
 */
export function resolveInitialLocale(lineLanguage?: string): Locale {
  return readStoredLocale() ?? lineLocale(lineLanguage) ?? 'th'
}
