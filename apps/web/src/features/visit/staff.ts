import type { Locale } from '@/i18n'
import { clockLabel } from '@/i18n/time'
import { stepTitle } from './journey'
import type { Journey } from './queries'

/**
 * Staff-facing labels and tones for the visit monitor (#37). Staff screens
 * may show operational vocabulary — DESIGN.md's plain-Thai rule is a patient
 * rule — so the raw status stays visible where staff needs it. The staff
 * console is out of FR-19's scope and pins the locale to Thai (ADR-0012 §2).
 */
export type StaffTone = 'routable' | 'busy' | 'ready' | 'quiet'

const STEP_TONES: Record<string, StaffTone> = {
  COMPLETED: 'routable',
  STARTED: 'busy',
  READY: 'ready',
  WAITING: 'quiet',
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
  WAITING: 'รอผลตรวจ',
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
 * A started step wins over the recommended one; a finished visit is
 * explicit. More than one step may be actionable at once (ADR-0009 §6) —
 * this shows CarePath's single recommendation, not the full set.
 */
export function visitPosition(journey: Journey): string {
  if (journey.completed) return 'เสร็จสิ้นทุกขั้นตอน'
  if (journey.status === 'CANCELLED') return 'การมารับบริการถูกยกเลิก'
  const started = journey.steps.find((step) => step.status === 'STARTED')
  if (started) return `กำลัง${stepTitle(started, 'th')}`
  if (journey.recommended) return `ถัดไป · ${stepTitle(journey.recommended, 'th')}`
  return 'ไม่มีขั้นตอนที่ดำเนินการได้'
}

/** "2/5" — completed steps over all steps of the visit. */
export function stepProgress(journey: Journey): string {
  const done = journey.steps.filter((step) => step.status === 'COMPLETED').length
  return `${done}/${journey.steps.length}`
}

/**
 * Clock time of the last projection sync, e.g. "14:05 น." (#94). Locale-
 * taking like every other mapper, but the staff console pins 'th'
 * (ADR-0012 §2) — its callers pass the pin, not the patient toggle's
 * state. Replaces this file's hardcoded `toLocaleTimeString('th-TH', …)`,
 * which was the app's one straggler.
 */
export function syncedAtLabel(syncedAt: string, locale: Locale): string {
  return clockLabel(syncedAt, locale)
}

/** What the staff controls may do to a step (#38) — null when nothing is legal. */
export type StepAction = { to: 'STARTED' | 'COMPLETED'; label: string }

export function stepAction(status: string): StepAction | null {
  if (status === 'READY') return { to: 'STARTED', label: 'เริ่มขั้นตอน' }
  if (status === 'STARTED') return { to: 'COMPLETED', label: 'ทำเสร็จแล้ว' }
  return null
}

/**
 * Thai phrasing for a failed transition. The server detail (already a plain
 * message) rides along; 409 is the HIS rejecting an illegal move — the
 * common case when two staff race on the same step.
 */
export function transitionErrorText(status: number, detail: string): string {
  if (status === 0) return `เปลี่ยนสถานะไม่สำเร็จ — ${detail}`
  if (status === 409) return `ตอนนี้เปลี่ยนสถานะนี้ไม่ได้ (${detail}) — รีเฟรชแล้วลองใหม่`
  if (status === 400) return `คำสั่งไม่ถูกต้อง (${detail})`
  return `เปลี่ยนสถานะไม่สำเร็จ (สถานะ ${status}${detail ? ` · ${detail}` : ''})`
}
