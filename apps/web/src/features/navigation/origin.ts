// The navigate screen's fallback origin: with no location observation yet,
// the patient most plausibly stands where their latest finished step happened
// (ADR-0004's manual-selection fallback, decided by the screen instead of a
// human). This is a display assumption only — nothing is recorded, and a real
// observation (QR scan, manual pick) always wins once one exists.
import type { Locale } from '@/i18n'
import { format, messagesFor } from '@/i18n'
import type { Journey, JourneyStep } from '@/features/visit'
import { stepTitle } from '@/features/visit'

export type AssumedOrigin = {
  /** The completed step's service point place entry, as a routing start. */
  nodeId: string
  /** Localized title of the step the assumption is anchored on. */
  stepTitle: string
}

function stepEntryNode(step: JourneyStep): string | null {
  return step.servicePoint?.place?.entryNodeId ?? null
}

/**
 * Where to assume the patient is before any location fix lands: the latest
 * COMPLETED step that resolves to a navigation node. Steps without a mapped
 * service point (or a place without an entry node) are skipped — walking back
 * to an earlier anchor beats showing no route at all. Registration completes
 * with visit.opened, so an active visit almost always has one.
 */
export function assumedOrigin(journey: Journey | undefined, locale: Locale): AssumedOrigin | null {
  if (!journey) return null
  for (let i = journey.steps.length - 1; i >= 0; i--) {
    const step = journey.steps[i]
    if (step.status !== 'COMPLETED') continue
    const nodeId = stepEntryNode(step)
    if (!nodeId) continue
    return { nodeId, stepTitle: stepTitle(step, locale) }
  }
  return null
}

/**
 * The panel line for the assumed state — visibly not a location fix
 * (DESIGN.md's honesty rule): it names the anchor step rather than claiming
 * to know where the patient is.
 */
export function assumedOriginLabel(origin: AssumedOrigin, locale: Locale): string {
  return format(messagesFor(locale), 'navigate.assumedLocation', { step: origin.stepTitle })
}
