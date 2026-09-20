import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  LOCALE_STORAGE_KEY,
  lineLocale,
  readStoredLocale,
  resolveInitialLocale,
  storeLocale,
} from './initial-locale'

/**
 * The #94 boot-locale chain: stored choice → LINE language → Thai. Every
 * storage touch is guarded (some in-app webviews ship with storage
 * disabled and throw on access) — a preference must never crash the app.
 */

function fakeStorage(initial: Record<string, string> = {}) {
  const map = new Map(Object.entries(initial))
  return {
    getItem: (key: string) => map.get(key) ?? null,
    setItem: (key: string, value: string) => void map.set(key, value),
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('readStoredLocale', () => {
  it('returns a stored supported locale', () => {
    vi.stubGlobal('window', { localStorage: fakeStorage({ [LOCALE_STORAGE_KEY]: 'en' }) })
    expect(readStoredLocale()).toBe('en')
  })

  it('ignores an unsupported or corrupt stored value', () => {
    vi.stubGlobal('window', { localStorage: fakeStorage({ [LOCALE_STORAGE_KEY]: 'ja' }) })
    expect(readStoredLocale()).toBeUndefined()
    vi.stubGlobal('window', { localStorage: fakeStorage({ [LOCALE_STORAGE_KEY]: '' }) })
    expect(readStoredLocale()).toBeUndefined()
  })

  it('survives a webview whose storage access throws', () => {
    vi.stubGlobal('window', {
      get localStorage(): Storage {
        throw new Error('the document is sandboxed')
      },
    })
    expect(readStoredLocale()).toBeUndefined()
  })

  it('survives having no window at all (SSR / test render)', () => {
    expect(readStoredLocale()).toBeUndefined()
  })
})

describe('storeLocale', () => {
  it('writes the key the reader looks for', () => {
    const storage = fakeStorage()
    vi.stubGlobal('window', { localStorage: storage })
    storeLocale('en')
    expect(readStoredLocale()).toBe('en')
  })

  it('never throws when storage is unavailable', () => {
    vi.stubGlobal('window', {
      get localStorage(): Storage {
        throw new Error('denied')
      },
    })
    expect(() => storeLocale('th')).not.toThrow()
  })
})

describe('lineLocale', () => {
  it.each([
    ['en', 'en'],
    ['en-US', 'en'],
    ['th', 'th'],
  ])('maps LINE language %s to %s', (language, expected) => {
    expect(lineLocale(language)).toBe(expected)
  })

  it('maps unsupported LINE languages to nothing — Thai is the default anyway', () => {
    expect(lineLocale('ja')).toBeUndefined()
    expect(lineLocale(undefined)).toBeUndefined()
  })
})

describe('resolveInitialLocale', () => {
  it('lets a stored choice win over the LINE language', () => {
    vi.stubGlobal('window', { localStorage: fakeStorage({ [LOCALE_STORAGE_KEY]: 'en' }) })
    expect(resolveInitialLocale('th')).toBe('en')
  })

  it('adopts an English LINE when nothing is stored — the foreign patient case (US-11)', () => {
    expect(resolveInitialLocale('en')).toBe('en')
  })

  it('falls back to Thai for an unsupported LINE language or nothing at all', () => {
    expect(resolveInitialLocale('ja')).toBe('th')
    expect(resolveInitialLocale(undefined)).toBe('th')
  })
})
