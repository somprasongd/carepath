import type { ApiError } from '@/api/client'
import { clockLabel } from '@/i18n/time'
import { format, messagesFor, type Locale } from '@/i18n'
import type { SharedJourney, SharedStep } from './queries'

/**
 * Pure mapping for the /shared screen (#90) — the relative-facing half of
 * ADR-0011. Everything the screen renders as text is produced here so it is
 * testable without a DOM, following features/visit/journey.ts.
 *
 * The API payload is already redacted server-side (ADR-0011 §3) and carries
 * display strings, not codes — `title`, `servicePointName` and `floorName`
 * are authored server-side in Thai and stay as-is in every locale until the
 * contract learns to carry codes (the ADR-0012 server-side trigger). This
 * layer's job is everything else: the patient language, no domain
 * vocabulary, honest states.
 */

/** Coarse status → what a relative reads. Exhaustive over the contract enum. */
export function sharedStatusLabel(
  status: SharedJourney['status'] | SharedStep['status'],
  locale: Locale,
): string {
  const catalog = messagesFor(locale)
  switch (status) {
    case 'WAITING':
      return catalog['shared.status.waiting']
    case 'IN_SERVICE':
      return catalog['shared.status.inService']
    case 'DONE':
      return catalog['shared.status.done']
    default:
      return catalog['shared.status.awaiting']
  }
}

export function shareExpiryLabel(expiresAt: string, locale: Locale): string {
  return format(messagesFor(locale), 'share.validUntil', { time: clockLabel(expiresAt, locale) })
}

export function sharedUpdatedLabel(updatedAt: string, locale: Locale): string {
  return format(messagesFor(locale), 'shared.updatedAt', { time: clockLabel(updatedAt, locale) })
}

/** Where the patient is, as one quiet line; never a code or an id. */
export function sharedWhereLine(
  step: SharedStep | null | undefined,
  locale: Locale,
): string {
  if (!step?.servicePointName) return messagesFor(locale)['shared.awaitingServicePoint']
  return step.floorName ? `${step.servicePointName} · ${step.floorName}` : step.servicePointName
}

/** The five states the screen can be in (issue #90, the /shared screen). */
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
  locale: Locale
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
    title: step?.title ?? messagesFor(input.locale)['shared.preparingNextStep'],
    where: sharedWhereLine(step, input.locale),
    statusLabel: sharedStatusLabel(step?.status ?? data.status, input.locale),
    updatedLabel: sharedUpdatedLabel(data.updatedAt, input.locale),
    expiryLabel: shareExpiryLabel(data.expiresAt, input.locale),
  }
}
