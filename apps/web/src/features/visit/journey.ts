import type { JourneyStep as RailStep, JourneyStepState } from '@/design-system'
import type { ApiError } from '@/api/client'
import type { Journey, JourneyStep } from './queries'

/**
 * Clinic codes → plain-Thai clinic names (DESIGN.md: never surface domain
 * vocabulary like a raw clinic code). An unmapped code falls back to itself.
 */
const CLINIC_NAMES: Record<string, string> = {
  MED: 'อายุรกรรม',
  SURG: 'ศัลยกรรม',
}

/** Step kind → plain-Thai title for every kind except CLINIC (handled below). */
const KIND_TITLES: Record<string, string> = {
  REGISTRATION: 'ลงทะเบียน',
  LAB: 'เจาะเลือด',
  XRAY: 'เอกซเรย์',
  EKG: 'ตรวจคลื่นไฟฟ้าหัวใจ',
  ULTRASOUND: 'อัลตราซาวด์',
  CASHIER: 'ชำระเงิน',
  PHARMACY: 'รับยา',
}

/**
 * Plain-Thai title for a journey step (ADR-0009 — steps are addressed by
 * kind/clinicCode now, not an HIS serviceCode). A round above 1 means the
 * patient is returning to a doctor they already saw this visit. `kind` is
 * typed as a plain string, not the schema enum, so an unmapped kind still
 * renders instead of being a type error — the fallback is the point.
 */
export function thaiStepTitle(
  step: Pick<JourneyStep, 'clinicCode' | 'round'> & { kind: string },
): string {
  if (step.kind === 'CLINIC') {
    const clinic = step.clinicCode ? (CLINIC_NAMES[step.clinicCode] ?? step.clinicCode) : ''
    const suffix = clinic ? ` · ${clinic}` : ''
    return step.round && step.round > 1 ? `กลับไปพบแพทย์${suffix}` : `พบแพทย์${suffix}`
  }
  return KIND_TITLES[step.kind] ?? step.kind
}

/**
 * Map a Journey onto the patient's journey rail (ADR-0009). More than one
 * step can be actionable at once; the recommended one (or any STARTED step)
 * gets the rail's `current` emphasis, every other actionable step is `next`,
 * everything else stays `pending`.
 */
export function toJourneySteps(journey: Journey): RailStep[] {
  const actionableKeys = new Set(journey.actionable.map((s) => s.stepKey))
  const recommendedKey = journey.recommended?.stepKey
  const startedKey = journey.steps.find((s) => s.status === 'STARTED')?.stepKey

  return journey.steps.map((step) => {
    let state: JourneyStepState
    if (step.status === 'COMPLETED' || step.status === 'CANCELLED') {
      state = 'done'
    } else if (step.stepKey === startedKey || (!startedKey && step.stepKey === recommendedKey)) {
      state = 'current'
    } else if (actionableKeys.has(step.stepKey)) {
      state = 'next'
    } else {
      state = 'pending'
    }

    return {
      id: step.stepKey,
      state,
      title: thaiStepTitle(step),
      meta: stepMeta(step, state),
    }
  })
}

function stepMeta(step: JourneyStep, state: JourneyStepState): string {
  if (step.status === 'CANCELLED') return 'ยกเลิกแล้ว'
  if (step.status === 'COMPLETED') return 'เสร็จสิ้นแล้ว'
  if (step.status === 'WAITING') return 'รอผลตรวจ'
  if (state === 'current' || state === 'next') {
    return step.servicePoint ? `${step.servicePoint.name} · ${step.servicePoint.placeId}` : 'พร้อมให้บริการ'
  }
  return 'รอดำเนินการ'
}

/**
 * Load-failure phrasing for the patient, keyed off the HTTP status — the raw
 * error detail (English, internal) stays in the console, never on screen.
 */
export function visitLoadErrorMessage(error: ApiError | null): string {
  if (error?.status === 404) return 'ไม่พบข้อมูลการมาโรงพยาบาลของคุณ'
  if (error?.status === 502) return 'ระบบข้อมูลของโรงพยาบาลไม่พร้อมใช้งาน'
  return 'เชื่อมต่อระบบไม่สำเร็จ'
}
