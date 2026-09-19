import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import type { VisitView } from './queries'
import { serviceCodeOf, thaiStepTitle, toJourneySteps, visitLoadErrorMessage } from './journey'

/** Mirrors the VISIT-001 shape Mock HIS serves — steps + resolved next. */
function visitView(overrides: Partial<VisitView> = {}): VisitView {
  return {
    visitId: 'VISIT-001',
    patientRef: 'PATIENT-DEMO-001',
    status: 'ACTIVE',
    steps: [
      { sequence: 1, serviceCode: 'REGISTRATION', status: 'COMPLETED' },
      { sequence: 2, serviceCode: 'SCREENING', status: 'COMPLETED' },
      { sequence: 3, serviceCode: 'DOCTOR', status: 'COMPLETED' },
      { sequence: 4, serviceCode: 'LAB', status: 'READY' },
      { sequence: 5, serviceCode: 'PHARMACY', status: 'PENDING' },
    ],
    next: {
      sequence: 4,
      status: 'READY',
      servicePoint: { id: 'SP-LAB', code: 'LAB', name: 'Laboratory', placeId: 'LAB-01' },
    },
    ...overrides,
  }
}

describe('toJourneySteps', () => {
  it('marks completed steps done, the READY step current, and the step after it next', () => {
    const steps = toJourneySteps(visitView())

    expect(steps.map((s) => s.state)).toEqual([
      'done',
      'done',
      'done',
      'current',
      'next',
    ])
  })

  it('translates service codes to plain-Thai titles', () => {
    const steps = toJourneySteps(visitView())

    expect(steps.map((s) => s.title)).toEqual([
      'ลงทะเบียน',
      'คัดกรอง',
      'พบแพทย์',
      'เจาะเลือด',
      'รับยา',
    ])
  })

  it('carries the service point place into the current step meta', () => {
    const current = toJourneySteps(visitView()).find((s) => s.state === 'current')

    expect(current?.meta).toBe('Laboratory · LAB-01')
  })

  it('falls back to a ready-to-serve meta when the step has no service point', () => {
    const view = visitView({ next: { sequence: 4, status: 'READY' } })
    const current = toJourneySteps(view).find((s) => s.state === 'current')

    expect(current?.meta).toBe('พร้อมให้บริการ')
  })

  it('shows no current or next emphasis when the visit has no actionable step', () => {
    const view = visitView({
      steps: [
        { sequence: 1, serviceCode: 'REGISTRATION', status: 'COMPLETED' },
        { sequence: 2, serviceCode: 'PHARMACY', status: 'COMPLETED' },
      ],
      next: undefined,
    })

    expect(toJourneySteps(view).map((s) => s.state)).toEqual(['done', 'done'])
  })

  it('keeps an unknown HIS service code readable instead of hiding the step', () => {
    const view = visitView({
      steps: [
        { sequence: 1, serviceCode: 'MYSTERY', status: 'READY' },
      ],
      next: { sequence: 1, status: 'READY' },
    })

    const [step] = toJourneySteps(view)
    expect(step.title).toBe('MYSTERY')
    expect(step.state).toBe('current')
  })
})

describe('thaiStepTitle', () => {
  it('falls back to the raw code for unmapped services', () => {
    expect(thaiStepTitle('UNKNOWN-SERVICE')).toBe('UNKNOWN-SERVICE')
  })
})

describe('serviceCodeOf', () => {
  it('resolves the code behind a NextStep sequence', () => {
    expect(serviceCodeOf(visitView(), 4)).toBe('LAB')
  })

  it('returns an empty string for an unknown sequence', () => {
    expect(serviceCodeOf(visitView(), 99)).toBe('')
  })
})

describe('visitLoadErrorMessage', () => {
  it.each([
    [404, 'ไม่พบข้อมูลการมาโรงพยาบาลของคุณ'],
    [502, 'ระบบข้อมูลของโรงพยาบาลไม่พร้อมใช้งาน'],
    [500, 'เชื่อมต่อระบบไม่สำเร็จ'],
    [0, 'เชื่อมต่อระบบไม่สำเร็จ'],
  ])('phrases status %i for the patient', (status, expected) => {
    expect(visitLoadErrorMessage(new ApiError(status, 'internal detail'))).toBe(expected)
  })

  it('handles a missing error object', () => {
    expect(visitLoadErrorMessage(null)).toBe('เชื่อมต่อระบบไม่สำเร็จ')
  })
})
