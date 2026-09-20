// Pure mapping from the navigation route API (#28) onto what the patient
// sees: geometry for the SVG overlay and turn-by-turn cues in the patient's
// language (ADR-0012). Distances in the API are authored SVG units, not
// metres — nothing here converts or displays them, so no made-up walking
// times (DESIGN.md).
import type { components } from '@/api/schema'
import { format, lookup, messagesFor, type Locale } from '@/i18n'
import type { LocationObservation } from './queries'

export type NavigationRoute = components['schemas']['NavigationRoute']
export type NavNode = components['schemas']['NavNode']
export type NavEdge = components['schemas']['NavEdge']

export type Point = { x: number; y: number }

/**
 * Where the patient stands on `floorId`'s plan — the route's first node, but
 * only when it actually sits on that floor: a cross-floor route starts on the
 * patient's floor, and a "you are here" mark at the lift on the destination
 * floor would lie (DESIGN.md's honesty rule, #35).
 */
export function routeOriginOnFloor(nodes: NavNode[], floorId: string): Point | null {
  const first = nodes[0]
  return first && first.floorId === floorId ? { x: first.x, y: first.y } : null
}

/**
 * One plain line stating where the patient currently is (#35): floor,
 * positioning zone when the source carries one, and how it was known. Floor
 * ids and unknown provider names never reach the patient; an unmapped source
 * simply omits its part instead of showing raw vocabulary.
 */
export function currentLocationLabel(
  observation: Pick<LocationObservation, 'floorId' | 'zone' | 'source'>,
  floorLabel: (floorId: string) => string,
  locale: Locale,
): string {
  const catalog = messagesFor(locale)
  const parts = [
    format(catalog, 'navigate.locationNowAt', { floor: floorLabel(observation.floorId) }),
  ]
  if (observation.zone) parts.push(format(catalog, 'navigate.zone', { zone: observation.zone }))
  const source = lookup(catalog, `navigate.source.${observation.source}`)
  if (source) parts.push(source)
  return parts.join(' · ')
}

/**
 * One floor's walking lines in walk order — almost always a single line; a
 * path that revisits a floor keeps its two visits as two lines rather than
 * bridging them with a fake segment. A floor change starts a new line: the
 * lift/stairs node exists once per floor (linked by the graph's
 * transitions), so the boundary is simply the floorId prefix changing.
 */
export function routePolylinesByFloor(nodes: NavNode[]): Record<string, Point[][]> {
  const lines: Record<string, Point[][]> = {}
  let currentFloor: string | null = null
  let current: Point[] = []
  for (const node of nodes) {
    if (node.floorId !== currentFloor) {
      current = [{ x: node.x, y: node.y }]
      ;(lines[node.floorId] ??= []).push(current)
      currentFloor = node.floorId
      continue
    }
    current.push({ x: node.x, y: node.y })
  }
  return lines
}

/**
 * Turn-by-turn cues in the patient's language. Runs of corridor walking
 * collapse into one cue; every floor change through a lift or stairs becomes
 * its own cue naming the floor it lands on. Domain vocabulary (CORRIDOR,
 * ELEVATOR, node ids) never reaches the patient (DESIGN.md).
 */
export function turnByTurnSteps(
  nodes: NavNode[],
  segments: NavEdge[],
  destinationName: string,
  floorLabel: (floorId: string) => string,
  locale: Locale,
): string[] {
  if (nodes.length === 0) return []
  const catalog = messagesFor(locale)

  const cues: string[] = []
  let walked = false
  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i]
    if (segment.edgeType === 'CORRIDOR') {
      walked = true
      continue
    }

    // A floor change: cue the corridor walk that reached the lift/stairs,
    // then the vertical move itself, named by the floor it lands on.
    if (walked) cues.push(catalog['navigate.cue.followLine'])
    walked = false
    const target = nodes[i + 1]
    const key = segment.edgeType === 'ELEVATOR' ? 'navigate.cue.elevator' : 'navigate.cue.stairs'
    cues.push(format(catalog, key, { floor: floorLabel(target.floorId) }))
  }
  // Same-floor routes (and origin == destination) still get one walking cue
  // plus the arrival, so the panel is never empty when a route exists.
  if (walked || cues.length === 0) cues.push(catalog['navigate.cue.followLine'])
  cues.push(format(catalog, 'navigate.cue.arrive', { name: destinationName }))
  return cues
}

/**
 * The bbox of the displayed floor's walking lines as two corner
 * pseudo-points — the shape `routedViewBox` fits. Callers fold any floor
 * group translate in first (the pin's DOM-measure trick).
 */
export function routeBounds(lines: Point[][]): { from: Point; to: Point } | null {
  const points = lines.flat()
  if (points.length === 0) return null
  const xs = points.map((p) => p.x)
  const ys = points.map((p) => p.y)
  return {
    from: { x: Math.min(...xs), y: Math.min(...ys) },
    to: { x: Math.max(...xs), y: Math.max(...ys) },
  }
}
