import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import type { Journey, JourneyStep } from './queries'
import { thaiStepTitle, toJourneySteps, visitLoadErrorMessage } from './journey'

function step(overrides: Partial<JourneyStep>): JourneyStep {
  return {
    stepKey: 'STEP', sequence: 1, kind: 'REGISTRATION', status: 'PENDING',
    clinicCode: null, round: null, orderRefs: [], servicePointId: null,
    ...overrides,
  }
}

/** Mirrors the ADR-0009 shape Mock HIS/CarePath serve: registration done,
 * lab actionable, clinic pending behind it. */
function journey(overrides: Partial<Journey> = {}): Journey {
  const steps: JourneyStep[] = overrides.steps ?? [
    step({ stepKey: 'REGISTRATION', sequence: 1, kind: 'REGISTRATION', status: 'COMPLETED' }),
    step({
      stepKey: 'LAB:1', sequence: 2, kind: 'LAB', status: 'READY', orderRefs: ['ORD-1'],
      servicePointId: 'SP-LAB',
      servicePoint: { id: 'SP-LAB', code: 'ORDERTYPE:LAB', name: 'Laboratory', placeId: 'LAB-01' },
    }),
    step({ stepKey: 'CLINIC:MED:1', sequence: 3, kind: 'CLINIC', clinicCode: 'MED', round: 1, status: 'PENDING' }),
  ]
  const actionable = steps.filter((s) => s.status === 'READY')
  return {
    visitId: 'VISIT-001',
    patientRef: 'PATIENT-DEMO-001',
    status: 'ACTIVE',
    completed: false,
    steps,
    actionable,
    recommended: actionable[0] ?? null,
    syncedAt: '2026-09-19T02:00:00Z',
    ...overrides,
  }
}

describe('toJourneySteps', () => {
  it('marks completed steps done, the recommended READY step current, and other actionable steps next', () => {
    const j = journey()
    j.steps.push(
      step({
        stepKey: 'XRAY:1', sequence: 4, kind: 'XRAY', status: 'READY',
      }),
    )
    j.actionable = j.steps.filter((s) => s.status === 'READY')
    j.recommended = j.actionable[0]

    const rail = toJourneySteps(j)
    expect(rail.map((s) => s.state)).toEqual(['done', 'current', 'pending', 'next'])
  })

  it('marks a STARTED step current even when it is not the recommended one', () => {
    const j = journey({
      steps: [
        step({ stepKey: 'REGISTRATION', sequence: 1, kind: 'REGISTRATION', status: 'COMPLETED' }),
        step({ stepKey: 'CLINIC:MED:1', sequence: 2, kind: 'CLINIC', clinicCode: 'MED', round: 1, status: 'STARTED' }),
      ],
      actionable: [],
      recommended: null,
    })
    expect(toJourneySteps(j).map((s) => s.state)).toEqual(['done', 'current'])
  })

  it('translates step kinds to plain-Thai titles', () => {
    const titles = toJourneySteps(journey()).map((s) => s.title)
    expect(titles).toEqual(['ลงทะเบียน', 'เจาะเลือด', 'พบแพทย์ · อายุรกรรม'])
  })

  it('carries the service point place into the actionable step meta', () => {
    const current = toJourneySteps(journey()).find((s) => s.state === 'current')
    expect(current?.meta).toBe('Laboratory · LAB-01')
  })

  it('falls back to a ready-to-serve meta when the step has no service point', () => {
    const j = journey({
      steps: [step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' })],
      actionable: [step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' })],
      recommended: step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' }),
    })
    expect(toJourneySteps(j)[0].meta).toBe('พร้อมให้บริการ')
  })

  it('shows a waiting meta for a step gated on a pending result', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'WAITING' })],
      actionable: [],
      recommended: null,
    })
    expect(toJourneySteps(j)[0].meta).toBe('รอผลตรวจ')
  })

  it('shows no current or next emphasis when the visit has no actionable step', () => {
    const j = journey({
      steps: [
        step({ stepKey: 'REGISTRATION', kind: 'REGISTRATION', status: 'COMPLETED' }),
        step({ stepKey: 'CASHIER', kind: 'CASHIER', status: 'PENDING' }),
      ],
      actionable: [],
      recommended: null,
    })
    expect(toJourneySteps(j).map((s) => s.state)).toEqual(['done', 'pending'])
  })

  it('labels a return-to-clinic round with the round-2 phrasing', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' })],
      actionable: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' })],
      recommended: step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' }),
    })
    expect(toJourneySteps(j)[0].title).toBe('กลับไปพบแพทย์ · อายุรกรรม')
  })

  it('keeps an unknown clinic code readable instead of hiding the step', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' })],
      actionable: [step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' })],
      recommended: step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' }),
    })
    expect(toJourneySteps(j)[0].title).toBe('พบแพทย์ · PED')
  })
})

describe('thaiStepTitle', () => {
  it('falls back to the raw kind for an unmapped kind', () => {
    // A plain object, not the `step()` factory: `kind` here is intentionally
    // outside the schema's StepKind union, exercising the fallback for a
    // kind the client's type does not yet know about.
    expect(thaiStepTitle({ kind: 'UNKNOWN', clinicCode: null, round: null })).toBe('UNKNOWN')
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
