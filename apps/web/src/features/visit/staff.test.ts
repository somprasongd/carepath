import { describe, expect, it } from 'vitest'
import type { Journey, JourneyStep } from './queries'
import {
  stepAction,
  stepProgress,
  stepStatusLabel,
  stepTone,
  syncedAtLabel,
  transitionErrorText,
  visitPosition,
  visitStatusLabel,
  visitTone,
} from './staff'

function step(overrides: Partial<JourneyStep>): JourneyStep {
  return {
    stepKey: 'LAB:1', sequence: 1, kind: 'LAB', status: 'READY',
    clinicCode: null, round: null, orderRefs: [], servicePointId: null,
    ...overrides,
  }
}

function journey(overrides: Partial<Journey>): Journey {
  const ready = step({})
  return {
    visitId: 'VISIT-001',
    patientRef: 'PAT-001',
    status: 'ACTIVE',
    completed: false,
    steps: [ready],
    actionable: [ready],
    recommended: ready,
    syncedAt: '2026-09-19T04:05:00Z',
    ...overrides,
  }
}

describe('stepTone / visitTone', () => {
  it('maps every contract status to a badge tone', () => {
    expect(stepTone('COMPLETED')).toBe('routable')
    expect(stepTone('STARTED')).toBe('busy')
    expect(stepTone('READY')).toBe('ready')
    expect(stepTone('WAITING')).toBe('quiet')
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
    expect(stepStatusLabel('WAITING')).toBe('รอผลตรวจ')
    expect(stepStatusLabel('HOLD')).toBe('HOLD')
    expect(visitStatusLabel('ACTIVE')).toBe('กำลังรับบริการ')
    expect(visitStatusLabel('HOLD')).toBe('HOLD')
  })
})

describe('visitPosition', () => {
  it('leads with a STARTED step over the recommended one', () => {
    const j = journey({
      steps: [
        step({ stepKey: 'CLINIC:MED:1', kind: 'CLINIC', clinicCode: 'MED', round: 1, status: 'STARTED' }),
      ],
      actionable: [],
      recommended: null,
    })
    expect(visitPosition(j)).toBe('กำลังพบแพทย์ · อายุรกรรม')
  })

  it('falls back to the recommended actionable step', () => {
    const j = journey({ recommended: step({ stepKey: 'PHARMACY', kind: 'PHARMACY', status: 'READY' }) })
    expect(visitPosition(j)).toBe('ถัดไป · รับยา')
  })

  it('is explicit for finished and cancelled visits', () => {
    expect(visitPosition(journey({ completed: true, status: 'COMPLETED' }))).toBe(
      'เสร็จสิ้นทุกขั้นตอน',
    )
    expect(
      visitPosition(journey({ status: 'CANCELLED', steps: [], actionable: [], recommended: null })),
    ).toBe('การมารับบริการถูกยกเลิก')
  })

  it('shows the raw clinic code for an unmapped clinic', () => {
    const j = journey({
      recommended: step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1 }),
    })
    expect(visitPosition(j)).toBe('ถัดไป · พบแพทย์ · PED')
  })
})

describe('stepProgress', () => {
  it('counts completed steps over all steps', () => {
    const j = journey({
      steps: [
        step({ stepKey: 'S1', status: 'COMPLETED' }),
        step({ stepKey: 'S2', status: 'COMPLETED' }),
        step({ stepKey: 'S3', status: 'STARTED' }),
        step({ stepKey: 'S4', status: 'PENDING' }),
        step({ stepKey: 'S5', status: 'CANCELLED' }),
      ],
    })
    expect(stepProgress(j)).toBe('2/5')
  })
})

describe('syncedAtLabel', () => {
  it('formats the sync clock through the shared formatter, Thai marker included', () => {
    // The staff console pins 'th' (ADR-0012 §2); the time itself is
    // timezone-dependent, so assert shape + marker, not a fixed clock.
    expect(syncedAtLabel('2026-09-19T04:05:00Z', 'th')).toMatch(/^\d{2}:\d{2} น\.$/)
    expect(syncedAtLabel('2026-09-19T04:05:00Z', 'en')).toMatch(/^\d{2}:\d{2}$/)
  })
})

describe('stepAction (#38)', () => {
  it('starts a READY step and completes a STARTED one', () => {
    expect(stepAction('READY')).toEqual({ to: 'STARTED', label: 'เริ่มขั้นตอน' })
    expect(stepAction('STARTED')).toEqual({ to: 'COMPLETED', label: 'ทำเสร็จแล้ว' })
  })

  it('offers nothing for statuses with no legal forward move', () => {
    expect(stepAction('PENDING')).toBeNull()
    expect(stepAction('WAITING')).toBeNull()
    expect(stepAction('COMPLETED')).toBeNull()
    expect(stepAction('CANCELLED')).toBeNull()
  })
})

describe('transitionErrorText (#38)', () => {
  it('blames a stale projection on 409 and tells staff to retry', () => {
    expect(transitionErrorText(409, 'step is not READY')).toBe(
      'ตอนนี้เปลี่ยนสถานะนี้ไม่ได้ (step is not READY) — รีเฟรชแล้วลองใหม่',
    )
  })

  it('labels a bad request and keeps the server detail', () => {
    expect(transitionErrorText(400, 'unknown target status')).toBe(
      'คำสั่งไม่ถูกต้อง (unknown target status)',
    )
  })

  it('covers network loss and unexpected statuses without losing the detail', () => {
    expect(transitionErrorText(0, 'เชื่อมต่อ API ไม่ได้ (Error)')).toContain('เปลี่ยนสถานะไม่สำเร็จ')
    expect(transitionErrorText(500, 'boom')).toBe('เปลี่ยนสถานะไม่สำเร็จ (สถานะ 500 · boom)')
    expect(transitionErrorText(502, '')).toBe('เปลี่ยนสถานะไม่สำเร็จ (สถานะ 502)')
  })
})
