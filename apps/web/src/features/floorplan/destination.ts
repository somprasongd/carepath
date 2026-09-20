import { stepTitle, type Journey } from '@/features/visit'
import { floorPlanFor, floorPlanHasPlace } from './plans'

export type DestinationPlan = {
  /** AppBar title, e.g. "เส้นทางไปรับยา". */
  title: string
  /** The destination's name — also drawn beside the map pin (#25 AC3). */
  name: string
  /** Place line under the title, e.g. "ชั้น 1 · Pharmacy · PHARMACY-01". */
  subtitle: string
  floorId: string
  /** Thai floor label for the map caption, e.g. "ชั้น 1". */
  floorLabel: string
  /** Stable SVG place id — the join key to data-place-id in the plan asset. */
  placeId: string
  /** The destination's service point code — the `to` end of /navigation/route (#28). */
  servicePointCode: string
  /** Floor-local SVG units of the place's entry node; the pin falls back to the room's centre when absent. */
  x?: number
  y?: number
}

/**
 * What the navigate screen should show for a visit (#25). `plan` is the
 * destination-on-a-floor state; the others keep DESIGN.md's honesty rules —
 * never a wrong map, never a made-up destination.
 */
export type NavigatePlan =
  | { state: 'pending' }
  | { state: 'no-destination' }
  | { state: 'unsupported'; title: string; name: string }
  | ({ state: 'plan' } & DestinationPlan)

/**
 * The recommended step (ADR-0009: `journey.recommended`) is where the patient
 * is being sent; when nothing is recommended there is no destination yet.
 */
export function navigatePlanForJourney(
  journey: Journey | undefined,
  isPending: boolean,
): NavigatePlan {
  if (isPending) return { state: 'pending' }

  const recommended = journey?.recommended
  if (!journey || !recommended) return { state: 'no-destination' }

  const title = `เส้นทางไป${stepTitle(recommended, 'th')}`
  const servicePoint = recommended.servicePoint
  const place = servicePoint?.place ?? null
  const svg = place ? floorPlanFor(place.floorId) : null

  if (!place || !svg || !floorPlanHasPlace(svg, place.id)) {
    return {
      state: 'unsupported',
      title,
      name: servicePoint?.name ?? servicePoint?.code ?? '',
    }
  }

  const name = servicePoint?.name ?? place.name
  return {
    state: 'plan',
    title,
    name,
    subtitle: `ชั้น ${place.floor.code} · ${name} · ${place.id}`,
    floorId: place.floorId,
    floorLabel: `ชั้น ${place.floor.code}`,
    placeId: place.id,
    servicePointCode: servicePoint?.code ?? '',
    x: place.x,
    y: place.y,
  }
}
