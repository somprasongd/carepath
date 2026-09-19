import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { navigatePlanForJourney, floorLabelFor } from '@/features/floorplan'
import type { FloorPlanRoute } from '@/design-system'
import {
  routePolylinesByFloor,
  turnByTurnSteps,
  useCurrentLocation,
  useNavigationRoute,
} from '@/features/navigation'
import { DEFAULT_VISIT_ID, useJourney } from '@/features/visit'
import { NavigateScreen } from './-NavigateScreen'

/** `?visit=<id>` — same parameter as the journey screen, forwarded by its CTA. */
export const Route = createFileRoute('/patient/navigate')({
  validateSearch: (search: Record<string, unknown>): { visit?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
  }),
  component: PatientNavigateRoute,
})

function PatientNavigateRoute() {
  const navigate = useNavigate()
  const { visit } = Route.useSearch()
  const visitId = visit ?? DEFAULT_VISIT_ID

  // Same query key as the journey screen, so this is a cache read, not a
  // second round trip. The plan resolves the destination's place and floor
  // from the recommended step (ADR-0009) onto the floor-plan asset (#25).
  const { data, isPending } = useJourney(visitId)
  const plan = navigatePlanForJourney(data, isPending)

  // The live route (#29): the latest location observation is the start, the
  // recommended service point the destination. No observation yet (null)
  // keeps the query disabled and the screen on its destination-only view;
  // a new fix changes the query key, so the line redraws by itself (AC2).
  const { data: location } = useCurrentLocation(visitId)
  const servicePointCode = plan.state === 'plan' ? plan.servicePointCode : ''
  const { data: route } = useNavigationRoute(location?.nodeId ?? null, servicePointCode)

  const overlay =
    plan.state === 'plan' && route
      ? {
          mapRoute: {
            lines: routePolylinesByFloor(route.nodes)[plan.floorId] ?? [],
          } as FloorPlanRoute,
          cues: turnByTurnSteps(route.nodes, route.segments, plan.name, floorLabelFor),
        }
      : undefined

  return (
    <div className="h-dvh">
      <NavigateScreen
        plan={plan}
        route={overlay}
        onBack={() => navigate({ to: '/patient/journey', search: { visit: visitId } })}
      />
    </div>
  )
}
