import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { JourneyScreen } from './-JourneyScreen'

export const Route = createFileRoute('/patient/journey')({
  component: PatientJourneyRoute,
})

function PatientJourneyRoute() {
  const navigate = useNavigate()

  return (
    <div className="cp-viewport">
      <JourneyScreen onNavigate={() => navigate({ to: '/patient/navigate' })} />
    </div>
  )
}
