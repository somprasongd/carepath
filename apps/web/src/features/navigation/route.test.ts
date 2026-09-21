import { describe, expect, it } from 'vitest'
import { format, messagesFor } from '@/i18n'
import { floorLabelFor, type FloorPlanRef } from '@/features/floorplan'
import {
  currentLocationLabel,
  routeBounds,
  routeOriginOnFloor,
  routePolylinesByFloor,
  turnByTurnSteps,
  type NavEdge,
  type NavNode,
} from './route'

// The floor list as GET /api/v1/floors serves it (ADR-0015): the label is
// built from the floor's own code, so these cases no longer lean on a
// hardcoded floorId -> code table inside the app.
const FLOORS: FloorPlanRef[] = [
  { floorId: 'I-1301', buildingId: 'BLD-I13', code: '1', name: 'Ground Floor', levelOrder: 1 },
  { floorId: 'I-1302', buildingId: 'BLD-I13', code: '2', name: 'Upper Floor', levelOrder: 2 },
]

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
  const thFloor = (floorId: string) => floorLabelFor(floorId, 'th', FLOORS)
  const enFloor = (floorId: string) => floorLabelFor(floorId, 'en', FLOORS)
  const th = messagesFor('th')

  it('collapses a same-floor walk into one cue plus arrival', () => {
    const nodes = [node('I-1301', 150, 190), node('I-1301', 318, 190), node('I-1301', 885, 190)]
    const segments = [edge(0, 1, 'CORRIDOR'), edge(1, 2, 'CORRIDOR')]
    expect(turnByTurnSteps(nodes, segments, 'Pharmacy', thFloor, 'th')).toEqual([
      th['navigate.cue.followLine'],
      format(th, 'navigate.cue.arrive', { name: 'Pharmacy' }),
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
    expect(turnByTurnSteps(nodes, segments, 'Blood Collection', thFloor, 'th')).toEqual([
      th['navigate.cue.followLine'],
      format(th, 'navigate.cue.elevator', { floor: thFloor('I-1302') }),
      th['navigate.cue.followLine'],
      format(th, 'navigate.cue.arrive', { name: 'Blood Collection' }),
    ])
  })

  it('speaks the whole cue set in real English for the en locale', () => {
    const nodes = [
      node('I-1301', 150, 190),
      node('I-1301', 700, 300),
      node('I-1302', 700, 300),
      node('I-1302', 165, 405),
    ]
    const segments = [edge(0, 1, 'CORRIDOR'), edge(1, 2, 'ELEVATOR'), edge(2, 3, 'CORRIDOR')]
    expect(turnByTurnSteps(nodes, segments, 'Blood Collection', enFloor, 'en')).toEqual([
      'Follow the orange line on the map',
      'Take the elevator to Floor 2',
      'Follow the orange line on the map',
      'Arrive at Blood Collection — your destination',
    ])
  })

  it('names stairs instead of a lift for stair transitions', () => {
    const nodes = [node('I-1301', 700, 300), node('I-1302', 700, 300)]
    const segments = [edge(0, 1, 'STAIRS')]
    expect(turnByTurnSteps(nodes, segments, 'Lab', thFloor, 'th')).toEqual([
      format(th, 'navigate.cue.stairs', { floor: thFloor('I-1302') }),
      format(th, 'navigate.cue.arrive', { name: 'Lab' }),
    ])
  })

  it('still announces arrival when the patient already stands at the destination', () => {
    const nodes = [node('I-1301', 885, 190)]
    expect(turnByTurnSteps(nodes, [], 'Pharmacy', thFloor, 'th')).toEqual([
      th['navigate.cue.followLine'],
      format(th, 'navigate.cue.arrive', { name: 'Pharmacy' }),
    ])
  })

  it('returns no cues for an empty route', () => {
    expect(turnByTurnSteps([], [], 'Pharmacy', thFloor, 'th')).toEqual([])
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

describe('routeOriginOnFloor', () => {
  it('is the first node when the route starts on the displayed floor', () => {
    expect(
      routeOriginOnFloor([node('I-1301', 150, 190), node('I-1301', 885, 190)], 'I-1301'),
    ).toEqual({ x: 150, y: 190 })
  })

  it('is null when the route starts on another floor — the mark would lie', () => {
    expect(
      routeOriginOnFloor([node('I-1302', 300, 240), node('I-1301', 885, 190)], 'I-1301'),
    ).toBeNull()
  })

  it('is null with no nodes', () => {
    expect(routeOriginOnFloor([], 'I-1301')).toBeNull()
  })
})

describe('currentLocationLabel', () => {
  const floorLabel = (floorId: string) => floorLabelFor(floorId, 'en', FLOORS)
  const thFloorLabel = (floorId: string) => floorLabelFor(floorId, 'th', FLOORS)

  it('states the floor and an exact source without a zone', () => {
    expect(
      currentLocationLabel({ floorId: 'I-1301', zone: null, source: 'QR' }, floorLabel, 'en'),
    ).toBe('Current location · Floor 1 · QR scan')
  })

  it('carries the zone and source for a probabilistic fix', () => {
    expect(
      currentLocationLabel({ floorId: 'I-1301', zone: 'PUBLIC', source: 'ZIGBEE' }, floorLabel, 'en'),
    ).toBe('Current location · Floor 1 · Zone PUBLIC · Zigbee')
  })

  it('never surfaces an unknown provider name', () => {
    expect(
      currentLocationLabel({ floorId: 'I-1301', zone: null, source: 'SMOKE_SIGNAL' }, floorLabel, 'en'),
    ).toBe('Current location · Floor 1')
  })

  it('composes the same line from the Thai catalog (default locale)', () => {
    expect(
      currentLocationLabel({ floorId: 'I-1301', zone: 'PUBLIC', source: 'QR' }, thFloorLabel, 'th'),
    ).toBe(
      format(messagesFor('th'), 'navigate.locationNowAt', { floor: thFloorLabel('I-1301') }) +
        ' · ' +
        format(messagesFor('th'), 'navigate.zone', { zone: 'PUBLIC' }) +
        ' · ' +
        messagesFor('th')['navigate.source.QR'],
    )
  })
})
