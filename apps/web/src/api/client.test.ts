import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiGet, apiPost } from './client'

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
    expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/visits/V-1')
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
    expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/auth/session', {
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
})
