import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authMode } from '@/auth/auth-mode'
import { ApiError, apiGet, apiPost, apiPostNoContent, setApiAuthToken } from '@/api/client'
import type { components } from '@/api/schema'

// Every queryFn here throws ApiError, so hooks can read `error.status`.
declare module '@tanstack/react-query' {
  interface Register {
    defaultError: ApiError
  }
}

export type Journey = components['schemas']['Journey']
export type JourneyStep = components['schemas']['JourneyStep']
export type PlanningRules = components['schemas']['PlanningRules']
export type VisitQueue = components['schemas']['JourneyQueue']
export type VisitQueueStep = components['schemas']['JourneyQueueStep']
export type StationQueue = components['schemas']['StationQueue']
export type StationQueueEntry = components['schemas']['StationQueueEntry']

/**
 * One station's live queue (FR-15, #102): who is being served (STARTED,
 * latest call first) and who is waiting (READY, earliest arrival first).
 * Polled on the staff console's cadence so calls made elsewhere land here
 * without a reload; calling the next patient is the same staff transition
 * the patients' screen sees — its success invalidates this key.
 */
export function stationQueueQueryOptions(servicePointId: string) {
  return queryOptions({
    queryKey: ['staff', 'queue', servicePointId] as const,
    queryFn: () =>
      apiGet<StationQueue>(`/api/v1/staff/queue/${encodeURIComponent(servicePointId)}`),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useStationQueue(servicePointId: string, options?: { enabled?: boolean }) {
  return useQuery({
    ...stationQueueQueryOptions(servicePointId),
    enabled: options?.enabled ?? true,
  })
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

/**
 * The journey-planning rules in force (FR-13, #100): the phase ladder, the
 * order-type mapping, and worked examples — everything derived from the
 * API's planner itself, so this screen can never drift from the rules
 * actually running. The rules only change with a deploy, so a long staleTime
 * just avoids re-fetching a static payload on every visit.
 */
export function planningRulesQueryOptions() {
  return queryOptions({
    queryKey: ['staff', 'planning-rules'] as const,
    queryFn: () => apiGet<PlanningRules>('/api/v1/staff/planning-rules'),
    staleTime: 5 * 60_000,
  })
}

export function usePlanningRules() {
  return useQuery(planningRulesQueryOptions())
}

/**
 * Whether the journey can still change. A completed or cancelled visit is
 * final — mirrors `visitOutcome` in journey.ts, kept local because that
 * module already imports types from this one.
 */
function journeyIsFinal(journey: Journey): boolean {
  return journey.status === 'CANCELLED' || journey.completed
}

// Visits this page session has successfully claimed, plus the claims
// currently in flight. The claim (#96) is what mints read access to a
// visit — every patient visit-scoped call goes through ensureVisitClaimed
// first, so a QR deep-link straight into /patient/navigate or a share
// command works without passing the journey screen first.
const claimedVisits = new Set<string>()
const claimsInFlight = new Map<string, Promise<void>>()
// Visits whose claim-by-name route answered a 404 that is NOT the
// journey-not-found envelope: the route itself is retired (the mint is
// live, #136 layer 2), so retrying can never mint a claim. Remembered for
// the page session so the 15s polls stop firing a dead POST.
const unclaimableVisits = new Set<string>()

/**
 * Bind this session's identity to the visit (#96) before reading or
 * commanding it. Claim-by-VN is the front door only on demo deployments
 * WITHOUT the visit-link mint: there knowing the visit id (typed VN,
 * scanned QR) is the credential. Wherever the mint is live, the server
 * retires the route (a live mint makes knowing a VN worthless on purpose —
 * #136), the slip token claims instead (redeem, or the visit-token session
 * bootstrap below), and this becomes a no-op whose 404 is remembered. Line
 * mode never had it: an un-redeemed ?visit= simply reads as the journey's
 * not-found state.
 * Idempotent server-side, and once per page session is enough — the Set keeps
 * the 15s polls and parallel queries from re-firing the POST. Only a claim
 * that SUCCEEDED is remembered: a failed one is retried by the next call
 * (query retry, refetch), so one transient failure can't wedge the visit's
 * queries into permanent 404s until a reload. A visit that was never
 * projected 404s here with the same "journey not found" the read itself
 * would give.
 */
export async function ensureVisitClaimed(visitId: string): Promise<void> {
  if (authMode === 'line') return
  if (claimedVisits.has(visitId) || unclaimableVisits.has(visitId)) return
  // Concurrent callers (poll + parallel queries) share one in-flight POST.
  const existing = claimsInFlight.get(visitId)
  if (existing) {
    await existing
    return
  }
  const claim = apiPostNoContent(`/api/v1/journeys/${encodeURIComponent(visitId)}/claim`)
    .then(() => {
      claimedVisits.add(visitId)
    })
    .catch((err) => {
      // "journey not found" is the visit-not-projected answer — retryable,
      // the projection may still land. Any other 404 means the route itself
      // is gone (the deployment retired claim-by-name with a live mint), and
      // no retry can succeed: remember it and let the read surface its own
      // not-found instead of a dead POST on every poll.
      if (err instanceof ApiError && err.status === 404 && err.message !== 'journey not found') {
        unclaimableVisits.add(visitId)
        return
      }
      throw err
    })
    .finally(() => {
      claimsInFlight.delete(visitId)
    })
  claimsInFlight.set(visitId, claim)
  await claim
}

/**
 * Exchange the slip-held link token (#136) for the visit it addresses. The
 * token rides the request body, never a URL; the server records the #96
 * claim for this session's identity, so the returned visit id needs no
 * further claim. Unknown, rotated, cancelled, and past-grace tokens all
 * answer the same 404 — the caller decides what that screen says.
 */
export async function redeemVisitLinkToken(token: string): Promise<string> {
  const response = await apiPost<{ visitId: string }>('/api/v1/journeys/claim', { token })
  claimedVisits.add(response.visitId)
  return response.visitId
}

/**
 * Bootstrap a patient session straight from the slip token (#136 layer 2)
 * over POST /auth/session (source "visit-token") — the QR-only front door
 * for hospitals without a LINE OA: no login exists for the token to top
 * up, so the token itself mints the session, and the server claims the
 * visit for the visit-scoped identity before answering. The fresh bearer
 * replaces whatever was in flight (a shared demo identity claiming the
 * visit would re-open exactly the door this mode exists to close).
 * Unknown, rotated, cancelled, and past-grace tokens answer the same 404
 * redeem does.
 */
export async function createVisitTokenSession(token: string): Promise<string> {
  const session = await apiPost<components['schemas']['CreateSessionResponse']>(
    '/api/v1/auth/session',
    { source: 'visit-token', idToken: token },
  )
  if (!session.visitId) {
    // The server only omits visitId for a contract violation — treat it as
    // a dead token rather than navigating to a visitless journey screen.
    throw new ApiError(404, 'visit-token session returned no visitId')
  }
  setApiAuthToken(session.sessionToken)
  claimedVisits.add(session.visitId)
  return session.visitId
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

/**
 * The patient queue picture (FR-17, #101): for each actionable step, how
 * many people are waiting ahead at its service point and that point's
 * average wait today. Polled on the journey's cadence — the queue moves
 * faster than the plan — and keyed under the journey's key so a transition
 * invalidating ['journey', visitId] refreshes the queue too. avgWaitMinutes
 * null means "no samples yet", not "zero minutes": the card renders words
 * for that case, never a guessed number.
 */
export function visitQueueQueryOptions(visitId: string) {
  return queryOptions({
    queryKey: ['journey', visitId, 'queue'] as const,
    queryFn: async () => {
      await ensureVisitClaimed(visitId)
      return apiGet<VisitQueue>(`/api/v1/journeys/${encodeURIComponent(visitId)}/queue`)
    },
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useVisitQueue(visitId: string, options?: { enabled?: boolean }) {
  return useQuery({ ...visitQueueQueryOptions(visitId), enabled: options?.enabled ?? true })
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
      // The whole staff namespace, not just the visit list: a transition is
      // also what the station queue screens live on (#102) — the called
      // patient must move from waiting to serving immediately.
      void queryClient.invalidateQueries({ queryKey: ['staff'] })
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
      void queryClient.invalidateQueries({ queryKey: ['staff'] })
    },
  })
}
