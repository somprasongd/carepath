import { describe, expect, it } from 'vitest'
import type { Journey } from './queries'
import { journeyQueryOptions, transitionStepUrl } from './queries'

function journey(overrides: Partial<Journey> = {}): Journey {
  return {
    visitId: 'VISIT-001',
    patientRef: 'PATIENT-DEMO-001',
    status: 'ACTIVE',
    completed: false,
    steps: [],
    actionable: [],
    recommended: null,
    syncedAt: '2026-09-20T02:00:00Z',
    ...overrides,
  }
}

// The interval is computed from the cached journey, so the test calls it the
// way TanStack does — with the query's current state.
const intervalFor = (data: Journey | undefined) => {
  const { refetchInterval } = journeyQueryOptions('VISIT-001')
  expect(refetchInterval).toBeTypeOf('function')
  const interval = refetchInterval as (query: { state: { data?: Journey } }) =>
    | number
    | false
    | undefined
  return interval({ state: { data } })
}

describe('transitionStepUrl (#38 staff controls)', () => {
  it('keeps the stepKey colons raw — the API route param does not decode %3A', () => {
    expect(transitionStepUrl('VISIT-002', 'CLINIC:MED:1')).toBe(
      '/api/v1/journeys/VISIT-002/steps/CLINIC:MED:1/transition',
    )
    expect(transitionStepUrl('VISIT-002', 'XRAY:1')).toBe(
      '/api/v1/journeys/VISIT-002/steps/XRAY:1/transition',
    )
  })

  it('still encodes the visit id, which is a plain opaque token', () => {
    expect(transitionStepUrl('VISIT 2', 'REGISTRATION')).toBe(
      '/api/v1/journeys/VISIT%202/steps/REGISTRATION/transition',
    )
  })
})

describe('journeyQueryOptions realtime (#36)', () => {
  it('polls the journey every 15s while the visit can still change', () => {
    expect(intervalFor(journey())).toBe(15_000)
  })

  it('keeps polling before the first payload lands', () => {
    expect(intervalFor(undefined)).toBe(15_000)
  })

  it('stops polling once the visit is final — completed or cancelled', () => {
    expect(intervalFor(journey({ completed: true }))).toBe(false)
    expect(intervalFor(journey({ status: 'CANCELLED' }))).toBe(false)
  })

  it('keys the query by visit alone, so every screen reads one cache entry', () => {
    expect(journeyQueryOptions('VISIT-001').queryKey).toEqual(['journey', 'VISIT-001'])
    expect(journeyQueryOptions('VISIT-001').queryKey).toEqual(
      journeyQueryOptions('VISIT-001').queryKey,
    )
  })
})
