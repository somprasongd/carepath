import { Navigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { useStaffAuth } from './StaffAuthContext'

/**
 * Gate for /staff/*: while the session restores render nothing, and once it
 * is clear no session survives, bounce to /login. Role checks are NOT
 * enforcement — the API is — this only keeps the console's screens from
 * flashing their frames to a signed-out visitor.
 */
export function RequireStaffAuth({ children }: { children: ReactNode }) {
  const { status } = useStaffAuth()

  if (status === 'loading') return null
  if (status === 'unauthenticated') return <Navigate to="/login" replace />

  return <>{children}</>
}
