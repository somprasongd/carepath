import type { ReactNode } from 'react'
import { useAuth } from './AuthContext'

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status } = useAuth()

  if (status !== 'authenticated') {
    // LoginGate (rendered above this in the tree) already owns every
    // non-authenticated UI state; this is a defensive second gate, and the
    // reusable primitive for any future route that needs the same check.
    return null
  }

  return <>{children}</>
}
