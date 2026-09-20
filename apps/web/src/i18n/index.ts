import { createContext, createElement, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { en } from './locales/en'
import { th, type Catalog, type MessageKey } from './locales/th'

export type Locale = 'th' | 'en'
export type { Catalog, MessageKey }

const CATALOGS: Record<Locale, Catalog> = { th, en }

/**
 * All messages for a locale — plain data, no React, so pure mapping code
 * (`features/visit/journey.ts`) can localize without a hook.
 */
export function messagesFor(locale: Locale): Catalog {
  return CATALOGS[locale]
}

/**
 * Catalog lookup by a runtime-built key (`sp.${code}`, `step.title.${kind}`).
 * Returns undefined for an absent key — callers own the fallback (ADR-0012
 * §1: an unknown code renders something readable, never crashes).
 */
export function lookup(catalog: Catalog, key: string): string | undefined {
  return Object.hasOwn(catalog, key) ? catalog[key as MessageKey] : undefined
}

/**
 * Catalog entry with `{placeholder}` interpolation — `format(catalog, key,
 * { done: 2 })`. No plural rules yet; none of the patient text needs them
 * (ADR-0012 §3). `useT` and the pure locale-taking mappers both go through
 * here so interpolation has exactly one implementation.
 */
export function format(
  catalog: Catalog,
  key: MessageKey,
  params?: Record<string, string | number>,
): string {
  let text = catalog[key]
  if (params) {
    for (const [name, value] of Object.entries(params)) {
      text = text.replaceAll(`{${name}}`, String(value))
    }
  }
  return text
}

/**
 * App locale context. #92 ships the mechanism with Thai fixed: no toggle,
 * no persistence, no `<html lang>` writes — those land with #94. The default
 * value (rather than a null that throws) keeps provider-less usage — the
 * /design reference screens, tests — rendering in Thai instead of crashing.
 * `initialLocale` exists for the English render test now and for #94's
 * resolved persisted/LINE locale at the app root later.
 */
type LocaleContextValue = { locale: Locale; setLocale: (locale: Locale) => void }

const LocaleContext = createContext<LocaleContextValue>({
  locale: 'th',
  setLocale: () => {},
})

export function LocaleProvider({
  children,
  initialLocale = 'th',
}: {
  children: ReactNode
  initialLocale?: Locale
}) {
  const [locale, setLocale] = useState<Locale>(initialLocale)
  const value = useMemo<LocaleContextValue>(() => ({ locale, setLocale }), [locale])
  // createElement instead of JSX so this stays `index.ts`, as the i18n
  // module's plain-TS entry point.
  return createElement(LocaleContext.Provider, { value }, children)
}

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext)
}

/**
 * Translate a catalog key, with simple `{placeholder}` interpolation:
 * `t('journey.progress', { done: 2, total: 5 })`. Delegates to `format`.
 */
export function useT() {
  const { locale } = useLocale()
  const catalog = messagesFor(locale)
  return useCallback(
    (key: MessageKey, params?: Record<string, string | number>) => format(catalog, key, params),
    [catalog],
  )
}
