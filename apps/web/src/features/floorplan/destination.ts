import { servicePointLabel, stepTitle, type Journey } from '@/features/visit'
import { format, messagesFor, type Locale } from '@/i18n'

export type DestinationPlan = {
  /** AppBar title, e.g. "Directions to Medication pickup" in English. */
  title: string
  /** The destination's name — also drawn beside the map pin (#25 AC3). */
  name: string
  /** Place line under the title, e.g. "Floor 1 · Pharmacy · PHARMACY-01". */
  subtitle: string
  floorId: string
  /** Patient-facing floor label for the map caption, e.g. "Floor 1". */
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
  locale: Locale,
): NavigatePlan {
  if (isPending) return { state: 'pending' }

  const recommended = journey?.recommended
  if (!journey || !recommended) return { state: 'no-destination' }

  const catalog = messagesFor(locale)
  const title = format(catalog, 'navigate.routeTitle', {
    name: stepTitle(recommended, locale),
  })
  const servicePoint = recommended.servicePoint
  const place = servicePoint?.place ?? null

  // "Routable" is a fact about the map model, not about whichever drawing is
  // currently loaded: a place is routable when it resolves to an entry node
  // in the navigation graph (the packages/floorplans README's "rooms without
  // a node-* are selectable but not yet routable").
  //
  // This used to ask the bundled SVG whether it drew the place, which is no
  // longer a question with a synchronous answer — plans are fetched
  // (ADR-0015). Asking the journey data instead also decides earlier and
  // more honestly: the screen settles on supported-or-not before the drawing
  // arrives, so it never shows a route it then has to take away, and the
  // turn-by-turn text (which comes from the route API, not the plan) renders
  // without waiting for the map.
  if (!place || !place.entryNodeId) {
    return {
      state: 'unsupported',
      title,
      name: servicePoint ? servicePointLabel(servicePoint, locale) : '',
    }
  }

  // The service point's name follows the locale like every patient string
  // (ADR-0012 §1: catalog by code, server name only as the unmapped-code
  // fallback — #94's "service point names switch too" line).
  const name = servicePoint ? servicePointLabel(servicePoint, locale) : place.name
  const floorLabel = format(catalog, 'common.floor', { code: place.floor.code })
  return {
    state: 'plan',
    title,
    name,
    subtitle: format(catalog, 'navigate.subtitle', { floor: floorLabel, name, place: place.id }),
    floorId: place.floorId,
    floorLabel,
    placeId: place.id,
    servicePointCode: servicePoint?.code ?? '',
    x: place.x,
    y: place.y,
  }
}
