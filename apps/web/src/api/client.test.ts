import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  ApiError,
  adoptStaffTokens,
  apiGet,
  apiPost,
  clearStaffSession,
  loadStaffRefreshToken,
  onStaffSessionExpired,
  restoreStaffSession,
  setApiAuthToken,
} from './client'

afterEach(() => {
  vi.unstubAllGlobals()
})

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  })

describe('apiGet', () => {
  it('returns the parsed JSON body on success', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ visitId: 'V-1' })))

    await expect(apiGet<{ visitId: string }>('/api/v1/visits/V-1')).resolves.toEqual({
      visitId: 'V-1',
    })
    expect(fetch).toHaveBeenCalledWith('/api/v1/visits/V-1', {
      method: 'GET',
    })
  })

  it('surfaces the API error message for a non-OK response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'visit not found' }, 404)),
    )

    const failure = await apiGet('/api/v1/visits/NOPE').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(404)
    expect((failure as ApiError).message).toBe('visit not found')
  })

  it('falls back to the status code when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('gateway timeout', { status: 504 })),
    )

    const failure = await apiGet('/api/v1/visits/V-1').catch((e: unknown) => e)
    expect((failure as ApiError).status).toBe(504)
    expect((failure as ApiError).message).toBe('504')
  })

  it('wraps a network failure as status 0', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('ECONNREFUSED')))

    const failure = await apiGet('/api/v1/visits/V-1').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(0)
  })
})

describe('apiPost', () => {
  it('sends a JSON body and returns the parsed response on success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ sessionToken: 'tok-1' })),
    )

    await expect(
      apiPost<{ sessionToken: string }>('/api/v1/auth/session', { source: 'demo', idToken: '' }),
    ).resolves.toEqual({ sessionToken: 'tok-1' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source: 'demo', idToken: '' }),
    })
  })

  it('surfaces the API error message for a non-OK response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'invalid identity token' }, 401)),
    )

    const failure = await apiPost('/api/v1/auth/session', {}).catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(401)
    expect((failure as ApiError).message).toBe('invalid identity token')
  })

  it('wraps a network failure as status 0', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('ECONNREFUSED')))

    const failure = await apiPost('/api/v1/auth/session', {}).catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(0)
  })

  it('sends JSON and returns the parsed body', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ visitId: 'V-1', status: 'ACTIVE' }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      apiPost<{ visitId: string }>('/api/v1/journeys/V-1/steps/2/transition', {
        to: 'STARTED',
        source: 'staff-web',
      }),
    ).resolves.toEqual({ visitId: 'V-1', status: 'ACTIVE' })

    const [url, init] = vi.mocked(fetchMock).mock.calls[0]
    expect(url).toBe('/api/v1/journeys/V-1/steps/2/transition')
    expect(init?.method).toBe('POST')
    expect(new Headers(init?.headers).get('content-type')).toBe('application/json')
    expect(JSON.parse(String(init?.body))).toEqual({ to: 'STARTED', source: 'staff-web' })
  })

  it('surfaces the API error message for a rejected transition', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'step is not READY' }, 409)),
    )

    const failure = await apiPost('/api/v1/journeys/V-1/steps/9/transition', {
      to: 'COMPLETED',
    }).catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(409)
    expect((failure as ApiError).message).toBe('step is not READY')
  })
})

describe('setApiAuthToken', () => {
  afterEach(() => {
    setApiAuthToken(undefined)
  })

  it('attaches the session token as a Bearer header on GET', async () => {
    setApiAuthToken('tok-1')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ source: 'line' })))

    await apiGet('/api/v1/auth/session')
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/session', {
      method: 'GET',
      headers: { Authorization: 'Bearer tok-1' },
    })
  })

  it('merges the Bearer header with the per-call headers on POST', async () => {
    setApiAuthToken('tok-1')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ visitId: 'V-1' })))

    await apiPost('/api/v1/journeys/V-1/steps/1/transition', { to: 'STARTED' })
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/journeys/V-1/steps/1/transition',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: 'Bearer tok-1' },
        body: JSON.stringify({ to: 'STARTED' }),
      },
    )
  })

  it('omits the Authorization header once the token is cleared', async () => {
    setApiAuthToken('tok-1')
    setApiAuthToken(undefined)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ visitId: 'V-1' })))

    await apiGet('/api/v1/journeys/V-1')
    const init = vi.mocked(fetch).mock.calls[0]?.[1]
    expect(new Headers(init?.headers).get('authorization')).toBeNull()
  })
})

