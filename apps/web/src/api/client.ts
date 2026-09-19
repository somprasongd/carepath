/**
 * Thin fetch client for the CarePath API. Response types come from the
 * contract-generated schema.d.ts — never hand-write a shape that exists there.
 */

const apiBase = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

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

export async function apiPost<TReturn>(path: string, body: unknown): Promise<TReturn> {
  return request(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
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
 * with the pair, or null when no token is stored or the server refuses it.
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
    } catch {
      clearStaffSession()
      return null
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

async function request<TReturn>(path: string, init: RequestInit, retried = false): Promise<TReturn> {
  const response = await doFetch(path, withAuthHeader(init))

  // A staff request whose access token expired: try one silent rotation and
  // replay the call. The auth endpoints themselves are exempt — a failed
  // login must surface, not trigger a refresh loop.
  if (
    response.status === 401 &&
    staffAccessToken &&
    !retried &&
    !path.startsWith('/api/v1/auth/')
  ) {
    if (await refreshStaffSession()) {
      return request(path, init, true)
    }
    staffSessionExpired?.()
  }

  if (!response.ok) {
    throw await apiErrorFrom(response)
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
