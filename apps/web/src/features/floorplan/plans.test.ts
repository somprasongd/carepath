import { describe, expect, it } from 'vitest'
import { floorLabelFor, floorPlanFor, floorPlanHasPlace } from './plans'

describe('floorPlanFor', () => {
  it('serves the ground-floor plan keyed by its floor id', () => {
    const svg = floorPlanFor('I-1301')
    expect(svg).toBeTruthy()
    expect(svg).toContain('data-floor="I-1301"')
  })

  it('serves the upper-floor plan keyed by its floor id', () => {
    const svg = floorPlanFor('I-1302')
    expect(svg).toBeTruthy()
    expect(svg).toContain('data-floor="I-1302"')
  })

  it('returns null for a floor with no plan', () => {
    expect(floorPlanFor('I-9999')).toBeNull()
  })
})

describe('floorLabelFor', () => {
  it('labels a floor by its code in each locale', () => {
    expect(floorLabelFor('I-1301', 'th')).toBe('ชั้น 1')
    expect(floorLabelFor('I-1302', 'th')).toBe('ชั้น 2')
    expect(floorLabelFor('I-1301', 'en')).toBe('Floor 1')
    expect(floorLabelFor('I-1302', 'en')).toBe('Floor 2')
  })

  it('passes the raw floor id through for an unknown floor rather than inventing a label', () => {
    expect(floorLabelFor('I-9999', 'th')).toBe('I-9999')
    expect(floorLabelFor('I-9999', 'en')).toBe('I-9999')
  })
})

describe('floorPlanHasPlace', () => {
  const svg = floorPlanFor('I-1301') ?? ''

  it('finds a place the plan draws', () => {
    expect(floorPlanHasPlace(svg, 'PHARMACY-01')).toBe(true)
    expect(floorPlanHasPlace(svg, 'REG-01')).toBe(true)
  })

  it('rejects a place the plan does not draw', () => {
    expect(floorPlanHasPlace(svg, 'LAB-01')).toBe(false)
    expect(floorPlanHasPlace(svg, 'NOT-A-PLACE')).toBe(false)
  })
})