describe('staff session surface', () => {
  afterEach(() => {
    setApiAuthToken(undefined)
    clearStaffSession()
    onStaffSessionExpired(undefined)
    vi.unstubAllGlobals()
  })

  const staffPair = (access: string, refresh = 'refresh-1') => ({
    accessToken: access,
    refreshToken: refresh,
    identity: { userId: 'user-1', username: 'staff', displayName: 'Demo Staff', roles: ['STAFF'] },
  })

  // The node test environment has no localStorage; a tiny memory stub stands in.
  const memoryStorage = () => {
    const map = new Map<string, string>()
    return {
      getItem: (k: string) => map.get(k) ?? null,
      setItem: (k: string, v: string) => void map.set(k, v),
      removeItem: (k: string) => void map.delete(k),
    }
  }

  it('adopts the staff pair: access token attached, refresh token persisted', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1'))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ visits: [] })))

    await apiGet('/api/v1/staff/visits')
    const init = vi.mocked(fetch).mock.calls[0]?.[1]
    expect(new Headers(init?.headers).get('authorization')).toBe('Bearer staff-access-1')
    expect(loadStaffRefreshToken()).toBe('refresh-1')
  })

  it('prefers the staff token while both surfaces are signed in', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    setApiAuthToken('patient-token')
    adoptStaffTokens(staffPair('staff-access-1'))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ visits: [] })))

    await apiGet('/api/v1/staff/visits')
    const init = vi.mocked(fetch).mock.calls[0]?.[1]
    expect(new Headers(init?.headers).get('authorization')).toBe('Bearer staff-access-1')
  })

  it('rotates through one refresh on a 401 and replays the request with the new token', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'missing or invalid access token' }, 401))
      .mockResolvedValueOnce(jsonResponse(staffPair('staff-access-2', 'refresh-2'), 200))
      .mockResolvedValueOnce(jsonResponse({ visits: ['V-1'] }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await apiGet('/api/v1/staff/visits')
    expect(result).toEqual({ visits: ['V-1'] })
    expect(fetchMock).toHaveBeenCalledTimes(3)
    // The replayed call carries the rotated access token…
    const replayInit = fetchMock.mock.calls[2]?.[1]
    expect(new Headers(replayInit?.headers).get('authorization')).toBe('Bearer staff-access-2')
    // …and the new refresh token replaced the spent one.
    expect(loadStaffRefreshToken()).toBe('refresh-2')
  })

  it('shares one refresh across concurrent 401s', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockResolvedValueOnce(jsonResponse(staffPair('staff-access-2', 'refresh-2')))
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
    vi.stubGlobal('fetch', fetchMock)

    await Promise.all([apiGet('/api/v1/staff/visits'), apiPost('/api/v1/journeys/V-1/steps/S/transition', { to: 'STARTED' })])
    // Two failing calls + exactly one refresh + two replays.
    expect(fetchMock).toHaveBeenCalledTimes(5)
  })

  it('clears the session and fires the expiry hook when the refresh is refused', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const expired = vi.fn()
    onStaffSessionExpired(expired)
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockResolvedValueOnce(jsonResponse({ error: 'unknown refresh token' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    const failure = await apiGet('/api/v1/staff/visits').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(401)
    expect(expired).toHaveBeenCalledTimes(1)
    expect(loadStaffRefreshToken()).toBeNull()
  })

  it('keeps the session when the refresh fails transiently (offline, 5xx)', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const expired = vi.fn()
    onStaffSessionExpired(expired)
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockRejectedValueOnce(new Error('ECONNREFUSED'))
    vi.stubGlobal('fetch', fetchMock)

    // A network blip must not log the staff user out: the transport error
    // surfaces, the expiry hook stays silent, the refresh token survives.
    const failure = await apiGet('/api/v1/staff/visits').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(0)
    expect(expired).not.toHaveBeenCalled()
    expect(loadStaffRefreshToken()).toBe('refresh-1')
  })

  it('recovers on the next attempt after a transient refresh failure', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockRejectedValueOnce(new Error('ECONNREFUSED'))
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockResolvedValueOnce(jsonResponse(staffPair('staff-access-2', 'refresh-2')))
      .mockResolvedValueOnce(jsonResponse({ visits: ['V-1'] }))
    vi.stubGlobal('fetch', fetchMock)

    const blip = await apiGet('/api/v1/staff/visits').catch((e: unknown) => e)
    expect((blip as ApiError).status).toBe(0)
    // Once the API is reachable again, the same call rotates and replays.
    await expect(apiGet('/api/v1/staff/visits')).resolves.toEqual({ visits: ['V-1'] })
    expect(loadStaffRefreshToken()).toBe('refresh-2')
  })

  it('a transient failure at boot rejects the restore but keeps the token', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    localStorage.setItem('carepath.staff.refreshToken', 'refresh-1')
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('ECONNREFUSED')))

    await expect(restoreStaffSession()).rejects.toBeInstanceOf(ApiError)
    expect(loadStaffRefreshToken()).toBe('refresh-1')
  })

  it('never retries a 401 on the auth endpoints themselves', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'invalid credentials' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    const failure = await apiPost('/api/v1/auth/login', { username: 'x', password: 'y' }).catch(
      (e: unknown) => e,
    )
    expect(failure).toBeInstanceOf(ApiError)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('retries a 401 at most once — the second 401 surfaces', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    adoptStaffTokens(staffPair('staff-access-1', 'refresh-1'))
    const expired = vi.fn()
    onStaffSessionExpired(expired)
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'expired' }, 401))
      .mockResolvedValueOnce(jsonResponse(staffPair('staff-access-2', 'refresh-2')))
      .mockResolvedValue(jsonResponse({ error: 'expired again' }, 401))
    vi.stubGlobal('fetch', fetchMock)

    const failure = await apiGet('/api/v1/staff/visits').catch((e: unknown) => e)
    expect((failure as ApiError).status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('restores a session from a persisted refresh token exactly once', async () => {
    vi.stubGlobal('localStorage', memoryStorage())
    localStorage.setItem('carepath.staff.refreshToken', 'refresh-1')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(staffPair('staff-access-9', 'refresh-9')))
      .mockResolvedValue(jsonResponse({ visits: [] }))
    vi.stubGlobal('fetch', fetchMock)

    // Two concurrent restores (StrictMode double-mount) share one refresh.
    const [a, b] = await Promise.all([restoreStaffSession(), restoreStaffSession()])
    expect(a).toEqual(b)
    expect(a?.accessToken).toBe('staff-access-9')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(loadStaffRefreshToken()).toBe('refresh-9')

    await apiGet('/api/v1/staff/visits')
    const init = fetchMock.mock.calls[1]?.[1]
    expect(new Headers(init?.headers).get('authorization')).toBe('Bearer staff-access-9')
  })
})
