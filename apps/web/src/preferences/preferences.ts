/**
 * Device-local patient preferences (#99, FR-20): the accessible-route flag
 * and large-text mode. Same rules as the locale preference (#94,
 * i18n/initial-locale.ts): best-effort localStorage — some in-app webviews
 * ship with storage disabled and throw on access, and no preference is
 * ever worth crashing the app over — and never a property of the visit, so
 * it must never ride the session token (ADR-0010). One mechanism for every
 * boolean preference; don't grow a second one.
 */

/** localStorage keys, namespaced like 'carepath.locale'. */
export const ACCESSIBLE_ONLY_STORAGE_KEY = 'carepath.accessibleOnly'
export const LARGE_TEXT_STORAGE_KEY = 'carepath.largeText'

/** The class large-text mode toggles on <html>; styles/index.css keys the
 *  token overrides off it (tokens, not point overrides — apps/web rules). */
export const LARGE_TEXT_CLASS = 'carepath-large-text'

/**
 * The stored choice, or undefined for "never chosen / unreadable". Anything
 * other than the exact strings 'true'/'false' counts as never chosen.
 */
export function readStoredFlag(key: string): boolean | undefined {
  try {
    const value = window.localStorage.getItem(key)
    if (value === 'true') return true
    if (value === 'false') return false
    return undefined
  } catch {
    return undefined
  }
}

/** Remember the choice; storage being unavailable just means no memory. */
export function storeFlag(key: string, value: boolean): void {
  try {
    window.localStorage.setItem(key, String(value))
  } catch {
    // Best effort by design.
  }
}

/**
 * Apply (or clear) large-text mode on the document root. Pure DOM — the
 * visual change comes entirely from the CSS token overrides under
 * `html.carepath-large-text`, so no React re-render is involved. Boot calls
 * this before the first render (main.tsx) so there is no flash of small
 * text; the toggle component calls it on click.
 */
export function applyLargeText(enabled: boolean): void {
  document.documentElement.classList.toggle(LARGE_TEXT_CLASS, enabled)
}
