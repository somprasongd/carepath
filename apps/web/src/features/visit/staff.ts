import { thaiStepTitle } from './journey'
import type { Journey } from './queries'

/**
 * Staff-facing labels and tones for the visit monitor (#37). Staff screens
 * may show operational vocabulary — DESIGN.md's plain-Thai rule is a patient
 * rule — so the raw status stays visible where staff needs it.
 */
export type StaffTone = 'routable' | 'busy' | 'ready' | 'quiet'

const STEP_TONES: Record<string, StaffTone> = {
  COMPLETED: 'routable',
  STARTED: 'busy',
  READY: 'ready',
  PENDING: 'quiet',
  CANCELLED: 'quiet',
}

const VISIT_TONES: Record<string, StaffTone> = {
  ACTIVE: 'busy',
  COMPLETED: 'routable',
  CANCELLED: 'quiet',
}

export function stepTone(status: string): StaffTone {
  return STEP_TONES[status] ?? 'quiet'
}

export function visitTone(status: string): StaffTone {
  return VISIT_TONES[status] ?? 'quiet'
}

const STEP_LABELS: Record<string, string> = {
  COMPLETED: 'เสร็จสิ้น',
  STARTED: 'กำลังให้บริการ',
  READY: 'พร้อมให้บริการ',
  PENDING: 'รอคิว',
  CANCELLED: 'ยกเลิก',
}

const VISIT_LABELS: Record<string, string> = {
  ACTIVE: 'กำลังรับบริการ',
  COMPLETED: 'เสร็จสิ้น',
  CANCELLED: 'ยกเลิก',
}

/** Unknown statuses fall back to the raw code — staff can read the enum. */
export function stepStatusLabel(status: string): string {
  return STEP_LABELS[status] ?? status
}

export function visitStatusLabel(status: string): string {
  return VISIT_LABELS[status] ?? status
}

/**
 * Where this patient is right now — the one-line summary of a list row.
 * A started step wins over a ready one; a finished visit is explicit.
 */
export function visitPosition(journey: Journey): string {
  if (journey.completed) return 'เสร็จสิ้นทุกขั้นตอน'
  if (journey.status === 'CANCELLED') return 'การมารับบริการถูกยกเลิก'
  if (journey.current) return `กำลัง${thaiStepTitle(journey.current.serviceCode)}`
  if (journey.next) return `ถัดไป · ${thaiStepTitle(journey.next.serviceCode)}`
  return 'ไม่มีขั้นตอนที่ดำเนินการได้'
}

/** "2/5" — completed steps over all steps of the visit. */
export function stepProgress(journey: Journey): string {
  const done = journey.steps.filter((step) => step.status === 'COMPLETED').length
  return `${done}/${journey.steps.length}`
}

/** Clock time of the last projection sync, e.g. "14:05". */
export function syncedAtLabel(syncedAt: string): string {
  return new Date(syncedAt).toLocaleTimeString('th-TH', {
    hour: '2-digit',
    minute: '2-digit',
  })
}
