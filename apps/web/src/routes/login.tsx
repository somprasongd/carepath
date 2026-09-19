import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { LoginScreen } from './-LoginScreen'

/** Staff/admin/executive entry point. Patients never see this — they enter via LINE. */
export const Route = createFileRoute('/login')({
  component: LoginRoute,
})

function LoginRoute() {
  const navigate = useNavigate()
  return <LoginScreen onSignIn={(landing) => navigate({ to: landing })} />
}
