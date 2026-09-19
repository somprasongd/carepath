import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { DEFAULT_VISIT_ID } from '@/features/visit'
import { JourneyScreen } from './-JourneyScreen'

/** `?visit=<id>` — which HIS visit this journey renders; demo default for now. */
export const Route = createFileRoute('/patient/journey')({
  validateSearch: (search: Record<string, unknown>): { visit?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
  }),
  component: PatientJourneyRoute,
})

function PatientJourneyRoute() {
  const navigate = useNavigate()
  const { visit } = Route.useSearch()
  const visitId = visit ?? DEFAULT_VISIT_ID

  return (
    <div className="h-dvh">
      <JourneyScreen
        visitId={visitId}
        onNavigate={() => navigate({ to: '/patient/navigate', search: { visit: visitId } })}
      />
    </div>
  )
}
