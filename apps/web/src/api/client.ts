/**
 * Thin fetch client for the CarePath API. Response types come from the
 * contract-generated schema.d.ts — never hand-write a shape that exists there.
 */

// Same-origin by default: in dev the Vite server proxies /api to the backend
// (vite.config.ts), and in production the nginx edge serves web + /api from
// one origin — so a relative path reaches the API in both. Set
// VITE_API_BASE_URL only to target an API on a different origin.
const apiBase = import.meta.env.VITE_API_BASE_URL ?? ''

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function apiGet<TReturn>(path: string): Promise<TReturn> {
  return request(path, { method: 'GET' })
}

/**
 * GET a non-JSON body — the floor plan SVG (ADR-0015), which is served as
 * `image/svg+xml` rather than wrapped in JSON. Goes through `request` like
 * every other authenticated call rather than its own doFetch, so it gets
 * the same silent rotate-and-replay on a 401 as the rest of the client:
 * this route is unguarded today, but the helper is general-purpose, and a
 * future guarded text endpoint should not have to rediscover that gap.
 */
export async function apiGetText(path: string): Promise<string> {
  return request(path, { method: 'GET' }, false, 'text')
}

/**
 * POST a floor plan (ADR-0015). The upload endpoint takes the SVG as the
 * raw body rather than a multipart field — the file *is* the request, and
 * the server rejects anything that is not a plan long before content-type
 * would have told it apart.
 */
export async function apiPostSvg<TReturn>(path: string, svg: string): Promise<TReturn> {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'image/svg+xml' },
    body: svg,
  })
}

