import { describe, expect, it } from 'vitest'
import { anchorQrPayload, wayfindingAnchors } from './graphs'
import type { NavNode } from './queries'

// What GET /api/v1/navigation/nodes returns (ADR-0015 §7), shaped like the
// seeded graph: ids are globally unique because the authored ones are
// floor-local — node-lift exists on both floors.
const nodes: NavNode[] = [
  { id: 'I-1301/node-main-entrance', floorId: 'I-1301', x: 150, y: -10, nodeType: 'ENTRANCE' },
  { id: 'I-1301/node-lift', floorId: 'I-1301', x: 450, y: 205, nodeType: 'ELEVATOR' },
  { id: 'I-1301/node-stairs', floorId: 'I-1301', x: 962, y: 205, nodeType: 'STAIRS' },
  { id: 'I-1301/node-reception', floorId: 'I-1301', x: 150, y: 190, nodeType: 'PLACE_ENTRY' },
  { id: 'I-1301/node-ramp', floorId: 'I-1301', x: 318, y: 190, nodeType: 'CORRIDOR' },
  { id: 'I-1302/node-lift', floorId: 'I-1302', x: 450, y: 305, nodeType: 'ELEVATOR' },
]

describe('wayfindingAnchors', () => {
  const anchors = wayfindingAnchors(nodes)

  it('extracts lifts, stairs, and entrances', () => {
    const kinds = anchors.map((a) => a.kind)
    expect(kinds).toContain('ELEVATOR')
    expect(kinds).toContain('STAIRS')
    expect(kinds).toContain('ENTRANCE')
  })

  it('names nodes globally — the lift exists on both floors', () => {
    const lifts = anchors.filter((a) => a.kind === 'ELEVATOR').map((a) => a.nodeId)
    expect(lifts).toContain('I-1301/node-lift')
    expect(lifts).toContain('I-1302/node-lift')
  })

  it('never carries a place entry node — places are service points, not anchors', () => {
    expect(anchors.some((a) => a.nodeId.includes('node-reception'))).toBe(false)
  })

  it('leaves corridors out — nobody puts a sticker on a hallway', () => {
    expect(anchors.some((a) => a.nodeId.includes('node-ramp'))).toBe(false)
  })

  // These cards get printed and stuck to walls, which is why the graph they
  // come from is now the server's rather than a build-time copy: a node the
  // server has dropped must stop producing a sticker.
  it('reflects the graph it is given, not a bundled copy', () => {
    expect(wayfindingAnchors([])).toEqual([])
  })
})

describe('anchorQrPayload', () => {
  it('encodes the bare node reference the QR provider parses', () => {
    expect(
      anchorQrPayload({ nodeId: 'I-1301/node-lift', kind: 'ELEVATOR', floorId: 'I-1301' }),
    ).toBe('node/I-1301/node-lift')
  })
})
