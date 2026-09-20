import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, apiGet, apiPost, apiPostNoContent } from '@/api/client'
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

// Visits this page session has already claimed. The claim (#96) is what
// mints read access to a visit — every patient visit-scoped call goes
// through ensureVisitClaimed first, so a QR deep-link straight into
// /patient/navigate or a share command works without passing the journey
// screen first.
const claimedVisits = new Set<string>()

/**
 * Bind this session's identity to the visit (#96) before reading or
 * commanding it. Claiming is the patient's front door made explicit: knowing
 * the visit id (typed VN, scanned QR) is the credential. Idempotent server-
 * side, and once per page session is enough — the Set keeps the 15s polls
 * and parallel queries from re-firing the POST. A visit that was never
 * projected 404s here with the same "journey not found" the read itself
 * would give.
 */
export async function ensureVisitClaimed(visitId: string): Promise<void> {
  if (claimedVisits.has(visitId)) return
  claimedVisits.add(visitId)
  await apiPostNoContent(`/api/v1/journeys/${encodeURIComponent(visitId)}/claim`)
}

/**
 * The journey plan CarePath derived for one visit (ADR-0009) — the patient
 * screens' single data source. The staff detail view used to read this too;
 * since #96 the read is claim-guarded, so staff derives its detail from the
 * /staff/visits list instead.
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
    queryFn: async () => {
      await ensureVisitClaimed(visitId)
      return apiGet<Journey>(`/api/v1/journeys/${encodeURIComponent(visitId)}`)
    },
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
/**
 * The transition endpoint URL. The stepKey goes in raw: its colons
 * (CLINIC:MED:1) are legal path characters, but encodeURIComponent turns
 * them into %3A, which the API's route param does not decode — the
 * transition then 404s on every clinic/order step. Keys come from the
 * planner's restricted alphabet, so they are path-safe as-is.
 */
export function transitionStepUrl(visitId: string, stepKey: string): string {
  return `/api/v1/journeys/${encodeURIComponent(visitId)}/steps/${stepKey}/transition`
}

export function useTransitionStep(visitId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { stepKey: string; to: StepTargetStatus }) =>
      apiPost<Journey>(transitionStepUrl(visitId, input.stepKey), {
        to: input.to,
        source: 'staff-web',
      }),
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
