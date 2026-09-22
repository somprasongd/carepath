import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiGet, apiPost } from '@/api/client'
import { ensureVisitClaimed } from '@/features/visit'
import type { components } from '@/api/schema'

export type LocationObservation = components['schemas']['LocationObservation']
export type AmenitySearch = components['schemas']['AmenitySearch']
export type AmenityNearby = components['schemas']['AmenityNearby']

/**
 * The visit's current location (ADR-0004 observation) — the routing start
 * point. No observation yet (404) is the normal first state of a visit, not
 * an error: the query resolves to null and the navigate screen falls back
 * to its destination-only view until a QR scan or Zigbee fix lands. The
 * short poll keeps the drawn route following location changes (#29 AC2)
 * even before the journey realtime pass (#36). The claim runs first (#96):
 * an unclaimed visit would also 404, and that 404 must not be read as
 * "no location yet".
 */
export function currentLocationQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['journey', visitId, 'location'] as const,
    queryFn: async () => {
      await ensureVisitClaimed(visitId)
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

/** The sources the visit's own report endpoint resolves through (ADR-0004). */
export type LocationReportSource = 'QR' | 'MANUAL'

/**
 * Report where the patient stands — a scanned QR string, or a manually picked
 * node id — and make it the current location (#32/#33's write half). The 200
 * body *is* the observation, so it lands straight in the location cache; a new
 * nodeId changes the route query's key, and the walking line plus the
 * "you are here" mark redraw by themselves (#29 AC2) — no reload, no refetch
 * wait for the poll.
 */
export function useReportLocation(visitId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: { source: LocationReportSource; raw: string }) => {
      await ensureVisitClaimed(visitId)
      return apiPost<LocationObservation>(
        `/api/v1/journeys/${encodeURIComponent(visitId)}/location`,
        input,
      )
    },
    onSuccess: (observation) => {
      queryClient.setQueryData(
        currentLocationQueryOptions(visitId).queryKey,
        observation,
      )
    },
  })
}

/**
 * The shortest walkable route between the current location's node and the
 * destination service point (#28) — the SVG overlay and turn cues (#29)
 * both render from this. Disabled until both ends are known; a 404 (no
 * walkable path) surfaces as the query's error for the screen to fall back
 * on, matching the honest no-route states in DESIGN.md.
 *
 * accessibleOnly (#99, FR-20) opts the route out of the stairs: the flag
 * joins the query key, so flipping it refetches and the line redraws via
 * the elevator by itself. Sent only when on — the canonical URL for the
 * default route stays param-free.
 */
export function navigationRouteQueryOptions(
  fromNodeId: string | null,
  servicePointCode: string | null,
  accessibleOnly = false,
) {
  return queryOptions({
    queryKey: ['navigation', 'route', fromNodeId, servicePointCode, accessibleOnly] as const,
    queryFn: () =>
      apiGet<components['schemas']['NavigationRoute']>(
        `/api/v1/navigation/route?from=${encodeURIComponent(fromNodeId ?? '')}&to=${encodeURIComponent(servicePointCode ?? '')}${accessibleOnly ? '&accessibleOnly=true' : ''}`,
      ),
    // Null code = no service-point destination in play (screen still
    // loading, or a toPlace amenity destination owns the route instead,
    // #109) — better parked than asked with an empty `to`.
    enabled: fromNodeId !== null && servicePointCode !== null,
    staleTime: 15_000,
  })
}

export function useNavigationRoute(
  fromNodeId: string | null,
  servicePointCode: string | null,
  accessibleOnly = false,
) {
  return useQuery(navigationRouteQueryOptions(fromNodeId, servicePointCode, accessibleOnly))
}

/**
 * The shortest walkable route to a place rather than a service point (#109)
 * — how an amenity from /navigation/amenities is navigated: amenity places
 * have no service point code, so the destination rides `toPlace` instead of
 * `to`. Same honesty rules as the code route: disabled until the origin is
 * known, a 404 (no walkable path) is the query's error.
 */
export function placeRouteQueryOptions(
  fromNodeId: string | null,
  placeId: string | null,
  accessibleOnly = false,
) {
  return queryOptions({
    queryKey: ['navigation', 'route', fromNodeId, 'place', placeId, accessibleOnly] as const,
    queryFn: () =>
      apiGet<components['schemas']['NavigationRoute']>(
        `/api/v1/navigation/route?from=${encodeURIComponent(fromNodeId ?? '')}&toPlace=${encodeURIComponent(placeId ?? '')}${accessibleOnly ? '&accessibleOnly=true' : ''}`,
      ),
    enabled: fromNodeId !== null && placeId !== null,
    staleTime: 15_000,
  })
}

export function usePlaceRoute(
  fromNodeId: string | null,
  placeId: string | null,
  accessibleOnly = false,
) {
  return useQuery(placeRouteQueryOptions(fromNodeId, placeId, accessibleOnly))
}

/**
 * Amenities near where the patient stands (#109 / FR-26) — the ranked list
 * the journey screen's while-you-wait card is built from. Disabled until a
 * location exists: "nearby" has no meaning without an origin, so the card
 * rather than the API decides to stay hidden. The graph is static, so the
 * only thing a refetch can change is which amenities a new location
 * reaches — the 15s staleTime matches the location poll that drives it.
 */
export function nearbyAmenitiesQueryOptions(fromNodeId: string | null, accessibleOnly = false) {
  return queryOptions({
    queryKey: ['navigation', 'amenities', fromNodeId, accessibleOnly] as const,
    queryFn: () =>
      apiGet<AmenitySearch>(
        `/api/v1/navigation/amenities?from=${encodeURIComponent(fromNodeId ?? '')}${accessibleOnly ? '&accessibleOnly=true' : ''}`,
      ),
    enabled: fromNodeId !== null,
    staleTime: 15_000,
  })
}

export function useNearbyAmenities(fromNodeId: string | null, accessibleOnly = false) {
  return useQuery(nearbyAmenitiesQueryOptions(fromNodeId, accessibleOnly))
}
