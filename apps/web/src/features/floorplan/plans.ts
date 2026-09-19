// Floor-plan assets keyed by floor id, imported raw straight from
// packages/floorplans — the same cross-package reference `gen:api` uses for
// the contract — so the SVGs have exactly one source of truth and no copied
// asset to drift. Selector conventions (g[data-floor], [data-place-id],
// #route-layer) are documented in packages/floorplans/README.md.
import i1301Ground from '../../../../../packages/floorplans/floors/i-1301-ground.svg?raw'
import i1302Upper from '../../../../../packages/floorplans/floors/i-1302-upper.svg?raw'

const PLANS: Record<string, string> = {
  'I-1301': i1301Ground,
  'I-1302': i1302Upper,
}

/** The floor-plan SVG for a floor id (e.g. "I-1301"); null when no plan exists. */
export function floorPlanFor(floorId: string): string | null {
  return PLANS[floorId] ?? null
}

/**
 * Thai label for a floor the app can show on a plan ("ชั้น 1"). Journey
 * places carry their floor's code from the API, but route nodes only know
 * floorId — this bridges the two for cross-floor turn cues (#29). Same
 * static scope as PLANS: the floors with a plan asset.
 */
export function floorLabelFor(floorId: string): string {
  const code = FLOOR_CODES[floorId]
  return code ? `ชั้น ${code}` : floorId
}

const FLOOR_CODES: Record<string, string> = {
  'I-1301': '1',
  'I-1302': '2',
}

/**
 * Whether the plan really carries this place — rooms and nav nodes both
 * hold data-place-id, so one hit is enough. This guards the honest
 * "ยังไม่รองรับเส้นทาง" state when the map model knows a place the plan
 * asset hasn't drawn yet.
 */
export function floorPlanHasPlace(svg: string, placeId: string): boolean {
  return svg.includes(`data-place-id="${placeId}"`)
}
