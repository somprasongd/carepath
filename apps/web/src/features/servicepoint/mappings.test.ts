import { describe, expect, it } from 'vitest'
import type { ServicePoint } from './queries'
import { toServicePointView, zoneForService } from './mappings'

function servicePoint(overrides: Partial<ServicePoint> = {}): ServicePoint {
  return {
    id: 'SP-LAB',
    code: 'LAB',
    name: 'Laboratory',
    placeId: 'LAB-01',
    place: {
      id: 'LAB-01',
      floorId: 'I-1302',
      name: 'Blood Collection',
      type: 'ROOM',
      x: 165,
      y: 405,
      entryNodeId: 'node-blood-collection',
      floor: {
        id: 'I-1302',
        buildingId: 'BLD-I13',
        code: '2',
        name: 'Upper Floor',
        levelOrder: 2,
      },
    },
    ...overrides,
  }
}

describe('toServicePointView', () => {
  it('resolves a mapped place to place · name · floor, routable via its entry node', () => {
    const view = toServicePointView(servicePoint())

    expect(view.service).toBe('LAB')
    expect(view.serviceName).toBe('Laboratory')
    expect(view.target).toBe('LAB-01 · Blood Collection · ชั้น 2')
    expect(view.zone).toBe('diagnostic')
    expect(view.status).toBe('routable')
  })

  it('shows the explicit unmapped state when the place is not in the map', () => {
    const view = toServicePointView(servicePoint({ place: null }))

    expect(view.target).toBe('ยังไม่อยู่ในผังอาคาร')
    expect(view.status).toBe('not-routable')
  })

  it('stays not-routable when the place has no navigation entry node', () => {
    const view = toServicePointView(
      servicePoint({
        place: {
          ...servicePoint().place!,
          entryNodeId: undefined,
        },
      }),
    )

    expect(view.target).toBe('LAB-01 · Blood Collection · ชั้น 2')
    expect(view.status).toBe('not-routable')
  })

  it.each([
    ['REGISTRATION', 'public'],
    ['DOCTOR', 'opd'],
    ['XRAY', 'diagnostic'],
    ['PHARMACY', 'pharmacy'],
    ['NURSE', 'public'],
  ] as const)('maps service %s to zone %s', (code, zone) => {
    expect(toServicePointView(servicePoint({ code })).zone).toBe(zone)
  })
})

describe('zoneForService', () => {
  // #87: the overview rows carry whatever code the service point table has,
  // including ADR-0009 binding prefixes — classified by the same call.
  it.each([
    ['ORDERTYPE:LAB', 'diagnostic'],
    ['ORDERTYPE:XRAY', 'diagnostic'],
    ['CLINIC:MED', 'opd'],
    ['CASHIER', 'public'],
    ['REGISTRATION', 'public'],
    ['PHARMACY', 'pharmacy'],
    ['WELLNESS', 'public'],
  ] as const)('maps %s to zone %s', (code, zone) => {
    expect(zoneForService(code)).toBe(zone)
  })
})
