import { afterEach, describe, expect, it, vi } from 'vitest'
import { setApiAuthToken, setApiShareToken } from '@/api/client'
import { QueryClient } from '@tanstack/react-query'
import type { SharedJourney } from './queries'
import { shareUrl, sharedJourneyQueryOptions } from './queries'

afterEach(() => {
  vi.unstubAllGlobals()
  setApiShareToken(undefined)
  setApiAuthToken(undefined)
})

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  })

// The queryFn ignores its context; the cast keeps the test call sites honest.
const sharedQueryFn = () => sharedJourneyQueryOptions(true).queryFn as () => Promise<SharedJourney>

describe('shareUrl', () => {
  it('encodes the visit id as the opaque token it is', () => {
    expect(shareUrl('VISIT 2')).toBe('/api/v1/journeys/VISIT%202/share')
    expect(shareUrl('VISIT-002')).toBe('/api/v1/journeys/VISIT-002/share')
  })
})

describe('sharedJourneyQueryOptions (#90)', () => {
  it('keys the cache by a constant — the token must never enter a query key', () => {
    expect(sharedJourneyQueryOptions(true).queryKey).toEqual(['shared-journey'])
    expect(sharedJourneyQueryOptions(false).queryKey).toEqual(['shared-journey'])
  })

  it('never fires without a token in the fragment', () => {
    expect(sharedJourneyQueryOptions(true).enabled).toBe(true)
    expect(sharedJourneyQueryOptions(false).enabled).toBe(false)
  })

  it('polls every 15s — even after DONE — and stops only when the link is dead', () => {
    const { refetchInterval, staleTime } = sharedJourneyQueryOptions(true)
    expect(staleTime).toBe(15_000)
    expect(refetchInterval).toBeTypeOf('function')
    const interval = refetchInterval as (
      query: { state: { data?: SharedJourney; error?: { status: number } } },
    ) => number | false | undefined
    expect(interval({ state: { data: { status: 'WAITING' } as SharedJourney } })).toBe(15_000)
    expect(interval({ state: { data: undefined } })).toBe(15_000)
    // DONE keeps polling so a revoke/expiry flips an open tab without a
    // reload — the live-revocation promise of the share flow.
    expect(interval({ state: { data: { status: 'DONE' } as SharedJourney } })).toBe(15_000)
    expect(interval({ state: { error: { status: 401 } } })).toBe(false)
    expect(interval({ state: { error: { status: 502 } } })).toBe(15_000)
  })

  it('sends only the share bearer — the patient session token must not ride along', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: 'WAITING' }))
    vi.stubGlobal('fetch', fetchMock)
    setApiShareToken('a'.repeat(64))
    setApiAuthToken('patient-session-token')

    await sharedQueryFn()()

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/shared/journey', {
      headers: { Authorization: `Bearer ${'a'.repeat(64)}` },
    })
  })

  it('fails before any request when the token slot is empty', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(sharedQueryFn()()).rejects.toMatchObject({ status: 401 })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('resolves through a real QueryClient so the option shape is mountable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ status: 'DONE' })))
    setApiShareToken('b'.repeat(64))

    const client = new QueryClient()
    const data = await client.fetchQuery(sharedJourneyQueryOptions(true))
    expect(data.status).toBe('DONE')
  })
})
