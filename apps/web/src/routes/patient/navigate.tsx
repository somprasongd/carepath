import { useState } from 'react'
import { Navigate, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useLocale } from '@/i18n'
import {
  amenityDestinationPlan,
  floorLabelFor,
  navigatePlanForJourney,
  planUrlFor,
  useFloorPlan,
  useFloors,
  usePlaces,
  type NavigatePlan,
  type Place,
} from '@/features/floorplan'
import type { FloorPlanRoute } from '@/design-system'
import {
  assumedOrigin,
  assumedOriginLabel,
  currentLocationLabel,
  qrScannerSupported,
  routeOriginOnFloor,
  routePolylinesByFloor,
  turnByTurnSteps,
  useCurrentLocation,
  useNavigationRoute,
  usePlaceRoute,
  useVoiceGuidance,
  voiceSteps,
} from '@/features/navigation'
import { useJourney } from '@/features/visit'
import {
  ACCESSIBLE_ONLY_STORAGE_KEY,
  readStoredFlag,
  storeFlag,
} from '@/preferences'
import { NavigateScreen } from './-NavigateScreen'
import { ScanOverlay } from './-ScanOverlay'

/** `?visit=<VN>` — same parameter as the journey screen, forwarded by its CTA.
 * `?toPlace=<placeId>` overrides the destination (#109): the amenity card
 * hands the screen an amenity place to walk to instead of the next step. */
export const Route = createFileRoute('/patient/navigate')({
  validateSearch: (search: Record<string, unknown>): { visit?: string; toPlace?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
    toPlace:
      typeof search.toPlace === 'string' && search.toPlace !== '' ? search.toPlace : undefined,
  }),
  component: PatientNavigateRoute,
})

/** The toPlace entry point's plan: pending while the place list loads, then
 * the amenity plan for that row — a place id no row answers is honestly
 * no-destination (a stale link, not a crash). */
function placeLookupPlan(
  places: Place[] | undefined,
  placeId: string,
  pending: boolean,
  locale: ReturnType<typeof useLocale>['locale'],
): NavigatePlan {
  if (!places && pending) return { state: 'pending' }
  return amenityDestinationPlan(places?.find((place) => place.id === placeId), locale)
}

