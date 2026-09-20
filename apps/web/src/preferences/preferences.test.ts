import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  ACCESSIBLE_ONLY_STORAGE_KEY,
  LARGE_TEXT_CLASS,
  LARGE_TEXT_STORAGE_KEY,
  applyLargeText,
  readStoredFlag,
  storeFlag,
} from './preferences'

/**
 * The #99 preference store: same contract as #94's locale storage — every
 * touch guarded (a webview with storage disabled must never crash the
 * app), corrupt values read as "never chosen".
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

describe('readStoredFlag', () => {
  it('reads back what storeFlag wrote', () => {
    vi.stubGlobal('window', { localStorage: fakeStorage() })
    storeFlag(ACCESSIBLE_ONLY_STORAGE_KEY, true)
    storeFlag(LARGE_TEXT_STORAGE_KEY, false)
    expect(readStoredFlag(ACCESSIBLE_ONLY_STORAGE_KEY)).toBe(true)
    expect(readStoredFlag(LARGE_TEXT_STORAGE_KEY)).toBe(false)
  })

  it('reads corrupt values as never chosen', () => {
    vi.stubGlobal('window', { localStorage: fakeStorage({ [LARGE_TEXT_STORAGE_KEY]: 'yes' }) })
    expect(readStoredFlag(LARGE_TEXT_STORAGE_KEY)).toBeUndefined()
  })

  it('survives a webview whose storage access throws', () => {
    vi.stubGlobal('window', {
      get localStorage(): Storage {
        throw new Error('the document is sandboxed')
      },
    })
    expect(readStoredFlag(LARGE_TEXT_STORAGE_KEY)).toBeUndefined()
    expect(() => storeFlag(LARGE_TEXT_STORAGE_KEY, true)).not.toThrow()
  })
})

describe('applyLargeText', () => {
  it('toggles the token-scale class on the document root', () => {
    const toggle = vi.fn()
    vi.stubGlobal('document', { documentElement: { classList: { toggle } } })

    applyLargeText(true)
    expect(toggle).toHaveBeenCalledWith(LARGE_TEXT_CLASS, true)

    applyLargeText(false)
    expect(toggle).toHaveBeenCalledWith(LARGE_TEXT_CLASS, false)
  })
})