export async function apiPost<TReturn>(path: string, body: unknown): Promise<TReturn> {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function apiPut<TReturn>(path: string, body: unknown): Promise<TReturn> {
  return request(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

/**
 * DELETE expecting no body (204 today: stop-sharing, #89). Goes through the
 * patient bearer like the other patient-surface commands.
 */
export async function apiDelete(path: string): Promise<void> {
  const response = await doFetch(path, withAuthHeader({ method: 'DELETE' }))
  if (!response.ok) {
    throw await apiErrorFrom(response)
  }
}

/**
 * Body-less POST expecting no body (204 today: the visit claim, #96). Same
 * bearer rule as apiDelete — a patient-surface command, so the session token
 * rides it; apiPost cannot be used because it both sends and parses JSON.
 */
export async function apiPostNoContent(path: string): Promise<void> {
  const response = await doFetch(path, withAuthHeader({ method: 'POST' }))
  if (!response.ok) {
    throw await apiErrorFrom(response)
  }
}

/**
 * Body-less POST that returns JSON (the floor-plan activate/rollback call,
 * ADR-0015 — there is nothing to upload, only a plan id already in the
 * path). apiPostNoContent's returning sibling: same body-less shape, but
 * goes through `request` for the response parsing and the 401 retry rather
 * than being handed a document body through apiPostSvg.
 */
export async function apiPostAction<TReturn>(path: string): Promise<TReturn> {
  return request(path, { method: 'POST' })
}

// ---------------------------------------------------------------------------
// Share token (ADR-0011 §5) — the third credential, and the only one the
// /shared surface may carry. It arrives in the URL fragment of /shared
// (browsers never send fragments to a server) and is read once by the shared
// screen. It must not ride the generic bearer below: the patient/staff
// tokens are invalid on the shared endpoint, and the share token is invalid
// everywhere else — the cross-token rule holds in every direction.

let shareToken: string | undefined

export function setApiShareToken(token: string | undefined) {
  shareToken = token
}

/**
 * GET with the share token alone. Throws before any request when no token is
 * stored, so the shared screen can show its "no link" state instead of
 * firing a request that could only 401.
 */
export async function apiGetShared<TReturn>(path: string): Promise<TReturn> {
  if (!shareToken) {
    throw new ApiError(401, 'ไม่มีโทเคนของลิงก์')
  }
  const response = await doFetch(path, { headers: { Authorization: `Bearer ${shareToken}` } })
  if (!response.ok) {
    throw await apiErrorFrom(response)
  }
  return (await response.json()) as TReturn
}

// ---------------------------------------------------------------------------
// Token slots (ADR-0010 §12)
//
// The app has two surfaces with two different bearers: the patient LIFF
// surface (session token from /auth/session) and the staff console (JWT
// access token from /auth/login). Only this file ever touches them. The
// access token lives in memory only; the staff refresh token is the one
// secret persisted (localStorage), so a console reload can restore the
// session without re-typing the password.

let patientToken: string | undefined
let staffAccessToken: string | undefined

const STAFF_REFRESH_KEY = 'carepath.staff.refreshToken'

function storage(): Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | undefined {
  return typeof localStorage === 'undefined' ? undefined : localStorage
}

/**
 * Set (or clear) the patient session token attached as `Authorization:
 * Bearer` to every request. The auth providers own this — LiffAuthProvider
 * stores the server-issued token, DemoAuthProvider its placeholder.
 */
export function setApiAuthToken(token: string | undefined) {
  patientToken = token
}

/** The staff token pair as the API returns it (contract: AuthTokens). */
export interface StaffTokenPair {
  accessToken: string
  refreshToken: string
  identity: StaffIdentity
}

export interface StaffIdentity {
  userId: string
  username: string
  displayName: string
  roles: string[]
}

/** Store a freshly issued staff pair: access token in memory, refresh persisted. */
export function adoptStaffTokens(pair: StaffTokenPair) {
  staffAccessToken = pair.accessToken
  storage()?.setItem(STAFF_REFRESH_KEY, pair.refreshToken)
}

/** The persisted staff refresh token, or null when no session is restorable. */
export function loadStaffRefreshToken(): string | null {
  return storage()?.getItem(STAFF_REFRESH_KEY) ?? null
}

/** Forget the staff session entirely (logout, or refresh that failed for good). */
export function clearStaffSession() {
  staffAccessToken = undefined
  storage()?.removeItem(STAFF_REFRESH_KEY)
}

/**
 * Drop the in-memory staff access token while keeping the persisted refresh
 * token — called when the staff console unmounts. Since #96 the patient
 * surface needs its own session bearer, and the slots prefer the staff
 * token when both are set; leaving /staff for /patient in one tab would
 * otherwise send a staff JWT to patient routes (401). Returning to the
 * console re-exchanges the refresh token, so the session survives the trip.
 */
export function releaseStaffAccessToken() {
  staffAccessToken = undefined
}

// Called when the staff session is gone for good (refresh refused). The
// StaffAuthProvider registers the UI half — flip its state and head to
// /login — so this module never touches the router.
let staffSessionExpired: (() => void) | undefined

export function onStaffSessionExpired(handler: (() => void) | undefined) {
  staffSessionExpired = handler
}

let refreshInFlight: Promise<StaffTokenPair | null> | undefined

/**
 * Exchange the persisted refresh token for a new pair, once at a time: every
 * concurrent caller (a StrictMode double-mount, several 401s landing
 * together) shares one round-trip, because spending a refresh token twice
 * trips the server's reuse detector and burns the whole session. Resolves
 * with the pair, or null when no token is stored or the server refuses it
 * (401 — unknown, expired, revoked, or already-spent). Any other failure
 * (offline, 5xx) is transient: it rethrows with the session intact, so the
 * next attempt retries the exchange instead of destroying it.
 */
function refreshStaffSession(): Promise<StaffTokenPair | null> {
  refreshInFlight ??= (async () => {
    const refreshToken = loadStaffRefreshToken()
    if (!refreshToken) return null
    try {
      const pair = await rawRequest<StaffTokenPair>('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken }),
      })
      adoptStaffTokens(pair)
      return pair
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        clearStaffSession()
        return null
      }
      throw err
    }
  })().finally(() => {
    refreshInFlight = undefined
  })
  return refreshInFlight
}

/**
 * Restore the staff session from the persisted refresh token (console boot,
 * reload). Single-flight by construction — see refreshStaffSession.
 */
export function restoreStaffSession(): Promise<StaffTokenPair | null> {
  return refreshStaffSession()
}

function bearerToken(): string | undefined {
  // The two surfaces never run in the same tab for long; when they do, the
  // staff console is the one talking to guarded endpoints.
  return staffAccessToken ?? patientToken
}

function withAuthHeader(init: RequestInit): RequestInit {
  const token = bearerToken()
  if (!token) return init
  const headers: Record<string, string> = {
    ...(init.headers as Record<string, string> | undefined),
    Authorization: `Bearer ${token}`,
  }
  return { ...init, headers }
}

async function rawRequest<TReturn>(path: string, init: RequestInit): Promise<TReturn> {
  const response = await doFetch(path, init)
  if (!response.ok) {
    throw await apiErrorFrom(response)
  }
  return (await response.json()) as TReturn
}

async function request<TReturn>(
  path: string,
  init: RequestInit,
  retried = false,
  // 'text' for a non-JSON body such as the floor-plan SVG (apiGetText):
  // still goes through the 401 rotate-and-replay below, which a fetch
  // outside this function would not get.
  parseAs: 'json' | 'text' = 'json',
): Promise<TReturn> {
  const response = await doFetch(path, withAuthHeader(init))

  // A staff request whose access token expired: try one silent rotation and
  // replay the call. The auth endpoints themselves are exempt — a failed
  // login must surface, not trigger a refresh loop. A refused rotation ends
  // the session; a transient one (offline, 5xx) propagates as the network
  // error it is, with the session intact for the next attempt.
  if (
    response.status === 401 &&
    staffAccessToken &&
    !retried &&
    !path.startsWith('/api/v1/auth/')
  ) {
    if (await refreshStaffSession()) {
      return request(path, init, true, parseAs)
    }
    staffSessionExpired?.()
  }

  if (!response.ok) {
    throw await apiErrorFrom(response)
  }
  if (parseAs === 'text') {
    return (await response.text()) as TReturn
  }
  return (await response.json()) as TReturn
}

async function doFetch(path: string, init: RequestInit): Promise<Response> {
  try {
    return await fetch(`${apiBase}${path}`, init)
  } catch (cause) {
    throw new ApiError(0, `เชื่อมต่อ API ไม่ได้ (${String(cause)})`)
  }
}

async function apiErrorFrom(response: Response): Promise<ApiError> {
  let detail = `${response.status}`
  try {
    const body = (await response.json()) as { error?: string }
    if (body.error) detail = body.error
  } catch {
    // Non-JSON error body — keep the status code as the detail.
  }
  return new ApiError(response.status, detail)
}
