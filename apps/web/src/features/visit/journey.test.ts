import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import { messagesFor } from '@/i18n'
import type { Journey, JourneyStep } from './queries'
import {
  journeyProgress,
  journeyProgressLabel,
  recommendationReasonLabel,
  servicePointLabel,
  stepTitle,
  toJourneySteps,
  visitLoadErrorMessage,
  visitOutcome,
} from './journey'

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

    const rail = toJourneySteps(j, 'th')
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
    expect(toJourneySteps(j, 'th').map((s) => s.state)).toEqual(['done', 'current'])
  })

  it('translates step kinds to plain-Thai titles', () => {
    const titles = toJourneySteps(journey(), 'th').map((s) => s.title)
    expect(titles).toEqual(['ลงทะเบียน', 'เจาะเลือด', 'พบแพทย์ · อายุรกรรม'])
  })

  it('carries the service point place into the actionable step meta', () => {
    const current = toJourneySteps(journey(), 'th').find((s) => s.state === 'current')
    expect(current?.meta).toBe('ห้องเจาะเลือด · LAB-01')
  })

  it('localizes the rail for English too (ADR-0012)', () => {
    const titles = toJourneySteps(journey(), 'en').map((s) => s.title)
    expect(titles).toEqual(['Check-in', 'Blood draw', 'See the doctor · Internal Medicine'])
    const current = toJourneySteps(journey(), 'en').find((s) => s.state === 'current')
    expect(current?.meta).toBe('Laboratory · LAB-01')
  })

  it('falls back to a ready-to-serve meta when the step has no service point', () => {
    const j = journey({
      steps: [step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' })],
      actionable: [step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' })],
      recommended: step({ stepKey: 'LAB:1', kind: 'LAB', status: 'READY' }),
    })
    expect(toJourneySteps(j, 'th')[0].meta).toBe('พร้อมให้บริการ')
  })

  it('shows a waiting meta for a step gated on a pending result', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'WAITING' })],
      actionable: [],
      recommended: null,
    })
    expect(toJourneySteps(j, 'th')[0].meta).toBe('รอผลตรวจ')
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
    expect(toJourneySteps(j, 'th').map((s) => s.state)).toEqual(['done', 'pending'])
  })

  it('labels a return-to-clinic round with the round-2 phrasing', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' })],
      actionable: [step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' })],
      recommended: step({ stepKey: 'CLINIC:MED:2', kind: 'CLINIC', clinicCode: 'MED', round: 2, status: 'READY' }),
    })
    expect(toJourneySteps(j, 'th')[0].title).toBe('กลับไปพบแพทย์ · อายุรกรรม')
  })

  it('keeps an unknown clinic code readable instead of hiding the step', () => {
    const j = journey({
      steps: [step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' })],
      actionable: [step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' })],
      recommended: step({ stepKey: 'CLINIC:PED:1', kind: 'CLINIC', clinicCode: 'PED', round: 1, status: 'READY' }),
    })
    expect(toJourneySteps(j, 'th')[0].title).toBe('พบแพทย์ · PED')
  })
})

describe('stepTitle', () => {
  it('falls back to the raw kind for an unmapped kind, in either locale', () => {
    // A plain object, not the `step()` factory: `kind` here is intentionally
    // outside the schema's StepKind union, exercising the fallback for a
    // kind the client's type does not yet know about.
    expect(stepTitle({ kind: 'UNKNOWN', clinicCode: null, round: null }, 'th')).toBe('UNKNOWN')
    expect(stepTitle({ kind: 'UNKNOWN', clinicCode: null, round: null }, 'en')).toBe('UNKNOWN')
  })

  it('translates a plain kind into English', () => {
    expect(stepTitle({ kind: 'PHARMACY', clinicCode: null, round: null }, 'en')).toBe('Medication pickup')
  })

  it('keeps the Thai phrasing unchanged for the Thai locale', () => {
    expect(stepTitle({ kind: 'CLINIC', clinicCode: 'MED', round: 1 }, 'th')).toBe('พบแพทย์ · อายุรกรรม')
    expect(stepTitle({ kind: 'CLINIC', clinicCode: 'MED', round: 2 }, 'th')).toBe('กลับไปพบแพทย์ · อายุรกรรม')
  })

  it('phrases the clinic rounds for English', () => {
    expect(stepTitle({ kind: 'CLINIC', clinicCode: 'MED', round: 1 }, 'en')).toBe('See the doctor · Internal Medicine')
    expect(stepTitle({ kind: 'CLINIC', clinicCode: 'MED', round: 2 }, 'en')).toBe('Return to the doctor · Internal Medicine')
    expect(stepTitle({ kind: 'CLINIC', clinicCode: 'PED', round: 1 }, 'en')).toBe('See the doctor · PED')
  })
})

