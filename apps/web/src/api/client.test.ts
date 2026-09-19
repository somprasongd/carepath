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
    expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/visits/V-1', {
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
    expect(url).toBe('http://localhost:8080/api/v1/journeys/V-1/steps/2/transition')
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
