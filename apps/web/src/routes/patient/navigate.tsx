import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { navigatePlanForJourney } from '@/features/floorplan'
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

  return (
    <div className="h-dvh">
      <NavigateScreen
        plan={navigatePlanForJourney(data, isPending)}
        onBack={() => navigate({ to: '/patient/journey', search: { visit: visitId } })}
      />
    </div>
  )
}
