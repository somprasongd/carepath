import { useNavigate } from '@tanstack/react-router'
import { useLocale, useT } from '@/i18n'
import { Card, ChevronRightIcon } from '@/design-system'
import { floorLabelFor, useFloors } from '@/features/floorplan'
import { useCurrentLocation, useNearbyAmenities } from '@/features/navigation'
import { ACCESSIBLE_ONLY_STORAGE_KEY, readStoredFlag } from '@/preferences'
import { amenityLabel, nearestByKind } from '../amenity'

/**
 * The while-you-wait card (FR-26 / #109): for each amenity kind, the
 * nearest one from where the patient stands, offered only when the queue
 * estimate says there is time to walk there and back. Informational ink
 * like QueueCard — the primary action of the screen stays the next-step
 * route; each row's "go" hands the navigate screen a toPlace destination,
 * the same map, cues and voice it already renders.
 *
 * Rendering this component is itself the wait gate's decision (the journey
 * screen checks amenitySuggestionEligible first), so the location poll and
 * the search only start once the wait is long enough to matter.
 */
export function AmenitySuggestions({
  visitId,
  waitMinutes,
}: {
  visitId: string
  waitMinutes: number
}) {
  const t = useT()
  const { locale } = useLocale()
  const navigate = useNavigate()
  // Rank by the same graph the patient will walk: a wheelchair user's
  // "nearest" restroom must not sit across the stairs (#99's stored pref).
  const accessibleOnly = readStoredFlag(ACCESSIBLE_ONLY_STORAGE_KEY) ?? false
  const { data: location } = useCurrentLocation(visitId)
  const fromNodeId = location?.nodeId ?? null
  const { data: search } = useNearbyAmenities(fromNodeId, accessibleOnly)
  // Cache hit on the journey screen's prefetch; needed only to name the
  // amenity's floor in the patient's language.
  const { data: floors } = useFloors()

  const picks = search ? nearestByKind(search.amenities) : []
  if (picks.length === 0) return null

  return (
    <Card radius="md" padding="md" className="mb-5">
      <div className="mb-1 font-sans text-caption text-ink-muted">{t('amenity.title')}</div>
      <p className="m-0 mb-3 font-sans text-body-sm text-ink-muted">
        {t('amenity.whileWaiting', { minutes: Math.round(waitMinutes) })}
      </p>
      <ul className="m-0 flex list-none flex-col gap-2 p-0">
        {picks.map(({ kind, amenity }) => (
          <li key={amenity.place.id}>
            <button
              type="button"
              onClick={() =>
                navigate({
                  to: '/patient/navigate',
                  search: { visit: visitId, toPlace: amenity.place.id },
                })
              }
              className="flex w-full cursor-pointer items-center justify-between gap-3 rounded-lg border border-line bg-surface px-3.5 py-2.5 text-left transition-colors hover:border-primary"
            >
              <span className="min-w-0 flex-1">
                <span className="block font-sans text-body-md font-bold text-ink">
                  {amenityLabel(kind, locale)}
                </span>
                <span className="block truncate font-sans text-caption text-ink-muted">
                  {floorLabelFor(amenity.place.floorId, locale, floors ?? [])}
                </span>
              </span>
              <span className="flex shrink-0 items-center gap-1 font-sans text-caption font-bold text-primary">
                {t('amenity.go')}
                <ChevronRightIcon />
              </span>
            </button>
          </li>
        ))}
      </ul>
    </Card>
  )
}
