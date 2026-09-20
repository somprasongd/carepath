import { afterEach, describe, expect, it, vi } from 'vitest'
import { planningRulesQueryOptions } from './queries'

/**
 * The #100 planning-rules wiring: the staff screen reads the rules the
 * planner actually runs on from GET /api/v1/staff/planning-rules — one
 * static URL, one stable key.
 */

afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse() {
  return new Response(JSON.stringify({ phases: [], orderTypes: [], examples: [] }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })
}

// The queryFn ignores its context; the cast keeps the test call sites honest.
function queryFnOf(options: ReturnType<typeof planningRulesQueryOptions>) {
  return options.queryFn as () => Promise<unknown>
}

describe('planningRulesQueryOptions', () => {
  it('reads the staff planning-rules endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = planningRulesQueryOptions()
    await queryFnOf(options)()

    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/staff/planning-rules')
    expect(options.queryKey).toEqual(['staff', 'planning-rules'])
  })
})
