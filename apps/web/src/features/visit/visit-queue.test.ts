import { afterEach, describe, expect, it, vi } from 'vitest'
import { visitQueueQueryOptions } from './queries'

/**
 * The #101 patient queue wiring: the journey screen reads the queue picture
 * from GET /api/v1/journeys/{visitId}/queue, keyed under the journey's key
 * so step transitions refresh it too.
 */

afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse() {
  return new Response(
    JSON.stringify({ visitId: 'VISIT-1', asOf: '2026-09-20T10:00:00Z', steps: [] }),
    { status: 200, headers: { 'content-type': 'application/json' } },
  )
}

// The queryFn ignores its context; the cast keeps the test call sites honest.
function queryFnOf(options: ReturnType<typeof visitQueueQueryOptions>) {
  return options.queryFn as () => Promise<unknown>
}

describe('visitQueueQueryOptions', () => {
  it('reads the visit queue endpoint with the visit claimed first', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = visitQueueQueryOptions('VISIT 1')
    await queryFnOf(options)()

    // The claim POST (#96) fires before the read, same as the journey query.
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/journeys/VISIT%201/claim')
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/journeys/VISIT%201/queue')
    expect(options.queryKey).toEqual(['journey', 'VISIT 1', 'queue'])
  })
})
