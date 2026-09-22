import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  navigationRouteQueryOptions,
  nearbyAmenitiesQueryOptions,
  placeRouteQueryOptions,
} from './queries'

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

const amenityResponse = () =>
  new Response(JSON.stringify({ from: 'x', amenities: [] }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })

// The queryFn ignores its context; the cast keeps the test call sites honest.
function queryFnOf<T extends { queryFn?: unknown }>(options: T) {
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

describe('placeRouteQueryOptions', () => {
  // The amenity destination's route (#109): `toPlace` names a place id — it
  // must ride its own key so it never collides with a same-endpoint service
  // point route, and the URL uses the param the place route expects.
  it('routes by place id and keys it separately from the code route', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = placeRouteQueryOptions('I-1301/node-reception', 'RESTROOM-01')
    await queryFnOf(options)()

    expect(lastUrl()).toBe('/api/v1/navigation/route?from=I-1301%2Fnode-reception&toPlace=RESTROOM-01')
    expect(options.queryKey).toEqual([
      'navigation',
      'route',
      'I-1301/node-reception',
      'place',
      'RESTROOM-01',
      false,
    ])
  })

  it('sends accessibleOnly=true only when the flag is on', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = placeRouteQueryOptions('I-1301/node-reception', 'RESTROOM-01', true)
    await queryFnOf(options)()

    expect(lastUrl()).toBe(
      '/api/v1/navigation/route?from=I-1301%2Fnode-reception&toPlace=RESTROOM-01&accessibleOnly=true',
    )
  })
})

describe('nearbyAmenitiesQueryOptions', () => {
  it('asks from the patient node and stays param-free by default', async () => {
    const fetchMock = vi.fn().mockResolvedValue(amenityResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = nearbyAmenitiesQueryOptions('I-1301/node-reception')
    await queryFnOf(options)()

    expect(lastUrl()).toBe('/api/v1/navigation/amenities?from=I-1301%2Fnode-reception')
    expect(options.queryKey).toEqual(['navigation', 'amenities', 'I-1301/node-reception', false])
  })

  it('sends and re-keys the accessible flag like the route queries do', async () => {
    const fetchMock = vi.fn().mockResolvedValue(amenityResponse())
    vi.stubGlobal('fetch', fetchMock)

    const options = nearbyAmenitiesQueryOptions('I-1301/node-reception', true)
    await queryFnOf(options)()

    expect(lastUrl()).toBe(
      '/api/v1/navigation/amenities?from=I-1301%2Fnode-reception&accessibleOnly=true',
    )
    expect(options.queryKey).toEqual(['navigation', 'amenities', 'I-1301/node-reception', true])
  })
})
