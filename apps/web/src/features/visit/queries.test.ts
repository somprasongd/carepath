import { afterEach, describe, expect, it, vi } from 'vitest'
import { setApiAuthToken } from '@/api/client'
import type { Journey } from './queries'
import {
  createVisitTokenSession,
  ensureVisitClaimed,
  journeyQueryOptions,
  transitionStepUrl,
} from './queries'

afterEach(() => {
  vi.unstubAllGlobals()
  // The QR bootstrap test adopts a visit-scoped bearer; leave the client
  // tokenless so later suites' exact-args assertions see bare requests.
  setApiAuthToken(undefined)
})

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  })

function journey(overrides: Partial<Journey> = {}): Journey {
  return {
    visitId: 'VISIT-001',
    patientRef: 'PATIENT-DEMO-001',
    status: 'ACTIVE',
    completed: false,
    steps: [],
    actionable: [],
    recommended: null,
    syncedAt: '2026-09-20T02:00:00Z',
    ...overrides,
  }
}

// The interval is computed from the cached journey, so the test calls it the
// way TanStack does — with the query's current state.
const intervalFor = (data: Journey | undefined) => {
  const { refetchInterval } = journeyQueryOptions('VISIT-001')
  expect(refetchInterval).toBeTypeOf('function')
  const interval = refetchInterval as (query: { state: { data?: Journey } }) =>
    | number
    | false
    | undefined
  return interval({ state: { data } })
}

describe('transitionStepUrl (#38 staff controls)', () => {
  it('keeps the stepKey colons raw — the API route param does not decode %3A', () => {
    expect(transitionStepUrl('VISIT-002', 'CLINIC:MED:1')).toBe(
      '/api/v1/journeys/VISIT-002/steps/CLINIC:MED:1/transition',
    )
    expect(transitionStepUrl('VISIT-002', 'XRAY:1')).toBe(
      '/api/v1/journeys/VISIT-002/steps/XRAY:1/transition',
    )
  })

  it('still encodes the visit id, which is a plain opaque token', () => {
    expect(transitionStepUrl('VISIT 2', 'REGISTRATION')).toBe(
      '/api/v1/journeys/VISIT%202/steps/REGISTRATION/transition',
    )
  })
})

describe('journeyQueryOptions realtime (#36)', () => {
  it('polls the journey every 15s while the visit can still change', () => {
    expect(intervalFor(journey())).toBe(15_000)
  })

  it('keeps polling before the first payload lands', () => {
    expect(intervalFor(undefined)).toBe(15_000)
  })

  it('stops polling once the visit is final — completed or cancelled', () => {
    expect(intervalFor(journey({ completed: true }))).toBe(false)
    expect(intervalFor(journey({ status: 'CANCELLED' }))).toBe(false)
  })

  it('keys the query by visit alone, so every screen reads one cache entry', () => {
    expect(journeyQueryOptions('VISIT-001').queryKey).toEqual(['journey', 'VISIT-001'])
    expect(journeyQueryOptions('VISIT-001').queryKey).toEqual(
      journeyQueryOptions('VISIT-001').queryKey,
    )
  })
})