describe('servicePointLabel', () => {
  const lab = { code: 'ORDERTYPE:LAB', name: 'Laboratory', placeId: 'LAB-01' }

  it('maps the code in both locales, colon codes included', () => {
    expect(servicePointLabel(lab, 'th')).toBe('ห้องเจาะเลือด')
    expect(servicePointLabel(lab, 'en')).toBe('Laboratory')
  })

  it('falls back to the server name for an unmapped code instead of breaking', () => {
    const nursery = { code: 'NURSERY', name: 'Nurse Station', placeId: 'NS-01' }
    expect(servicePointLabel(nursery, 'th')).toBe('Nurse Station')
    expect(servicePointLabel(nursery, 'en')).toBe('Nurse Station')
  })
})

describe('journeyProgress', () => {
  it('counts completed steps over the whole plan', () => {
    // Factory shape: registration done, lab READY, clinic pending.
    expect(journeyProgress(journey())).toEqual({ done: 1, total: 3 })
  })

  it('keeps cancelled steps in the total — the rail still recaps them', () => {
    const j = journey({
      steps: [
        step({ stepKey: 'REGISTRATION', kind: 'REGISTRATION', status: 'COMPLETED' }),
        step({ stepKey: 'XRAY:1', kind: 'XRAY', status: 'CANCELLED' }),
        step({ stepKey: 'CASHIER', kind: 'CASHIER', status: 'PENDING' }),
      ],
      actionable: [],
      recommended: null,
    })
    expect(journeyProgress(j)).toEqual({ done: 1, total: 3 })
  })

  it('is full when every step is completed', () => {
    const j = journey({
      steps: [step({ stepKey: 'REGISTRATION', kind: 'REGISTRATION', status: 'COMPLETED' })],
      actionable: [],
      recommended: null,
      completed: true,
      status: 'COMPLETED',
    })
    expect(journeyProgress(j)).toEqual({ done: 1, total: 1 })
  })
})

describe('journeyProgressLabel', () => {
  it('composes the progress line in each locale', () => {
    expect(journeyProgressLabel(journey(), 'th')).toBe('ความคืบหน้า · เสร็จแล้ว 1 จาก 3 ขั้นตอน')
    expect(journeyProgressLabel(journey(), 'en')).toBe('Progress · 1 of 3 steps done')
  })
})

describe('visitOutcome', () => {
  it('maps the contract finished signal to completed', () => {
    expect(
      visitOutcome(journey({ completed: true, status: 'COMPLETED', actionable: [], recommended: null })),
    ).toBe('completed')
  })

  it('maps a cancelled visit status to cancelled even with steps unfinished', () => {
    expect(visitOutcome(journey({ status: 'CANCELLED' }))).toBe('cancelled')
  })

  it('prefers cancelled if both signals are ever set', () => {
    expect(visitOutcome(journey({ status: 'CANCELLED', completed: true }))).toBe('cancelled')
  })

  it('returns null while the visit is still walking', () => {
    expect(visitOutcome(journey())).toBeNull()
  })
})

describe('visitLoadErrorMessage', () => {
  it.each([
    [404, 'We could not find a visit with that number.'],
    [502, 'The hospital record system is unavailable right now.'],
    [500, 'We could not reach the service.'],
    [0, 'We could not reach the service.'],
  ])('phrases status %i for the patient in English', (status, expected) => {
    expect(visitLoadErrorMessage(new ApiError(status, 'internal detail'), 'en')).toBe(expected)
  })

  it('maps the same statuses onto the Thai catalog (the default locale)', () => {
    expect(visitLoadErrorMessage(new ApiError(404, 'x'), 'th')).toBe(
      messagesFor('th')['journey.error.notFound'],
    )
    expect(visitLoadErrorMessage(new ApiError(502, 'x'), 'th')).toBe(
      messagesFor('th')['journey.error.hisUnavailable'],
    )
    expect(visitLoadErrorMessage(null, 'th')).toBe(messagesFor('th')['journey.error.unreachable'])
  })

  it('handles a missing error object', () => {
    expect(visitLoadErrorMessage(null, 'en')).toBe('We could not reach the service.')
  })
})

describe('recommendationReasonLabel', () => {
  it('maps each criterion code to the locale catalog text', () => {
    expect(recommendationReasonLabel('NEAREST', 'th')).toBe(
      messagesFor('th')['journey.recommendReason.nearest'],
    )
    expect(recommendationReasonLabel('PLAN_ORDER', 'en')).toBe(
      messagesFor('en')['journey.recommendReason.planOrder'],
    )
  })

  it('renders nothing without a recommendation or for an unknown code', () => {
    expect(recommendationReasonLabel(null, 'th')).toBeUndefined()
    expect(recommendationReasonLabel(undefined, 'en')).toBeUndefined()
  })
})
