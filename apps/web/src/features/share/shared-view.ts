import type { ApiError } from '@/api/client'
import type { SharedJourney, SharedStep } from './queries'

/**
 * Pure mapping for the /shared screen (#90) — the relative-facing half of
 * ADR-0011. Everything the screen renders as text is produced here so it is
 * testable without a DOM, following features/visit/journey.ts.
 *
 * The API payload is already redacted server-side (ADR-0011 §3); this layer's
 * job is language: plain Thai, no domain vocabulary, honest states.
 */

/** Coarse status → what a relative reads. Exhaustive over the contract enum. */
export function sharedStatusLabel(status: SharedJourney['status'] | SharedStep['status']): string {
  switch (status) {
    case 'WAITING':
      return 'กำลังรอคิว'
    case 'IN_SERVICE':
      return 'กำลังรับบริการ'
    case 'DONE':
      return 'เสร็จเรียบร้อย'
    default:
      return 'รออัปเดต'
  }
}

/** "14:05 น." — the only clock format the share surface uses. */
export function thaiClock(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')} น.`
}

export function shareExpiryLabel(expiresAt: string): string {
  return `ใช้ได้ถึง ${thaiClock(expiresAt)}`
}

export function sharedUpdatedLabel(updatedAt: string): string {
  return `อัปเดตล่าสุด ${thaiClock(updatedAt)}`
}

/** Where the patient is, as one quiet line; never a code or an id. */
export function sharedWhereLine(step: SharedStep | null | undefined): string {
  if (!step?.servicePointName) return 'รอยืนยันจุดบริการ'
  return step.floorName ? `${step.servicePointName} · ${step.floorName}` : step.servicePointName
}

/** The five states the screen can be in (issue #90 §หน้า /shared). */
export type SharedScreenModel =
  | { kind: 'loading' }
  | {
      kind: 'active'
      title: string
      where: string
      statusLabel: string
      updatedLabel: string
      expiryLabel: string
    }
  /** Every step finished — the relative can come pick the patient up. */
  | { kind: 'home' }
  /** Expired or revoked: one message, never a raw 401 (NFR-10). */
  | { kind: 'expired' }
  /** Opened with no token in the URL at all — nothing was requested. */
  | { kind: 'invalid' }
  /** The API could not be reached / answered unexpectedly — not about the link. */
  | { kind: 'unreachable' }

/**
 * Fold the query state into the screen model. Order matters: no token wins
 * before any request could have fired, and a 401 from the API is exactly the
 * expired/revoked case (the server never distinguishes them — ADR-0011 §5).
 */
export function sharedScreenModel(input: {
  hasToken: boolean
  isPending: boolean
  data?: SharedJourney
  error: ApiError | null
}): SharedScreenModel {
  if (!input.hasToken) return { kind: 'invalid' }
  if (input.isPending) return { kind: 'loading' }
  // A 401 is exactly the expired/revoked case — the server never
  // distinguishes them (ADR-0011 §5), and neither does this screen.
  if (input.error) {
    return input.error.status === 401 ? { kind: 'expired' } : { kind: 'unreachable' }
  }
  const data = input.data
  if (!data) return { kind: 'loading' }
  if (data.status === 'DONE') return { kind: 'home' }

  const step = data.currentStep
  return {
    kind: 'active',
    title: step?.title ?? 'ระหว่างเตรียมขั้นตอนถัดไป',
    where: sharedWhereLine(step),
    statusLabel: sharedStatusLabel(step?.status ?? data.status),
    updatedLabel: sharedUpdatedLabel(data.updatedAt),
    expiryLabel: shareExpiryLabel(data.expiresAt),
  }
}