describe('ensureVisitClaimed (#96 visit claim)', () => {
  it('POSTs the claim once per visit id per page session — polls and parallel callers share it', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await Promise.all([
      ensureVisitClaimed('VISIT-CLAIM-A'),
      ensureVisitClaimed('VISIT-CLAIM-A'),
    ])
    await ensureVisitClaimed('VISIT-CLAIM-A')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/journeys/VISIT-CLAIM-A/claim', {
      method: 'POST',
    })
  })

  it('claims per visit id — another visit still fires its own claim', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await ensureVisitClaimed('VISIT-CLAIM-A2')
    await ensureVisitClaimed('VISIT-CLAIM-B2')

    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('propagates the claim failure — an unprojected visit 404s like the read does', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'journey not found' }, 404)),
    )

    const failure = await ensureVisitClaimed('VISIT-CLAIM-C2').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(Error)
    expect((failure as { status?: number }).status).toBe(404)
  })

  it('retries a claim that failed — one transient failure must not wedge the visit', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: 'internal server error' }, 500))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    // First claim fails (e.g. a blip mid-deploy) — the query shows the error,
    // and its retry must POST again rather than skip the claim forever.
    const failure = await ensureVisitClaimed('VISIT-CLAIM-E2').catch((e: unknown) => e)
    expect((failure as { status?: number }).status).toBe(500)
    await expect(ensureVisitClaimed('VISIT-CLAIM-E2')).resolves.toBeUndefined()
    // Now it is remembered — later calls read without another claim POST.
    await ensureVisitClaimed('VISIT-CLAIM-E2')
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('the journey queryFn claims before it reads', async () => {
    const fetchMock = vi.fn().mockImplementation((path: string) => {
      if (path.endsWith('/claim')) {
        return Promise.resolve(new Response(null, { status: 204 }))
      }
      return Promise.resolve(jsonResponse(journey({ visitId: 'VISIT-CLAIM-D' })))
    })
    vi.stubGlobal('fetch', fetchMock)

    const queryFn = journeyQueryOptions('VISIT-CLAIM-D').queryFn as () => Promise<Journey>
    await expect(queryFn()).resolves.toMatchObject({ visitId: 'VISIT-CLAIM-D' })
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      '/api/v1/journeys/VISIT-CLAIM-D/claim',
      '/api/v1/journeys/VISIT-CLAIM-D',
    ])
  })

  it('treats a non-journey 404 from the claim as a retired route — remembered, not retried', async () => {
    // Mint live ⇒ claim-by-name is unregistered: fiber's own 404, no error
    // envelope. Swallowing it keeps the read as the surface of record, and
    // remembering it stops every 15s poll from re-firing the dead POST.
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response('Cannot POST /api/v1/journeys/VISIT-GONE/claim', { status: 404 }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(ensureVisitClaimed('VISIT-GONE')).resolves.toBeUndefined()
    await ensureVisitClaimed('VISIT-GONE')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})

describe('createVisitTokenSession (#136 QR-only front door)', () => {
  it('bootstraps a claimed, visit-scoped session and rides it as the patient bearer', async () => {
    const fetchMock = vi.fn().mockImplementation((path: string) => {
      if (path === '/api/v1/auth/session') {
        return Promise.resolve(
          jsonResponse({
            sessionToken: 'sess-visit-1',
            visitId: 'VISIT-QR-1',
            identity: { source: 'visit', externalId: 'visit:VISIT-QR-1' },
            expiresAt: '2026-09-20T03:00:00Z',
          }),
        )
      }
      return Promise.resolve(jsonResponse(journey({ visitId: 'VISIT-QR-1' })))
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(createVisitTokenSession('slip-token')).resolves.toBe('VISIT-QR-1')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source: 'visit-token', idToken: 'slip-token' }),
    })

    // The server claimed before answering: the journey read goes straight
    // through under the new bearer — no claim POST, no wedging.
    const queryFn = journeyQueryOptions('VISIT-QR-1').queryFn as () => Promise<Journey>
    await expect(queryFn()).resolves.toMatchObject({ visitId: 'VISIT-QR-1' })
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      '/api/v1/auth/session',
      '/api/v1/journeys/VISIT-QR-1',
    ])
    const read = fetchMock.mock.calls.find(([path]) => path === '/api/v1/journeys/VISIT-QR-1')
    expect(read?.[1]).toMatchObject({ headers: { Authorization: 'Bearer sess-visit-1' } })
  })

  it('surfaces the token 404 — the exchange screen shows its invalid state', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ error: 'journey not found' }, 404)),
    )

    const failure = await createVisitTokenSession('garbage').catch((e: unknown) => e)
    expect(failure).toBeInstanceOf(Error)
    expect((failure as { status?: number }).status).toBe(404)
  })
})
