// Navigation-graph assets keyed by floor id, imported raw from
// packages/floorplans/graphs — the same cross-package reference plans.ts
// uses for the SVGs, and the same JSON the API seeds its graph from
// (migration 000008): one source of truth, no second copy to drift.
import i1301Graph from '../../../../../packages/floorplans/graphs/i-1301.json?raw'
import i1302Graph from '../../../../../packages/floorplans/graphs/i-1302.json?raw'

/** The graph JSON's node shape (packages/floorplans/README.md): floor-local
 *  ids and coordinates in the floor's own SVG space. */
type GraphNode = {
  id: string
  x: number
  y: number
  type: string
  zone?: string
  placeId?: string
}

type FloorGraph = {
  floorId: string
  nodes: GraphNode[]
}

const GRAPHS: Record<string, FloorGraph> = {
  'I-1301': JSON.parse(i1301Graph) as FloorGraph,
  'I-1302': JSON.parse(i1302Graph) as FloorGraph,
}

/** Every floor that has a graph asset, in display order. */
export function graphFloors(): string[] {
  return Object.keys(GRAPHS)
}

/**
 * A map point a patient can stand at but that is not a service place —
 * lifts, stairs, entrances. These are the QR-sticker candidates beyond the
 * service points: the payload is the bare node reference the QR provider
 * resolves ("node/<floorId>/<localId>", qr.go).
 */
export type WayfindingAnchor = {
  /** Globally-unique node id, e.g. "I-1301/node-lift". */
  nodeId: string
  /** Graph node type — the label comes from the i18n catalog (anchor.kind.*). */
  kind: string
  floorId: string
}

/** The anchor node types, in display order. */
export const ANCHOR_KINDS = ['ELEVATOR', 'STAIRS', 'ENTRANCE'] as const

/** All wayfinding anchors across floors: lifts, stairs, and entrances. */
export function wayfindingAnchors(): WayfindingAnchor[] {
  const anchors: WayfindingAnchor[] = []
  for (const floorId of graphFloors()) {
    for (const node of GRAPHS[floorId]?.nodes ?? []) {
      if (!ANCHOR_KINDS.includes(node.type as (typeof ANCHOR_KINDS)[number])) continue
      anchors.push({ nodeId: `${floorId}/${node.id}`, kind: node.type, floorId })
    }
  }
  return anchors
}

/** The QR payload for an anchor's sticker (see the QR provider's node refs). */
export function anchorQrPayload(anchor: WayfindingAnchor): string {
  return `node/${anchor.nodeId}`
}
