import { queryOptions, useQuery } from '@tanstack/react-query'
import { ApiError, apiGet } from '@/api/client'
import type { components } from '@/api/schema'

export type LocationObservation = components['schemas']['LocationObservation']

/**
 * The visit's current location (ADR-0004 observation) — the routing start
 * point. No observation yet (404) is the normal first state of a visit, not
 * an error: the query resolves to null and the navigate screen falls back
 * to its destination-only view until a QR scan or Zigbee fix lands. The
 * short poll keeps the drawn route following location changes (#29 AC2)
 * even before the journey realtime pass (#36).
 */
export function currentLocationQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['journey', visitId, 'location'] as const,
    queryFn: async () => {
      try {
        return await apiGet<LocationObservation>(
          `/api/v1/journeys/${encodeURIComponent(visitId)}/location`,
        )
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null
        throw error
      }
    },
    staleTime: 10_000,
    refetchInterval: 15_000,
  })
}

export function useCurrentLocation(visitId: string) {
  return useQuery(currentLocationQueryOptions(visitId))
}

/**
 * The shortest walkable route between the current location's node and the
 * destination service point (#28) — the SVG overlay and turn cues (#29)
 * both render from this. Disabled until both ends are known; a 404 (no
 * walkable path) surfaces as the query's error for the screen to fall back
 * on, matching the honest no-route states in DESIGN.md.
 */
export function navigationRouteQueryOptions(fromNodeId: string | null, servicePointCode: string) {
  return queryOptions({
    queryKey: ['navigation', 'route', fromNodeId, servicePointCode] as const,
    queryFn: () =>
      apiGet<components['schemas']['NavigationRoute']>(
        `/api/v1/navigation/route?from=${encodeURIComponent(fromNodeId ?? '')}&to=${encodeURIComponent(servicePointCode)}`,
      ),
    enabled: fromNodeId !== null,
    staleTime: 15_000,
  })
}

export function useNavigationRoute(fromNodeId: string | null, servicePointCode: string) {
  return useQuery(navigationRouteQueryOptions(fromNodeId, servicePointCode))
}
