import { describe, expect, it } from 'vitest'
import { format, messagesFor } from '@/i18n'
import type { Journey, JourneyStep } from '@/features/visit'
import type { Place } from './queries'
import { amenityDestinationPlan, navigatePlanForJourney } from './destination'

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
      // The destination name rides the catalog by code (ADR-0012 §1), not
      // the server-authored name in the fixture.
      name: th['sp.PHARMACY'],
      subtitle: format(th, 'navigate.subtitle', {
        floor: format(th, 'common.floor', { code: '1' }),
        name: th['sp.PHARMACY'],
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
      // sp.PHARMACY's catalog name — the fixture's server name happens to
      // match it here; the unmapped fallback is pinned in its own test.
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

  it('falls back to the server-authored name for an unmapped service point code (#94 AC)', () => {
    // The live case this pins: CLINIC:MED's server name is Thai; in English
    // the catalog supplies "Internal Medicine" instead — and when a code has
    // no catalog entry at all, the server name shows as-is, Thai or not,
    // because an honest name beats a fabricated one (ADR-0012 §1).
    const plan = navigatePlanForJourney(
      journeyWithRecommended({
        stepKey: 'XRAY',
        sequence: 2,
        kind: 'XRAY',
        status: 'READY',
        servicePointId: 'SP-X',
        servicePoint: { id: 'SP-X', code: 'X', name: 'จุดกายภาพบำบัด', placeId: 'NOPE-01' },
      }),
      false,
      'en',
    )

    expect(plan).toEqual({ state: 'unsupported', title: 'Directions to X-ray', name: 'จุดกายภาพบำบัด' })
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

describe('amenityDestinationPlan', () => {
  const restroom: Place = {
    id: 'RESTROOM-01',
    floorId: 'I-1301',
    name: 'Restroom (Ground)',
    type: 'RESTROOM',
    x: 318,
    y: 190,
    entryNodeId: 'I-1301/node-ramp',
    floor: FLOOR_1,
  }

  it('plans a routable amenity with the cataloged kind label, not the staff name', () => {
    // The seeded place's name is staff-facing ("Restroom (Ground)"); the
    // patient gets the catalog's kind label (ADR-0012) — pinned in real
    // English here, from the catalog in the Thai test below.
    const plan = amenityDestinationPlan(restroom, 'en')

    expect(plan).toEqual({
      state: 'plan',
      title: 'Directions to Restroom',
      name: 'Restroom',
      subtitle: 'Floor 1 · Restroom · RESTROOM-01',
      floorId: 'I-1301',
      floorLabel: 'Floor 1',
      placeId: 'RESTROOM-01',
      // No servicePointCode: amenity routes ride toPlace on the place id.
      x: 318,
      y: 190,
    })
  })

  it('composes the same plan in Thai from the catalog', () => {
    const th = messagesFor('th')
    const plan = amenityDestinationPlan({ ...restroom, floor: FLOOR_2, floorId: 'I-1302' }, 'th')

    expect(plan).toEqual({
      state: 'plan',
      title: format(th, 'navigate.routeTitle', { name: th['amenity.kind.RESTROOM'] }),
      name: th['amenity.kind.RESTROOM'],
      subtitle: format(th, 'navigate.subtitle', {
        floor: format(th, 'common.floor', { code: '2' }),
        name: th['amenity.kind.RESTROOM'],
        place: 'RESTROOM-01',
      }),
      floorId: 'I-1302',
      floorLabel: format(th, 'common.floor', { code: '2' }),
      placeId: 'RESTROOM-01',
      x: 318,
      y: 190,
    })
  })

  it('falls back to the staff name for an uncataloged kind', () => {
    const plan = amenityDestinationPlan({ ...restroom, type: 'AMENITY', name: 'ร้านดอกไม้' }, 'en')
    // The cast is the case being pinned: a server contract newer than the
    // generated schema (a new Place.type the catalog has not mapped yet) —
    // the server name must show rather than a fabricated label.
    const unmapped = amenityDestinationPlan(
      { ...restroom, type: 'VENDING' as Place['type'] },
      'en',
    )

    expect(plan).toMatchObject({ state: 'plan', name: 'Amenity' })
    expect(unmapped).toMatchObject({ state: 'plan', name: 'Restroom (Ground)' })
  })

  it('is honestly unsupported for a place with no entry node', () => {
    const { entryNodeId, ...orphan } = restroom
    void entryNodeId
    const plan = amenityDestinationPlan(orphan, 'en')

    expect(plan).toEqual({
      state: 'unsupported',
      title: 'Directions to Restroom',
      name: 'Restroom',
    })
  })

  it('has no destination while the place row is still loading', () => {
    expect(amenityDestinationPlan(undefined, 'th')).toEqual({ state: 'no-destination' })
  })
})
