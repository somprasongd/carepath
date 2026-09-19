import { describe, expect, it } from 'vitest'
import type { Journey, JourneyStep } from './queries'
import {
  stepProgress,
  stepStatusLabel,
  stepTone,
  syncedAtLabel,
  visitPosition,
  visitStatusLabel,
  visitTone,
} from './staff'

function step(overrides: Partial<JourneyStep>): JourneyStep {
  return { sequence: 1, serviceCode: 'LAB', status: 'READY', servicePointId: null, ...overrides }
}

function journey(overrides: Partial<Journey>): Journey {
  return {
    visitId: 'VISIT-001',
    patientRef: 'PAT-001',
    status: 'ACTIVE',
    completed: false,
    steps: [step({})],
    current: null,
    next: step({ sequence: 1 }),
    syncedAt: '2026-09-19T04:05:00Z',
    ...overrides,
  }
}

describe('stepTone / visitTone', () => {
  it('maps every contract status to a badge tone', () => {
    expect(stepTone('COMPLETED')).toBe('routable')
    expect(stepTone('STARTED')).toBe('busy')
    expect(stepTone('READY')).toBe('ready')
    expect(stepTone('PENDING')).toBe('quiet')
    expect(stepTone('CANCELLED')).toBe('quiet')
    expect(visitTone('ACTIVE')).toBe('busy')
    expect(visitTone('COMPLETED')).toBe('routable')
    expect(visitTone('CANCELLED')).toBe('quiet')
  })

  it('falls back to quiet for an unknown status', () => {
    expect(stepTone('HOLD')).toBe('quiet')
    expect(visitTone('HOLD')).toBe('quiet')
  })
})

describe('status labels', () => {
  it('translates known statuses and passes unknown ones through', () => {
    expect(stepStatusLabel('READY')).toBe('พร้อมให้บริการ')
    expect(stepStatusLabel('HOLD')).toBe('HOLD')
    expect(visitStatusLabel('ACTIVE')).toBe('กำลังรับบริการ')
    expect(visitStatusLabel('HOLD')).toBe('HOLD')
  })
})

describe('visitPosition', () => {
  it('leads with the started step', () => {
    const j = journey({ current: step({ sequence: 2, serviceCode: 'LAB', status: 'STARTED' }) })
    expect(visitPosition(j)).toBe('กำลังเจาะเลือด')
  })

  it('falls back to the next ready step', () => {
    const j = journey({ next: step({ sequence: 3, serviceCode: 'PHARMACY', status: 'READY' }) })
    expect(visitPosition(j)).toBe('ถัดไป · รับยา')
  })

  it('is explicit for finished and cancelled visits', () => {
    expect(visitPosition(journey({ completed: true, status: 'COMPLETED' }))).toBe(
      'เสร็จสิ้นทุกขั้นตอน',
    )
    expect(visitPosition(journey({ status: 'CANCELLED', next: null }))).toBe(
      'การมารับบริการถูกยกเลิก',
    )
  })

  it('shows the raw code for an unmapped service', () => {
    const j = journey({ next: step({ serviceCode: 'MYSTERY' }) })
    expect(visitPosition(j)).toBe('ถัดไป · MYSTERY')
  })
})

describe('stepProgress', () => {
  it('counts completed steps over all steps', () => {
    const j = journey({
      steps: [
        step({ sequence: 1, status: 'COMPLETED' }),
        step({ sequence: 2, status: 'COMPLETED' }),
        step({ sequence: 3, status: 'STARTED' }),
        step({ sequence: 4, status: 'PENDING' }),
        step({ sequence: 5, status: 'CANCELLED' }),
      ],
    })
    expect(stepProgress(j)).toBe('2/5')
  })
})

describe('syncedAtLabel', () => {
  it('formats the sync timestamp as a clock time', () => {
    expect(syncedAtLabel('2026-09-19T04:05:00Z')).toMatch(/^\d{2}:\d{2}$/)
  })
})
