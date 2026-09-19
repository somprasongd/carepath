import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { NavigateScreen } from './-NavigateScreen'

export const Route = createFileRoute('/patient/navigate')({
  component: PatientNavigateRoute,
})

function PatientNavigateRoute() {
  const navigate = useNavigate()

  return (
    <div className="h-dvh">
      <NavigateScreen onBack={() => navigate({ to: '/patient/journey' })} />
    </div>
  )
}
