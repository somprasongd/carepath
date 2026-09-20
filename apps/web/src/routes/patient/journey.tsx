import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { authMode } from '@/auth/auth-mode'
import { readVisitLinkToken } from '@/features/visit/visit-link'
import { JourneyScreen } from './-JourneyScreen'
import { NoVisitScreen, VisitLinkExchange } from './-VisitLinkExchange'
import { VisitEntryScreen } from './-VisitEntryScreen'

/**
 * `?visit=<VN>` — the visit this journey renders, reached one of two ways:
 * the slip-link token (#136, `#vt=` captured at bootstrap and redeemed for
 * the visit id — production's only front door), or the demo VN entry screen
 * (ALLOW_DEMO_AUTH deployments). In line mode with neither, the no-visit
 * screen points back to the hospital's slip instead of accepting a typed VN.
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
    if (readVisitLinkToken() !== undefined) {
      return (
        <div className="h-dvh">
          <VisitLinkExchange />
        </div>
      )
    }
    return (
      <div className="h-dvh">
        {authMode === 'demo' ? <VisitEntryScreen /> : <NoVisitScreen />}
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
