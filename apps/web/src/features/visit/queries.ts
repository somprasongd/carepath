import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiGet, apiPost } from '@/api/client'
import type { components } from '@/api/schema'

// Every queryFn here throws ApiError, so hooks can read `error.status`.
declare module '@tanstack/react-query' {
  interface Register {
    defaultError: ApiError
  }
}

export type Journey = components['schemas']['Journey']
export type JourneyStep = components['schemas']['JourneyStep']

/**
 * The visit until the LINE LIFF hand-off exists: the demo visit seeded in
 * Mock HIS. Override per session with ?visit=<id> on the patient routes.
 */
export const DEFAULT_VISIT_ID = 'VISIT-001'

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

/**
 * Whether the journey can still change. A completed or cancelled visit is
 * final — mirrors `visitOutcome` in journey.ts, kept local because that
 * module already imports types from this one.
 */
function journeyIsFinal(journey: Journey): boolean {
  return journey.status === 'CANCELLED' || journey.completed
}

/**
 * The journey plan CarePath derived for one visit (ADR-0009) — the patient
 * screens' single data source and the staff detail view.
 *
 * #36 realtime via polling (the issue's MVP bar: the simplest, most stable
 * option for the hackathon). The journey refetches every 15s — matching the
 * location poll — so step changes made by HIS or staff appear on the patient
 * screens without a reload, and stop once the visit is final. The key is
 * shared by the journey and navigate screens, and TanStack's structural
 * sharing keeps an unchanged payload from re-rendering the rail, so updates
 * land in one place instead of duplicating UI state (AC2).
 */
export function journeyQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['journey', visitId] as const,
    queryFn: () =>
      apiGet<Journey>(`/api/v1/journeys/${encodeURIComponent(visitId)}`),
    staleTime: 15_000,
    refetchInterval: (query) =>
      query.state.data && journeyIsFinal(query.state.data) ? false : 15_000,
  })
}

export function useJourney(visitId: string) {
  return useQuery(journeyQueryOptions(visitId))
}

/** The statuses the staff controls can command (#38). */
export type StepTargetStatus = Extract<
  components['schemas']['TransitionRequest']['to'],
  'STARTED' | 'COMPLETED'
>

/**
 * Staff step controls (#38): transition one step via the application API,
 * addressed by its stable stepKey (ADR-0009 — sequence can change on a
 * replan), tagging the audit trail with source "staff-web". CarePath owns
 * step status directly; the 200 body is the refreshed journey, so the detail
 * cache is written directly and the list just needs invalidating.
 */
export function useTransitionStep(visitId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { stepKey: string; to: StepTargetStatus }) =>
      apiPost<Journey>(
        `/api/v1/journeys/${encodeURIComponent(visitId)}/steps/${encodeURIComponent(input.stepKey)}/transition`,
        { to: input.to, source: 'staff-web' },
      ),
    onSuccess: (journey) => {
      queryClient.setQueryData(journeyQueryOptions(visitId).queryKey, journey)
      void queryClient.invalidateQueries({ queryKey: ['staff', 'visits'] })
    },
  })
}

/**
 * Staff override (ADR-0009 §4): confirm a clinic round is finished without
 * waiting for the HIS's encounter.completed fact.
 */
export function useCloseRound(visitId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (clinicCode: string) =>
      apiPost<Journey>(
        `/api/v1/journeys/${encodeURIComponent(visitId)}/clinics/${encodeURIComponent(clinicCode)}/close-round`,
        {},
      ),
    onSuccess: (journey) => {
      queryClient.setQueryData(journeyQueryOptions(visitId).queryKey, journey)
      void queryClient.invalidateQueries({ queryKey: ['staff', 'visits'] })
    },
  })
}
