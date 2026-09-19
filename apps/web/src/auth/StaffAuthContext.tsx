import { createContext, useContext } from 'react'
import type { StaffIdentity } from '@/api/client'

export type StaffAuthStatus = 'loading' | 'unauthenticated' | 'authenticated'

export interface StaffAuthContextValue {
  status: StaffAuthStatus
  identity: StaffIdentity | null
  /** Sign in with username/password; resolves with the identity on success. */
  login: (username: string, password: string) => Promise<StaffIdentity>
  /** Revoke the refresh token server-side and forget the session locally. */
  logout: () => Promise<void>
  /** Re-run the boot-time restore (used after an expiry-driven sign-out). */
  retry: () => void
}

export const StaffAuthContext = createContext<StaffAuthContextValue | undefined>(undefined)

export function useStaffAuth(): StaffAuthContextValue {
  const ctx = useContext(StaffAuthContext)
  if (!ctx) {
    throw new Error('useStaffAuth must be used within a StaffAuthProvider')
  }
  return ctx
}
