import type { JourneyStep as RailStep, JourneyStepState } from '@/design-system'
import type { ApiError } from '@/api/client'
import { format, lookup, messagesFor, type Locale } from '@/i18n'
import type { Journey, JourneyStep } from './queries'

/**
 * Patient-facing title for a journey step (ADR-0012: display text is owned
 * by the client, keyed by the stable codes of ADR-0009). A round above 1
 * means the patient is returning to a doctor they already saw this visit.
 * `kind` is typed as a plain string, not the schema enum, so an unmapped
 * kind still renders instead of being a type error — the fallback is the
 * point.
 */
export function stepTitle(
  step: Pick<JourneyStep, 'clinicCode' | 'round'> & { kind: string },
  locale: Locale,
): string {
  const catalog = messagesFor(locale)
  if (step.kind === 'CLINIC') {
    const clinic = step.clinicCode
      ? lookup(catalog, `step.clinic.${step.clinicCode}`) ?? step.clinicCode
      : ''
    const suffix = clinic ? ` · ${clinic}` : ''
    const base = step.round && step.round > 1 ? catalog['step.seeDoctorAgain'] : catalog['step.seeDoctor']
    return `${base}${suffix}`
  }
  return lookup(catalog, `step.title.${step.kind}`) ?? step.kind
}

/**
 * Patient-facing service point name (ADR-0012 §1): the stable `code` maps
 * into the locale catalog; an unmapped code falls back to the server's
 * `name` — whatever language the admin set — instead of breaking the screen.
 */
export function servicePointLabel(
  servicePoint: Pick<NonNullable<JourneyStep['servicePoint']>, 'code' | 'name'>,
  locale: Locale,
): string {
  return lookup(messagesFor(locale), `sp.${servicePoint.code}`) ?? servicePoint.name
}

/**
 * Map a Journey onto the patient's journey rail (ADR-0009). More than one
 * step can be actionable at once; the recommended one (or any STARTED step)
 * gets the rail's `current` emphasis, every other actionable step is `next`,
 * everything else stays `pending`.
 */
export function toJourneySteps(journey: Journey, locale: Locale): RailStep[] {
  const actionableKeys = new Set(journey.actionable.map((s) => s.stepKey))
  const recommendedKey = journey.recommended?.stepKey
  const startedKey = journey.steps.find((s) => s.status === 'STARTED')?.stepKey

  return journey.steps.map((step) => {
    let state: JourneyStepState
    if (step.status === 'COMPLETED' || step.status === 'CANCELLED') {
      state = 'done'
    } else if (step.stepKey === startedKey || (!startedKey && step.stepKey === recommendedKey)) {
      state = 'current'
    } else if (actionableKeys.has(step.stepKey)) {
      state = 'next'
    } else {
      state = 'pending'
    }

    return {
      id: step.stepKey,
      state,
      title: stepTitle(step, locale),
      meta: stepMeta(step, state, locale),
    }
  })
}

function stepMeta(step: JourneyStep, state: JourneyStepState, locale: Locale): string {
  const catalog = messagesFor(locale)
  if (step.status === 'CANCELLED') return catalog['step.meta.cancelled']
  if (step.status === 'COMPLETED') return catalog['step.meta.completed']
  if (step.status === 'WAITING') return catalog['step.meta.waitingResult']
  if (state === 'current' || state === 'next') {
    return step.servicePoint
      ? `${servicePointLabel(step.servicePoint, locale)} · ${step.servicePoint.placeId}`
      : catalog['step.meta.ready']
  }
  return catalog['step.meta.pending']
}

/**
 * Journey progress for the patient home screen (#34): how many steps are
 * finished. Cancelled steps stay in `total` — they are part of the plan the
 * rail recaps (the `step.meta.cancelled` label), so 2 done of 5 with one
 * cancelled still reads honestly against the rail.
 */
export function journeyProgress(journey: Journey): { done: number; total: number } {
  const done = journey.steps.filter((s) => s.status === 'COMPLETED').length
  return { done, total: journey.steps.length }
}

/** The progress line under the screen lead, in the patient's language. */
export function journeyProgressLabel(journey: Journey, locale: Locale): string {
  const { done, total } = journeyProgress(journey)
  return format(messagesFor(locale), 'journey.progress', { done, total })
}

/**
 * How the visit ended for the patient home screen (#34). `completed` is the
 * contract's unambiguous finished signal (actionable empty, recommended
 * null); CANCELLED only exists on `status`. A visit can be neither — still
 * walking between steps.
 */
export type VisitOutcome = 'completed' | 'cancelled' | null

export function visitOutcome(journey: Journey): VisitOutcome {
  if (journey.status === 'CANCELLED') return 'cancelled'
  if (journey.completed) return 'completed'
  return null
}

/**
 * Load-failure phrasing for the patient, keyed off the HTTP status — the raw
 * error detail (English, internal) stays in the console, never on screen.
 */
export function visitLoadErrorMessage(error: ApiError | null, locale: Locale): string {
  const catalog = messagesFor(locale)
  if (error?.status === 404) return catalog['journey.error.notFound']
  if (error?.status === 502) return catalog['journey.error.hisUnavailable']
  return catalog['journey.error.unreachable']
}
