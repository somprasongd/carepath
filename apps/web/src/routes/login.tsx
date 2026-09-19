import { Navigate, createFileRoute } from '@tanstack/react-router'
import { StaffAuthProvider } from '@/auth/StaffAuthProvider'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import { LoginScreen } from './-LoginScreen'

/** Staff/admin entry point. Patients never see this — they enter via LINE. */
export const Route = createFileRoute('/login')({
  component: () => (
    <div className="h-dvh">
      <StaffAuthProvider>
        <LoginRoute />
      </StaffAuthProvider>
    </div>
  ),
})

// Already signed in (a session restored from the kept refresh token)? Straight
// to the console — the form is for everyone else.
function LoginRoute() {
  const { status } = useStaffAuth()
  if (status === 'authenticated') return <Navigate to="/staff/queue" replace />
  if (status === 'loading') return null
  return <LoginScreen />
}
