import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGetShared, apiPost } from '@/api/client'
import type { components } from '@/api/schema'

export type ShareLinkResponse = components['schemas']['ShareLinkResponse']
export type SharedJourney = components['schemas']['SharedJourney']
export type SharedStep = components['schemas']['SharedStep']

/**
 * Both share commands hit the same path — mint (POST) and revoke-all
 * (DELETE). The visit id is an opaque token, so it is encoded; unlike the
 * stepKey colons (features/visit/queries.ts), visit ids carry no raw
 * punctuation that the API route would not decode.
 */
export function shareUrl(visitId: string): string {
  return `/api/v1/journeys/${encodeURIComponent(visitId)}/share`
}

/**
 * Patient action (#89/#90): mint a tracking link for this visit. The token
 * comes back exactly once — the sheet that triggered it is the only place
 * it is ever held.
 */
export function useCreateShareLink(visitId: string) {
  return useMutation({
    mutationFn: () => apiPost<ShareLinkResponse>(shareUrl(visitId), {}),
  })
}

/**
 * Patient action (#89/#90): revoke every active link for the visit. A
 * relative's open tab flips to the expired state on its next 15s refetch —
 * invalidate the shared cache so a same-tab back-and-forth sees it too.
 */
export function useStopSharing(visitId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiDelete(shareUrl(visitId)),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['shared-journey'] })
    },
  })
}

/**
 * The relative's read (#90): poll the shared journey every 15s, matching the
 * patient screens (#36), and stop once the visit is DONE — there is nothing
 * left to follow.
 *
 * The key is a constant on purpose: the share token must never enter a query
 * key (it would land in devtools and cache snapshots — apps/web AGENTS.md
 * token rule). The token lives only in api/client.ts's share slot.
 */
export function sharedJourneyQueryOptions(hasToken: boolean) {
  return queryOptions({
    queryKey: ['shared-journey'] as const,
    queryFn: () => apiGetShared<SharedJourney>('/api/v1/shared/journey'),
    enabled: hasToken,
    staleTime: 15_000,
    refetchInterval: (query) => (query.state.data?.status === 'DONE' ? false : 15_000),
  })
}

export function useSharedJourney(hasToken: boolean) {
  return useQuery(sharedJourneyQueryOptions(hasToken))
}
