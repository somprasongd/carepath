import { format, messagesFor, type Locale } from '@/i18n'
import type { AmenityNearby } from '@/features/navigation'

/**
 * The amenity kinds the card offers, in display order (#109 / FR-26): the
 * three things a waiting patient actually goes looking for. The values are
 * Place.type codes — server-stable vocabulary the catalog maps to patient
 * text (ADR-0012); AMENITY is the catch-all for kinds the map has not
 * modeled specifically yet.
 */
export const AMENITY_KIND_ORDER = ['RESTROOM', 'WAITING_AREA', 'FOOD_STALL', 'AMENITY'] as const

export type AmenityKind = (typeof AMENITY_KIND_ORDER)[number]

const amenityKindSet: ReadonlySet<string> = new Set(AMENITY_KIND_ORDER)

/** Whether a Place.type is one of the cataloged amenity kinds. */
export function isAmenityKind(type: string): type is AmenityKind {
  return amenityKindSet.has(type)
}

/**
 * How long the visit is estimated to keep waiting before the suggestions
 * are worth a walk (#109): below this, the patient is better off staying
 * put — a detour that risks missing the call is a wrong suggestion, not a
 * helpful one. A client-owned presentation constant (ADR-0012 spirit): the
 * API ranks, the screen decides when showing is kind.
 */
export const AMENITY_MIN_WAIT_MINUTES = 10

/** The card's gate: an estimate exists and is long enough to walk away from. */
export function amenitySuggestionEligible(waitMinutes: number | null): boolean {
  return waitMinutes !== null && waitMinutes >= AMENITY_MIN_WAIT_MINUTES
}

/**
 * One suggestion per kind, in the fixed display order: the nearest of each
 * (the API list is already nearest-first, so the first seen per kind wins)
 * and kinds with no reachable place simply do not appear — an honest
 * shorter card beats a padded one. The kind comes back narrowed so callers
 * can label it without re-deriving the vocabulary.
 */
export function nearestByKind(amenities: AmenityNearby[]): Array<{
  kind: AmenityKind
  amenity: AmenityNearby
}> {
  const firstByKind = new Map<string, AmenityNearby>()
  for (const amenity of amenities) {
    if (!firstByKind.has(amenity.place.type)) {
      firstByKind.set(amenity.place.type, amenity)
    }
  }
  return AMENITY_KIND_ORDER.flatMap((kind) => {
    const nearest = firstByKind.get(kind)
    return nearest ? [{ kind, amenity: nearest }] : []
  })
}

/** The patient-facing label for an amenity kind — catalog by code (ADR-0012). */
export function amenityLabel(kind: AmenityKind, locale: Locale): string {
  return format(messagesFor(locale), `amenity.kind.${kind}`)
}
