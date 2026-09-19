import { describe, expect, it } from 'vitest'
import { routeBounds, routePolylinesByFloor, turnByTurnSteps, type NavEdge, type NavNode } from './route'

function node(floorId: string, x: number, y: number): NavNode {
  return { id: `${floorId}/n-${x}-${y}`, floorId, x, y, nodeType: 'CORRIDOR' }
}

function edge(from: number, to: number, edgeType: NavEdge['edgeType']): NavEdge {
  return {
    id: `e-${from}-${to}`,
    fromNodeId: `n-${from}`,
    toNodeId: `n-${to}`,
    edgeType,
    distance: 10,
    accessible: true,
  }
}

describe('routePolylinesByFloor', () => {
  it('groups a same-floor route into one walking line', () => {
    const lines = routePolylinesByFloor([
      node('I-1301', 150, 190),
      node('I-1301', 318, 190),
      node('I-1301', 445, 330),
      node('I-1301', 885, 190),
    ])
    expect(lines).toEqual({
      'I-1301': [
        [
          { x: 150, y: 190 },
          { x: 318, y: 190 },
          { x: 445, y: 330 },
          { x: 885, y: 190 },
        ],
      ],
    })
  })

  it('splits a cross-floor route into one line per floor in walk order', () => {
    const lines = routePolylinesByFloor([
      node('I-1301', 150, 190),
      node('I-1301', 620, 190),
      node('I-1301', 700, 300), // node-lift, ground side
      node('I-1302', 700, 300), // node-lift, upper side (same local coords)
      node('I-1302', 400, 300),
      node('I-1302', 165, 405),
    ])
    expect(Object.keys(lines)).toEqual(['I-1301', 'I-1302'])
    expect(lines['I-1301']).toEqual([
      [
        { x: 150, y: 190 },
        { x: 620, y: 190 },
        { x: 700, y: 300 },
      ],
    ])
    expect(lines['I-1302']).toEqual([
      [
        { x: 700, y: 300 },
        { x: 400, y: 300 },
        { x: 165, y: 405 },
      ],
    ])
  })

  it('keeps two visits to the same floor as two lines, not one bridged line', () => {
    const lines = routePolylinesByFloor([
      node('I-1301', 150, 190),
      node('I-1301', 700, 300),
      node('I-1302', 700, 300),
      node('I-1302', 165, 405),
      node('I-1302', 700, 300),
      node('I-1301', 700, 300),
      node('I-1301', 885, 190),
    ])
    expect(lines['I-1301']).toHaveLength(2)
    expect(lines['I-1302']).toHaveLength(1)
  })

  it('returns nothing for an empty route', () => {
    expect(routePolylinesByFloor([])).toEqual({})
  })
})

describe('turnByTurnSteps', () => {
  const label = (floorId: string) => (floorId === 'I-1301' ? 'ชั้น 1' : 'ชั้น 2')

  it('collapses a same-floor walk into one cue plus arrival', () => {
    const nodes = [node('I-1301', 150, 190), node('I-1301', 318, 190), node('I-1301', 885, 190)]
    const segments = [edge(0, 1, 'CORRIDOR'), edge(1, 2, 'CORRIDOR')]
    expect(turnByTurnSteps(nodes, segments, 'Pharmacy', label)).toEqual([
      'เดินตามเส้นสายส้มบนผัง',
      'ถึงPharmacy — จุดหมายของคุณ',
    ])
  })

  it('cues the lift ride with the floor it lands on', () => {
    const nodes = [
      node('I-1301', 150, 190),
      node('I-1301', 700, 300),
      node('I-1302', 700, 300),
      node('I-1302', 165, 405),
    ]
    const segments = [edge(0, 1, 'CORRIDOR'), edge(1, 2, 'ELEVATOR'), edge(2, 3, 'CORRIDOR')]
    expect(turnByTurnSteps(nodes, segments, 'Blood Collection', label)).toEqual([
      'เดินตามเส้นสายส้มบนผัง',
      'ใช้ลิฟต์ไปชั้น 2',
      'เดินตามเส้นสายส้มบนผัง',
      'ถึงBlood Collection — จุดหมายของคุณ',
    ])
  })

  it('names stairs instead of a lift for stair transitions', () => {
    const nodes = [node('I-1301', 700, 300), node('I-1302', 700, 300)]
    const segments = [edge(0, 1, 'STAIRS')]
    expect(turnByTurnSteps(nodes, segments, 'Lab', label)).toEqual([
      'ใช้บันไดไปชั้น 2',
      'ถึงLab — จุดหมายของคุณ',
    ])
  })

  it('still announces arrival when the patient already stands at the destination', () => {
    const nodes = [node('I-1301', 885, 190)]
    expect(turnByTurnSteps(nodes, [], 'Pharmacy', label)).toEqual([
      'เดินตามเส้นสายส้มบนผัง',
      'ถึงPharmacy — จุดหมายของคุณ',
    ])
  })

  it('returns no cues for an empty route', () => {
    expect(turnByTurnSteps([], [], 'Pharmacy', label)).toEqual([])
  })
})

describe('routeBounds', () => {
  it('is the bbox of every line on the floor', () => {
    expect(
      routeBounds([
        [
          { x: 150, y: 190 },
          { x: 445, y: 330 },
        ],
        [
          { x: 600, y: 100 },
          { x: 885, y: 190 },
        ],
      ]),
    ).toEqual({ from: { x: 150, y: 100 }, to: { x: 885, y: 330 } })
  })

  it('is null with no points', () => {
    expect(routeBounds([])).toBeNull()
    expect(routeBounds([[]])).toBeNull()
  })
})
