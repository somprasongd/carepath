// Floor-plan helpers over the data the API serves (ADR-0015).
//
// These used to read `?raw` imports of packages/floorplans, which is exactly
// what FR-11 could not live with: an admin who adds a place got a plan in the
// database and a bundle in the browser that had never heard of it. The SVG
// and the floor list now arrive through features/floorplan/queries.
import { format, messagesFor, type Locale } from '@/i18n'
import type { FloorPlanRef } from './queries'

/**
 * Patient-facing label for a floor ("Floor 1"), from the floor's own code.
 *
 * The code comes from the API and the wording from the catalog, so this needs
 * no per-floor entry in either — adding a floor does not touch i18n
 * (ADR-0012 §4's trigger fires for names, not for this).
 */
export function floorLabelFor(floorId: string, locale: Locale, floors: FloorPlanRef[]): string {
  const code = floors.find((floor) => floor.floorId === floorId)?.code
  return code ? format(messagesFor(locale), 'common.floor', { code }) : floorId
}

/** The plan URL for a floor, or undefined when that floor has no plan yet. */
export function planUrlFor(floorId: string, floors: FloorPlanRef[]): string | undefined {
  return floors.find((floor) => floor.floorId === floorId)?.planUrl
}

/**
 * Whether the plan really draws this place — rooms and nav nodes both carry
 * data-place-id, so one hit is enough. This guards the honest
 * route-unsupported state when the map model knows a place the current plan
 * has not drawn. Since ADR-0015 that gap is also reported at upload time, as
 * a PLACE_MISSING warning, so an admin sees it before a patient does.
 */
export function floorPlanHasPlace(svg: string, placeId: string): boolean {
  return svg.includes(`data-place-id="${placeId}"`)
}
