import { describe, expect, it } from 'vitest'
import { anchorQrPayload, graphFloors, wayfindingAnchors } from './graphs'

describe('wayfindingAnchors', () => {
  const anchors = wayfindingAnchors()

  it('extracts lifts, stairs, and entrances from the real graph assets', () => {
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
})

describe('anchorQrPayload', () => {
  it('encodes the bare node reference the QR provider parses', () => {
    expect(
      anchorQrPayload({ nodeId: 'I-1301/node-lift', kind: 'ELEVATOR', floorId: 'I-1301' }),
    ).toBe('node/I-1301/node-lift')
  })
})

describe('graphFloors', () => {
  it('lists every floor with a graph asset', () => {
    expect(graphFloors()).toEqual(['I-1301', 'I-1302'])
  })
})
