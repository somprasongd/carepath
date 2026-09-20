/**
 * The slip-held visit link (#136): `…/patient/journey#vt=<token>` — the token
 * rides the URL *fragment* so it never reaches an access log (ADR-0011 §5
 * precedent), and is moved into sessionStorage at bootstrap, before LIFF's
 * OAuth redirect can rewrite the URL and drop it. The exchange itself runs
 * inside the auth-gated journey route, where the session bearer exists.
 */
const VISIT_LINK_STORAGE_KEY = 'carepath:visit-link-token'

/**
 * Runs first thing at bootstrap (main.tsx), before `liff.init()`: capture
 * `#vt=` from the raw URL and strip the fragment. sessionStorage survives the
 * LIFF login redirect and same-tab reloads, but never leaks to another
 * patient on a shared device the way a URL would.
 */
export function stashVisitLinkToken(): void {
  const match = window.location.hash.match(/^#vt=([A-Za-z0-9_-]+)/)
  if (!match) return
  sessionStorage.setItem(VISIT_LINK_STORAGE_KEY, match[1])
  // Replace, not navigate: the fragment was never a route state, and the
  // token must not sit in the address bar for a screenshot to capture.
  window.history.replaceState(null, '', window.location.pathname + window.location.search)
}

export function readVisitLinkToken(): string | undefined {
  return sessionStorage.getItem(VISIT_LINK_STORAGE_KEY) ?? undefined
}

export function clearVisitLinkToken(): void {
  sessionStorage.removeItem(VISIT_LINK_STORAGE_KEY)
}
