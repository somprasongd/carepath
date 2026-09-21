import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiGetText, apiPostAction, apiPostSvg } from '@/api/client'
import type { components } from '@/api/schema'

export type FloorPlanRef = components['schemas']['FloorPlanRef']
export type NavNode = components['schemas']['NavNode']
export type StoredFloorPlan = components['schemas']['StoredFloorPlan']
export type FloorPlanWarning = components['schemas']['FloorPlanWarning']

/**
 * The floors and where to fetch their drawings (ADR-0015).
 *
 * This list used to be a constant next to the bundled SVGs, which is why it
 * could be one: both were fixed at build time. Since plans are uploaded, it
 * has to be served, and it is the pointer the app revalidates — the plan URL
 * it hands back is immutable, so everything expensive is cached behind it.
 */
export function floorsQueryOptions() {
  return queryOptions({
    queryKey: ['floors'] as const,
    queryFn: () => apiGet<FloorPlanRef[]>('/api/v1/floors'),
    // Short, because this is what stands between a replaced plan and the
    // patient who needs it. The SVG behind it costs nothing to keep.
    staleTime: 60_000,
  })
}

export function useFloors() {
  return useQuery(floorsQueryOptions())
}

/**
 * One floor's plan, by the immutable URL the floors listing gave us.
 *
 * The URL contains the plan's own digest, so the body can never change under
 * it: cached forever here and by the browser, and a redrawn floor arrives as
 * a new URL rather than as a stale hit.
 */
export function floorPlanQueryOptions(planUrl: string | undefined) {
  return queryOptions({
    queryKey: ['floor-plan', planUrl] as const,
    queryFn: () => apiGetText(planUrl!),
    enabled: Boolean(planUrl),
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

export function useFloorPlan(planUrl: string | undefined) {
  return useQuery(floorPlanQueryOptions(planUrl))
}

/**
 * The navigation graph's nodes (#105). The web app used to import these from
 * packages/floorplans at build time; it cannot any more, because the QR
 * stickers staff print are derived from this list and a stale copy would
 * send patients to nodes that no longer exist.
 */
export function navNodesQueryOptions() {
  return queryOptions({
    queryKey: ['navigation', 'nodes'] as const,
    queryFn: () => apiGet<NavNode[]>('/api/v1/navigation/nodes'),
    staleTime: 5 * 60_000,
  })
}

export function useNavNodes() {
  return useQuery(navNodesQueryOptions())
}

/**
 * A floor's stored plans, newest first (ADR-0015) — the history, and the
 * list a rollback picks from. ADMIN-only on the server.
 */
export function floorPlansQueryOptions(floorId: string | undefined) {
  return queryOptions({
    queryKey: ['admin', 'floor-plans', floorId] as const,
    queryFn: () => apiGet<StoredFloorPlan[]>(`/api/v1/admin/floors/${floorId}/plans`),
    enabled: Boolean(floorId),
  })
}

export function useFloorPlans(floorId: string | undefined) {
  return useQuery(floorPlansQueryOptions(floorId))
}

/**
 * The active plan's warnings, re-checked against the map model as it stands
 * right now (ADR-0015) — unlike a plan's own `warnings`, which is a snapshot
 * frozen at upload time and does not know about a place or node added or
 * removed since. This is what the health card renders; the stored warnings
 * are the plan's history, not its current standing.
 */
export function floorHealthQueryOptions(floorId: string | undefined) {
  return queryOptions({
    queryKey: ['admin', 'floor-health', floorId] as const,
    queryFn: () => apiGet<FloorPlanWarning[]>(`/api/v1/admin/floors/${floorId}/health`),
    enabled: Boolean(floorId),
  })
}

export function useFloorHealth(floorId: string | undefined) {
  return useQuery(floorHealthQueryOptions(floorId))
}

/**
 * Replace a floor's plan.
 *
 * A rejection here is the normal case, not an exception: the server refuses
 * a plan it cannot vouch for and says which rule broke, so the caller shows
 * that message rather than a generic failure.
 */
export function useUploadFloorPlan(floorId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (svg: string) =>
      apiPostSvg<StoredFloorPlan>(`/api/v1/admin/floors/${floorId}/plan`, svg),
    onSuccess: () => invalidatePlanState(queryClient, floorId),
  })
}

/** Point the floor back at a plan it already has. */
export function useActivateFloorPlan(floorId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (planId: string) =>
      apiPostAction<StoredFloorPlan>(`/api/v1/admin/floors/${floorId}/plans/${planId}/activate`),
    onSuccess: () => invalidatePlanState(queryClient, floorId),
  })
}

/**
 * After either write, the floors listing is what every screen reads the
 * current plan URL from — refetching it is what makes the new drawing
 * appear. The drawings themselves are never invalidated: their URLs carry
 * their own digests, so a changed plan is a different key, and the old one
 * staying cached is correct rather than stale.
 */
function invalidatePlanState(
  queryClient: ReturnType<typeof useQueryClient>,
  floorId: string,
) {
  void queryClient.invalidateQueries({ queryKey: ['floors'] })
  void queryClient.invalidateQueries({ queryKey: ['admin', 'floor-plans', floorId] })
  void queryClient.invalidateQueries({ queryKey: ['admin', 'floor-health', floorId] })
}

/**
 * Warm a floor's drawing ahead of the screen that shows it.
 *
 * Bundled plans cost nothing to open; fetched ones do, and the navigate
 * screen is reached by a deliberate tap from the journey screen — so the
 * plan can be on its way while the patient is still reading. The URL is
 * immutable, so this is at most one request per drawing, ever.
 */
export function usePrefetchFloorPlan(floorId: string | undefined) {
  const { data: floors } = useFloors()
  const planUrl = floors?.find((floor) => floor.floorId === floorId)?.planUrl
  // Not a mutation and not rendered — useQuery with the same key as the
  // navigate screen's is the prefetch, and the second read is a cache hit.
  useQuery(floorPlanQueryOptions(planUrl))
}
