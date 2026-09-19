import { Outlet, createFileRoute } from '@tanstack/react-router'
import { AuthProvider } from '@/auth/AuthProvider'
import { LoginGate } from '@/auth/LoginGate'
import { RequireAuth } from '@/auth/RequireAuth'

/**
 * Layout route for everything under /patient — the LINE LIFF surface. Gates
 * `journey` and `navigate` behind LIFF login (or the demo stub) from one
 * place instead of duplicating the check per route.
 */
export const Route = createFileRoute('/patient')({
  component: () => (
    <AuthProvider>
      <LoginGate>
        <RequireAuth>
          <Outlet />
        </RequireAuth>
      </LoginGate>
    </AuthProvider>
  ),
})
