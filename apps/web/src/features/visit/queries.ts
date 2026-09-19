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
export type Journey = components['schemas']['Journey']
export type JourneyStep = components['schemas']['JourneyStep']

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

/**
 * Staff visit monitor (#37): every projected journey, freshest sync first —
 * the CarePath projection, so a visit not yet ingested is simply absent.
 */
export function staffVisitsQueryOptions() {
  return queryOptions({
    queryKey: ['staff', 'visits'] as const,
    queryFn: () => apiGet<Journey[]>('/api/v1/staff/visits'),
    staleTime: 15_000,
  })
}

export function useStaffVisits() {
  return useQuery(staffVisitsQueryOptions())
}

/** The projected journey of one visit — the staff detail view. */
export function journeyQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['journey', visitId] as const,
    queryFn: () =>
      apiGet<Journey>(`/api/v1/journeys/${encodeURIComponent(visitId)}`),
    staleTime: 15_000,
  })
}

export function useJourney(visitId: string) {
  return useQuery(journeyQueryOptions(visitId))
}