function PatientNavigateRoute() {
  const navigate = useNavigate()
  const { visit, toPlace } = Route.useSearch()
  const { locale } = useLocale()
  // The floor list is server data since ADR-0015 — it carries each floor's
  // code (which the label is built from) and the URL of its current plan.
  const { data: floors } = useFloors()
  const floorLabel = (floorId: string) => floorLabelFor(floorId, locale, floors ?? [])

  // Reached without a visit (bookmark, stale link): send them to the
  // journey route's front door (slip-link exchange / demo VN entry) rather
  // than guessing a journey for them.
  if (visit === undefined) {
    return <Navigate to="/patient/journey" replace />
  }
  const visitId = visit

  // The scan overlay's mode when opened: camera where the browser supports
  // it, the place list everywhere else (the desktop demo path).
  const [scan, setScan] = useState<'camera' | 'pick' | null>(null)
  // The patient can flip floors on a cross-floor route; null means "follow
  // the default". An override the current route no longer contains simply
  // falls through to the default — no effect needed to expire it.
  const [floorOverride, setFloorOverride] = useState<string | null>(null)
  // Avoid-stairs routing (#99, FR-20): a device-local preference like the
  // locale (#94) — a wheelchair user should not re-opt-in every visit.
  const [accessibleOnly, setAccessibleOnly] = useState(
    () => readStoredFlag(ACCESSIBLE_ONLY_STORAGE_KEY) ?? false,
  )
  const toggleAccessibleOnly = () => {
    setAccessibleOnly((value) => {
      const next = !value
      storeFlag(ACCESSIBLE_ONLY_STORAGE_KEY, next)
      return next
    })
  }

  // Same query key as the journey screen, so this is a cache read, not a
  // second round trip. The plan resolves the destination's place and floor
  // from the recommended step (ADR-0009) onto the floor-plan asset (#25) —
  // unless a toPlace override (#109) names an amenity destination, in which
  // case the journey read stays for the location fallback and the plan
  // comes from the place row instead.
  const { data, isPending } = useJourney(visitId)
  const { data: places, isPending: placesPending } = usePlaces()
  const plan = toPlace
    ? placeLookupPlan(places, toPlace, placesPending, locale)
    : navigatePlanForJourney(data, isPending, locale)

  // The live route (#29): a location observation is the preferred start; the
  // latest finished step's service point is the assumed fallback (nothing is
  // recorded — a real fix always wins once one lands). Either way a changed
  // origin changes the route query's key, so the line redraws by itself.
  // An amenity destination rides toPlace — the two queries below never both
  // run: each is disabled unless its own destination form is the one in use.
  const { data: location } = useCurrentLocation(visitId)
  const assumed = location ? null : assumedOrigin(data, locale)
  const originNodeId = location?.nodeId ?? assumed?.nodeId ?? null
  const servicePointCode =
    !toPlace && plan.state === 'plan' ? plan.servicePointCode ?? null : null
  const { data: codeRoute } = useNavigationRoute(originNodeId, servicePointCode, accessibleOnly)
  const { data: placeRoute } = usePlaceRoute(originNodeId, toPlace ?? null, accessibleOnly)
  const route = toPlace ? placeRoute : codeRoute

  // Cross-floor view (#35 follow-up): show the floor the patient stands on
  // first — the "you are here" mark and the first walking leg — with a
  // switcher to the destination floor. Same-floor routes stay put.
  const routeFloorIds = route ? Array.from(new Set(route.nodes.map((n) => n.floorId))) : []
  const defaultFloorId = routeFloorIds[0] ?? (plan.state === 'plan' ? plan.floorId : null)
  const floorId =
    floorOverride && routeFloorIds.includes(floorOverride) ? floorOverride : defaultFloorId

  // The drawing for whichever floor is on screen. Its URL carries the plan's
  // own digest, so this is a one-time fetch per drawing and a redrawn floor
  // arrives as a new URL rather than a stale cache hit (ADR-0015).
  const planUrl = floorId ? planUrlFor(floorId, floors ?? []) : undefined
  const planQuery = useFloorPlan(planUrl)

  // floorLabel falls back to the raw floor id (e.g. "I-1302") until floors
  // has loaded — fine for the map card's chip, which the patient reads as a
  // place-holder, but not for the spoken-language turn-by-turn cue below, so
  // the whole overlay (cues included) waits for floors rather than let a
  // domain id slip into patient-facing text (DESIGN.md: patients never see
  // domain vocabulary).
  const mapOverlay =
    plan.state === 'plan' && route && floorId && floors !== undefined
      ? {
          floorId,
          floors: routeFloorIds,
          mapRoute: {
            lines: routePolylinesByFloor(route.nodes)[floorId] ?? [],
            // "You are here" only when the route really starts on the
            // displayed floor — never a mark at a floor the patient isn't
            // on (#35).
            origin: routeOriginOnFloor(route.nodes, floorId) ?? undefined,
          } as FloorPlanRoute,
          cues: turnByTurnSteps(route.nodes, route.segments, plan.name, floorLabel, locale),
        }
      : undefined

  // The spoken variant of the same route (#108, FR-25): catalog text and
  // floor labels only — never the destination's name (a specialty clinic's
  // name is medical data, and speech is loud). Empty until a route exists;
  // the hook re-speaks by itself when a fresh fix redraws the route.
  const voiceCues =
    plan.state === 'plan' && route
      ? voiceSteps(route.nodes, route.segments, floorLabel, locale)
      : []
  const voice = useVoiceGuidance(voiceCues, locale)

  return (
    <div className="h-dvh">
      <NavigateScreen
        plan={plan}
        planSvg={planQuery.data}
        // Distinguish "still coming" from "not coming": the floors list
        // having no plan for this floor is as final as a failed fetch.
        planUnavailable={planQuery.isError || (floors !== undefined && planUrl === undefined)}
        floorLabel={floorLabel}
        route={mapOverlay}
        onFloorChange={setFloorOverride}
        currentLocation={location ? currentLocationLabel(location, floorLabel, locale) : undefined}
        assumedLocation={assumed ? assumedOriginLabel(assumed, locale) : undefined}
        accessibleOnly={accessibleOnly}
        onToggleAccessibleOnly={toggleAccessibleOnly}
        voice={voice}
        onScan={() => setScan(qrScannerSupported() ? 'camera' : 'pick')}
        onPickLocation={() => setScan('pick')}
        onBack={() => navigate({ to: '/patient/journey', search: { visit: visitId } })}
      />
      {scan && (
        <ScanOverlay
          visitId={visitId}
          initialMode={scan}
          /** The floor the map is showing — the pick list opens on it. */
          defaultFloorId={floorId ?? undefined}
          onClose={() => setScan(null)}
          onReported={() => {
            setScan(null)
            // A fresh fix moves the origin: drop any stale floor pick so the
            // map follows the new location's floor and the "you are here"
            // mark lands where the patient just reported standing.
            setFloorOverride(null)
          }}
        />
      )}
    </div>
  )
}
