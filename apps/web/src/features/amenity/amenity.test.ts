import { describe, expect, it } from 'vitest'
import { format, messagesFor } from '@/i18n'
import type { AmenityNearby } from '@/features/navigation'
import {
  AMENITY_MIN_WAIT_MINUTES,
  amenityLabel,
  amenitySuggestionEligible,
  isAmenityKind,
  nearestByKind,
} from './amenity'

function amenity(type: AmenityNearby['place']['type'], id = `${type}-01`): AmenityNearby {
  return {
    // The non-kind fields are irrelevant to grouping; only type and identity
    // reach the assertions.
    place: {
      id,
      floorId: 'I-1301',
      name: id,
      type,
      floor: { id: 'I-1301', buildingId: 'BLD-I13', code: '1', name: 'Ground Floor', levelOrder: 1 },
    },
    distance: 100,
  }
}

describe('amenitySuggestionEligible', () => {
  it('shows only when an estimate exists and is long enough to walk away from', () => {
    expect(amenitySuggestionEligible(null)).toBe(false)
    expect(amenitySuggestionEligible(0)).toBe(false)
    expect(amenitySuggestionEligible(AMENITY_MIN_WAIT_MINUTES - 1)).toBe(false)
    expect(amenitySuggestionEligible(AMENITY_MIN_WAIT_MINUTES)).toBe(true)
    expect(amenitySuggestionEligible(45)).toBe(true)
  })
})

describe('nearestByKind', () => {
  it('picks the nearest of each kind from the nearest-first list, in display order', () => {
    // Arrival order is the ranking: RESTROOM-NEAR (350) precedes
    // RESTROOM-FAR (700), so it is the one the card shows.
    const picks = nearestByKind([
      amenity('RESTROOM', 'RESTROOM-NEAR'),
      amenity('FOOD_STALL', 'FOOD-01'),
      amenity('RESTROOM', 'RESTROOM-FAR'),
      amenity('WAITING_AREA', 'WAITING-01'),
    ])

    expect(picks.map((pick) => pick.kind)).toEqual(['RESTROOM', 'WAITING_AREA', 'FOOD_STALL'])
    expect(picks[0]?.amenity.place.id).toBe('RESTROOM-NEAR')
  })

  it('drops kinds with no reachable place instead of padding the card', () => {
    expect(nearestByKind([amenity('RESTROOM')]).map((pick) => pick.kind)).toEqual(['RESTROOM'])
    expect(nearestByKind([])).toEqual([])
    // Unmapped kinds do not appear — the catalog has no honest label for them.
    expect(nearestByKind([amenity('ROOM')])).toEqual([])
  })
})

describe('amenityLabel', () => {
  it('labels every cataloged kind in both locales (ADR-0012: catalog by code)', () => {
    for (const locale of ['th', 'en'] as const) {
      for (const kind of ['RESTROOM', 'WAITING_AREA', 'FOOD_STALL', 'AMENITY'] as const) {
        expect(amenityLabel(kind, locale)).toBe(
          format(messagesFor(locale), `amenity.kind.${kind}`),
        )
      }
    }
    expect(amenityLabel('RESTROOM', 'en')).toBe('Restroom')
  })

  it('narrows Place.type via isAmenityKind', () => {
    expect(isAmenityKind('RESTROOM')).toBe(true)
    expect(isAmenityKind('ROOM')).toBe(false)
  })
})
