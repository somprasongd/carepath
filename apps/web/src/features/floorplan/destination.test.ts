import { describe, expect, it } from 'vitest'
import { format, messagesFor } from '@/i18n'
import type { Journey, JourneyStep } from '@/features/visit'
import { navigatePlanForJourney } from './destination'

const FLOOR_1 = {
  id: 'I-1301',
  buildingId: 'BLD-I13',
  code: '1',
  name: 'Ground Floor',
  levelOrder: 1,
}
const FLOOR_2 = {
  id: 'I-1302',
  buildingId: 'BLD-I13',
  code: '2',
  name: 'Upper Floor',
  levelOrder: 2,
}

function journeyWithRecommended(recommended: JourneyStep | null): Journey {
  return {
    visitId: 'VISIT-001',
    patientRef: 'PT-001',
    status: 'ACTIVE',
    completed: false,
    steps: recommended ? [recommended] : [],
    actionable: recommended ? [recommended] : [],
    recommended,
    syncedAt: '2026-09-19T10:00:00Z',
  }
}

describe('navigatePlanForJourney', () => {
  it('stays pending while the journey loads', () => {
    expect(navigatePlanForJourney(undefined, true, 'th')).toEqual({ state: 'pending' })
    expect(navigatePlanForJourney(undefined, true, 'en')).toEqual({ state: 'pending' })
  })

  it('has no destination when nothing is recommended', () => {
    expect(navigatePlanForJourney(undefined, false, 'th')).toEqual({ state: 'no-destination' })

    const finished = journeyWithRecommended(null)
    expect(navigatePlanForJourney(finished, false, 'en')).toEqual({ state: 'no-destination' })
  })

  it('plans a mapped ground-floor destination from the journey (#25 AC2/AC3)', () => {
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'PHARMACY',
        sequence: 2,
        kind: 'PHARMACY',
        status: 'READY',
        servicePointId: 'SP-PHARMACY',
        servicePoint: {
          id: 'SP-PHARMACY',
          code: 'PHARMACY',
          name: 'Pharmacy',
          placeId: 'PHARMACY-01',
          place: {
            id: 'PHARMACY-01',
            floorId: 'I-1301',
            name: 'Pharmacy',
            type: 'ROOM',
            x: 885,
            y: 190,
            entryNodeId: 'I-1301/node-pharmacy',
            floor: FLOOR_1,
          },
        },
      }),
      false,
      'th',
    )

    // Thai expectations come from the catalog so copy tweaks do not rot the
    // test (the default-Thai screen itself is the thing pinned below in en).
    const th = messagesFor('th')
    expect(plan).toEqual({
      state: 'plan',
      title: format(th, 'navigate.routeTitle', { name: th['step.title.PHARMACY'] }),
      name: 'Pharmacy',
      subtitle: format(th, 'navigate.subtitle', {
        floor: format(th, 'common.floor', { code: '1' }),
        name: 'Pharmacy',
        place: 'PHARMACY-01',
      }),
      floorId: 'I-1301',
      floorLabel: format(th, 'common.floor', { code: '1' }),
      placeId: 'PHARMACY-01',
      servicePointCode: 'PHARMACY',
      x: 885,
      y: 190,
    })
  })

  it('composes the same plan in real English for the en locale', () => {
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'PHARMACY',
        sequence: 2,
        kind: 'PHARMACY',
        status: 'READY',
        servicePointId: 'SP-PHARMACY',
        servicePoint: {
          id: 'SP-PHARMACY',
          code: 'PHARMACY',
          name: 'Pharmacy',
          placeId: 'PHARMACY-01',
          place: {
            id: 'PHARMACY-01',
            floorId: 'I-1301',
            name: 'Pharmacy',
            type: 'ROOM',
            x: 885,
            y: 190,
            entryNodeId: 'I-1301/node-pharmacy',
            floor: FLOOR_1,
          },
        },
      }),
      false,
      'en',
    )

    expect(plan).toEqual({
      state: 'plan',
      title: 'Directions to Medication pickup',
      name: 'Pharmacy',
      subtitle: 'Floor 1 · Pharmacy · PHARMACY-01',
      floorId: 'I-1301',
      floorLabel: 'Floor 1',
      placeId: 'PHARMACY-01',
      servicePointCode: 'PHARMACY',
      x: 885,
      y: 190,
    })
  })

  it('plans an upper-floor destination — the other seeded plan', () => {
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'LAB:ORD-118',
        sequence: 2,
        kind: 'LAB',
        status: 'READY',
        servicePointId: 'SP-LAB',
        servicePoint: {
          id: 'SP-LAB',
          code: 'LAB',
          name: 'Blood Collection',
          placeId: 'LAB-01',
          place: {
            id: 'LAB-01',
            floorId: 'I-1302',
            name: 'Blood Collection',
            type: 'ROOM',
            x: 165,
            y: 405,
            entryNodeId: 'I-1302/node-blood-collection',
            floor: FLOOR_2,
          },
        },
      }),
      false,
      'th',
    )

    const th = messagesFor('th')
    expect(plan).toMatchObject({
      state: 'plan',
      title: format(th, 'navigate.routeTitle', { name: th['step.title.LAB'] }),
      floorId: 'I-1302',
      floorLabel: format(th, 'common.floor', { code: '2' }),
      placeId: 'LAB-01',
    })
  })

  it('falls back to the honest unsupported state when the place is unmapped', () => {
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'XRAY',
        sequence: 2,
        kind: 'XRAY',
        status: 'READY',
        servicePointId: 'SP-X',
        servicePoint: { id: 'SP-X', code: 'X', name: 'Somewhere', placeId: 'NOPE-01' },
      }),
      false,
      'th',
    )

    const th = messagesFor('th')
    expect(plan).toEqual({
      state: 'unsupported',
      title: format(th, 'navigate.routeTitle', { name: th['step.title.XRAY'] }),
      name: 'Somewhere',
    })
  })

  it('falls back to the honest unsupported state for a floor with no plan asset', () => {
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'PHARMACY',
        sequence: 2,
        kind: 'PHARMACY',
        status: 'READY',
        servicePointId: 'SP-X',
        servicePoint: {
          id: 'SP-X',
          code: 'X',
          name: 'Somewhere',
          placeId: 'X-01',
          place: {
            id: 'X-01',
            floorId: 'I-9999',
            name: 'Somewhere',
            type: 'ROOM',
            floor: { id: 'I-9999', buildingId: 'BLD-Z', code: '9', name: 'Ninth Floor', levelOrder: 9 },
          },
        },
      }),
      false,
      'en',
    )

    expect(plan).toEqual({
      state: 'unsupported',
      title: 'Directions to Medication pickup',
      name: 'Somewhere',
    })
  })
})
