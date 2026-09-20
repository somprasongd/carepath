import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { JourneyScreen } from './-JourneyScreen'
import { VisitEntryScreen } from './-VisitEntryScreen'

/**
 * `?visit=<VN>` — the HIS visit reference this journey renders. Required:
 * without it the VN entry screen asks for it first (no demo default).
 */
export const Route = createFileRoute('/patient/journey')({
  validateSearch: (search: Record<string, unknown>): { visit?: string } => ({
    visit: typeof search.visit === 'string' && search.visit !== '' ? search.visit : undefined,
  }),
  component: PatientJourneyRoute,
})

function PatientJourneyRoute() {
  const navigate = useNavigate()
  const { visit } = Route.useSearch()

  if (visit === undefined) {
    return (
      <div className="h-dvh">
        <VisitEntryScreen />
      </div>
    )
  }

  return (
    <div className="h-dvh">
      <JourneyScreen
        visitId={visit}
        onNavigate={() => navigate({ to: '/patient/navigate', search: { visit } })}
      />
    </div>
  )
}
