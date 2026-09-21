// Wayfinding anchors derived from the navigation graph the API serves
// (ADR-0015 §7).
//
// This used to read packages/floorplans/graphs/*.json at build time. It had
// to stop: the anchors below become the QR stickers staff print and put on
// walls, so a build-time copy that drifted from the database would send
// patients to nodes that no longer exist — a drift that leaves the screen.
import type { NavNode } from './queries'

/**
 * A map point a patient can stand at that is not a service place — lifts,
 * stairs, entrances. These are the QR-sticker candidates beyond the service
 * points: the payload is the bare node reference the QR provider resolves
 * ("node/<floorId>/<localId>", qr.go).
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

/** Lifts, stairs and entrances, across every floor of the given graph. */
export function wayfindingAnchors(nodes: NavNode[]): WayfindingAnchor[] {
  return nodes
    .filter((node) => ANCHOR_KINDS.includes(node.nodeType as (typeof ANCHOR_KINDS)[number]))
    .map((node) => ({ nodeId: node.id, kind: node.nodeType, floorId: node.floorId }))
}

/** The QR payload for an anchor's sticker (see the QR provider's node refs). */
export function anchorQrPayload(anchor: WayfindingAnchor): string {
  return `node/${anchor.nodeId}`
}
