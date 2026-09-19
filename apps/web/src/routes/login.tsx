import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { LoginScreen } from './-LoginScreen'

/** Staff/admin/executive entry point. Patients never see this — they enter via LINE. */
export const Route = createFileRoute('/login')({
  component: LoginRoute,
})

function LoginRoute() {
  const navigate = useNavigate()
  return (
    <div className="h-dvh">
      <LoginScreen onSignIn={(landing) => navigate({ to: landing })} />
    </div>
  )
}
