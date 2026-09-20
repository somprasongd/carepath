import { afterEach, describe, expect, it, vi } from 'vitest'
import { navigationRouteQueryOptions } from './queries'

/**
 * The #99 accessible-route wiring: the flag must ride both the query key
 * (so a flip refetches instead of serving the stairs route from cache) and
 * the URL — and only when on, so the default route's canonical URL stays
 * param-free.
 */

afterEach(() => {
  vi.unstubAllGlobals()
})

const jsonResponse = () =>
  new Response(JSON.stringify({ nodes: [], segments: [], totalDistance: 0 }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })

// The queryFn ignores its context; the cast keeps the test call sites honest.
function queryFnOf(options: ReturnType<typeof navigationRouteQueryOptions>) {
  return options.queryFn as () => Promise<unknown>
}

function lastUrl(): string {
  return vi.mocked(fetch).mock.calls[0][0] as string
}

describe('navigationRouteQueryOptions', () => {
  it('keeps the default URL param-free', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = navigationRouteQueryOptions('I-1301/node-reception', 'PHARMACY')
    await queryFnOf(options)()

    expect(lastUrl()).toBe('/api/v1/navigation/route?from=I-1301%2Fnode-reception&to=PHARMACY')
    expect(options.queryKey).toEqual([
      'navigation',
      'route',
      'I-1301/node-reception',
      'PHARMACY',
      false,
    ])
  })

  it('sends accessibleOnly=true and re-keys when the flag is on', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = navigationRouteQueryOptions(
      'I-1301/node-reception',
      'PHARMACY',
      true,
    )
    await queryFnOf(options)()

    expect(lastUrl()).toBe(
      '/api/v1/navigation/route?from=I-1301%2Fnode-reception&to=PHARMACY&accessibleOnly=true',
    )
    expect(options.queryKey).toEqual([
      'navigation',
      'route',
      'I-1301/node-reception',
      'PHARMACY',
      true,
    ])
  })
})
