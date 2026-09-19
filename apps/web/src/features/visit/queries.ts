import { queryOptions, useQuery } from '@tanstack/react-query'
import { ApiError, apiGet } from '@/api/client'
import type { components } from '@/api/schema'

// Every queryFn here throws ApiError, so hooks can read `error.status`.
declare module '@tanstack/react-query' {
  interface Register {
    defaultError: ApiError
  }
}

export type VisitView = components['schemas']['VisitView']
export type NextStep = components['schemas']['NextStep']

/**
 * The visit until the LINE LIFF hand-off exists: the demo visit seeded in
 * Mock HIS. Override per session with ?visit=<id> on the patient routes.
 */
export const DEFAULT_VISIT_ID = 'VISIT-001'

export function visitQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['visit', visitId] as const,
    queryFn: () =>
      apiGet<VisitView>(`/api/v1/visits/${encodeURIComponent(visitId)}`),
    staleTime: 30_000,
  })
}

export function useVisit(visitId: string) {
  return useQuery(visitQueryOptions(visitId))
}
