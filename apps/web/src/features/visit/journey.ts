import type { JourneyStep, JourneyStepState } from '@/design-system'
import type { ApiError } from '@/api/client'
import type { VisitView } from './queries'

/**
 * HIS service codes → the plain-Thai step names patients see (DESIGN.md:
 * never surface domain vocabulary like `ServicePoint` or status enums).
 * An unmapped code falls back to itself so an unknown HIS service still
 * renders — it just reads like a code.
 */
const THAI_STEP_TITLES: Record<string, string> = {
  REGISTRATION: 'ลงทะเบียน',
  SCREENING: 'คัดกรอง',
  DOCTOR: 'พบแพทย์',
  LAB: 'เจาะเลือด',
  PHARMACY: 'รับยา',
  CASHIER: 'ชำระเงิน',
}

export function thaiStepTitle(serviceCode: string): string {
  return THAI_STEP_TITLES[serviceCode] ?? serviceCode
}

/**
 * Map a VisitView onto the patient's journey rail. The first READY step is
 * where the patient is now (`current` — the API's `next` is exactly this
 * step); the step right after it gets the rail's `next` emphasis, everything
 * later stays `pending`. Queue numbers wait for a queue API and are simply
 * absent.
 */
export function toJourneySteps(view: VisitView): JourneyStep[] {
  const next = view.next
  const currentIdx = next ? view.steps.findIndex((step) => step.sequence === next.sequence) : -1

  return view.steps.map((step, idx) => {
    let state: JourneyStepState
    if (step.status === 'COMPLETED') {
      state = 'done'
    } else if (idx === currentIdx) {
      state = 'current'
    } else if (idx === currentIdx + 1) {
      state = 'next'
    } else {
      state = 'pending'
    }

    return {
      id: `${step.sequence}`,
      state,
      title: thaiStepTitle(step.serviceCode),
      meta: stepMeta(step.status === 'COMPLETED', state, view),
    }
  })
}

function stepMeta(done: boolean, state: JourneyStepState, view: VisitView): string {
  if (done) return 'เสร็จสิ้นแล้ว'
  if (state === 'current') {
    const servicePoint = view.next?.servicePoint
    return servicePoint ? `${servicePoint.name} · ${servicePoint.placeId}` : 'พร้อมให้บริการ'
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

/** NextStep carries no serviceCode — look it up in the visit's own steps. */
export function serviceCodeOf(visit: VisitView, sequence: number): string {
  return visit.steps.find((step) => step.sequence === sequence)?.serviceCode ?? ''
}
