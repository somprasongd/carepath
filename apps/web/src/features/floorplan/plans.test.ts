import { describe, expect, it } from 'vitest'
import { floorLabelFor, floorPlanHasPlace, planUrlFor } from './plans'
import type { FloorPlanRef } from './queries'

// What GET /api/v1/floors returns (ADR-0015): the floor's code, which the
// label is built from, and the immutable URL of its current drawing. The
// third floor is the state the app has to handle honestly — a floor the map
// model knows about that no plan has been uploaded for.
const floors: FloorPlanRef[] = [
  {
    floorId: 'I-1301',
    buildingId: 'BLD-I13',
    code: '1',
    name: 'Ground Floor',
    levelOrder: 1,
    viewBox: '0 0 1600 900',
    planUrl: '/api/v1/floors/I-1301/plan/abc123.svg',
  },
  {
    floorId: 'I-1302',
    buildingId: 'BLD-I13',
    code: '2',
    name: 'Upper Floor',
    levelOrder: 2,
    viewBox: '0 0 1600 900',
    planUrl: '/api/v1/floors/I-1302/plan/def456.svg',
  },
  { floorId: 'I-1303', buildingId: 'BLD-I13', code: '3', name: 'Undrawn', levelOrder: 3 },
]

describe('floorLabelFor', () => {
  it('labels a floor by its code in each locale', () => {
    expect(floorLabelFor('I-1301', 'th', floors)).toBe('ชั้น 1')
    expect(floorLabelFor('I-1302', 'th', floors)).toBe('ชั้น 2')
    expect(floorLabelFor('I-1301', 'en', floors)).toBe('Floor 1')
    expect(floorLabelFor('I-1302', 'en', floors)).toBe('Floor 2')
  })

  // The code is server data and the wording is catalog data, so a floor
  // added by an admin is labelled without touching i18n at all.
  it('labels a floor the app has never heard of, from its code alone', () => {
    const added: FloorPlanRef[] = [
      { floorId: 'I-1404', buildingId: 'BLD-I14', code: '4', name: 'New Wing', levelOrder: 4 },
    ]
    expect(floorLabelFor('I-1404', 'th', added)).toBe('ชั้น 4')
    expect(floorLabelFor('I-1404', 'en', added)).toBe('Floor 4')
  })

  it('passes the raw floor id through for an unknown floor rather than inventing a label', () => {
    expect(floorLabelFor('I-9999', 'th', floors)).toBe('I-9999')
    expect(floorLabelFor('I-9999', 'en', floors)).toBe('I-9999')
  })
})

describe('planUrlFor', () => {
  it('returns the floor plan URL the listing carries', () => {
    expect(planUrlFor('I-1301', floors)).toBe('/api/v1/floors/I-1301/plan/abc123.svg')
  })

  it('returns undefined for a floor with no plan yet', () => {
    expect(planUrlFor('I-1303', floors)).toBeUndefined()
  })

  it('returns undefined for an unknown floor', () => {
    expect(planUrlFor('I-9999', floors)).toBeUndefined()
  })
})

describe('floorPlanHasPlace', () => {
  const svg =
    '<svg><g data-floor="I-1301">' +
    '<rect data-place-id="PHARMACY-01"/><rect data-place-id="REG-01"/>' +
    '</g></svg>'

  it('finds a place the plan draws', () => {
    expect(floorPlanHasPlace(svg, 'PHARMACY-01')).toBe(true)
    expect(floorPlanHasPlace(svg, 'REG-01')).toBe(true)
  })

  it('rejects a place the plan does not draw', () => {
    expect(floorPlanHasPlace(svg, 'LAB-01')).toBe(false)
    expect(floorPlanHasPlace(svg, 'NOT-A-PLACE')).toBe(false)
  })
})
