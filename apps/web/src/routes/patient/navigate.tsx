import { useState } from 'react'
import { Navigate, createFileRoute, useNavigate } from '@tanstack/react-router'
import { useLocale } from '@/i18n'
import { navigatePlanForJourney, floorLabelFor } from '@/features/floorplan'
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
} from '@/features/navigation'
import { useJourney } from '@/features/visit'
import { NavigateScreen } from './-NavigateScreen'
import { ScanOverlay } from './-ScanOverlay'

/** `?visit=<VN>` — same parameter as the journey screen, forwarded by its CTA. */
export const Route = createFileRoute('/patient/navigate')({
  validateSearch: (search: Record<string, unknown>): { visit?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
  }),
  component: PatientNavigateRoute,
})

function PatientNavigateRoute() {
  const navigate = useNavigate()
  const { visit } = Route.useSearch()
  const { locale } = useLocale()
  const floorLabel = (floorId: string) => floorLabelFor(floorId, locale)

  // Reached without a visit (bookmark, stale link): send them to the VN
  // entry screen rather than guessing a journey for them.
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

  // Same query key as the journey screen, so this is a cache read, not a
  // second round trip. The plan resolves the destination's place and floor
  // from the recommended step (ADR-0009) onto the floor-plan asset (#25).
  const { data, isPending } = useJourney(visitId)
  const plan = navigatePlanForJourney(data, isPending, locale)

  // The live route (#29): a location observation is the preferred start; the
  // latest finished step's service point is the assumed fallback (nothing is
  // recorded — a real fix always wins once one lands). Either way a changed
  // origin changes the route query's key, so the line redraws by itself.
  const { data: location } = useCurrentLocation(visitId)
  const assumed = location ? null : assumedOrigin(data, locale)
  const originNodeId = location?.nodeId ?? assumed?.nodeId ?? null
  const servicePointCode = plan.state === 'plan' ? plan.servicePointCode : ''
  const { data: route } = useNavigationRoute(originNodeId, servicePointCode)

  // Cross-floor view (#35 follow-up): show the floor the patient stands on
  // first — the "you are here" mark and the first walking leg — with a
  // switcher to the destination floor. Same-floor routes stay put.
  const routeFloorIds = route ? Array.from(new Set(route.nodes.map((n) => n.floorId))) : []
  const defaultFloorId = routeFloorIds[0] ?? (plan.state === 'plan' ? plan.floorId : null)
  const floorId =
    floorOverride && routeFloorIds.includes(floorOverride) ? floorOverride : defaultFloorId

  const mapOverlay =
    plan.state === 'plan' && route && floorId
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

  return (
    <div className="h-dvh">
      <NavigateScreen
        plan={plan}
        route={mapOverlay}
        onFloorChange={setFloorOverride}
        currentLocation={location ? currentLocationLabel(location, floorLabel, locale) : undefined}
        assumedLocation={assumed ? assumedOriginLabel(assumed, locale) : undefined}
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
