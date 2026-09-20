import type { ServicePoint } from '@/features/servicepoint'

/** One pickable place in the manual location list (the MANUAL source). */
export interface PickablePlace {
  entryNodeId: string
  name: string
  floorCode: string
}

/**
 * Collapse service points into the places a patient can say they stand at:
 * one entry node per place — several clinics share one nurse station, and the
 * report is about where the patient stands, not which service they await.
 * Service points without a resolved place or entry node are not pickable.
 */
export function pickablePlaces(servicePoints: ServicePoint[] | undefined): PickablePlace[] {
  const places = new Map<string, PickablePlace>()
  for (const sp of servicePoints ?? []) {
    const entryNodeId = sp.place?.entryNodeId
    if (!entryNodeId || !sp.place) continue
    if (!places.has(entryNodeId)) {
      places.set(entryNodeId, {
        entryNodeId,
        name: sp.place.name,
        floorCode: sp.place.floor.code,
      })
    }
  }
  return Array.from(places.values())
}

/**
 * Resolve which floor's places the list shows. Only the patient's explicit
 * chip pick is state; the default is recomputed on every call — the
 * service-points query can resolve after the list mounts, and a captured
 * initial value would latch to '' and keep the rendered list empty even while
 * the chips above it show the freshly loaded floors.
 */
export function activeFloorFor(
  pick: string | null,
  defaultFloorCode: string | undefined,
  floorCodes: string[],
): string {
  if (pick && floorCodes.includes(pick)) return pick
  if (defaultFloorCode && floorCodes.includes(defaultFloorCode)) return defaultFloorCode
  return floorCodes[0] ?? ''
}
