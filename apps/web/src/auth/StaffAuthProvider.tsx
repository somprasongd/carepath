import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import {
  adoptStaffTokens,
  apiPost,
  clearStaffSession,
  loadStaffRefreshToken,
  onStaffSessionExpired,
  restoreStaffSession,
  type StaffIdentity,
  type StaffTokenPair,
} from '@/api/client'
import { StaffAuthContext } from './StaffAuthContext'
import type { StaffAuthStatus } from './StaffAuthContext'

/**
 * The staff console's session (ADR-0010): restore on mount by exchanging the
 * persisted refresh token, sign in with username/password, sign out by
 * revoking it. When the API client finds the session unrecoverable mid-flight
 * (refresh refused), it calls back here and the state flips to
 * unauthenticated so RequireStaffAuth routes the user back to /login.
 */
export function StaffAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<StaffAuthStatus>('loading')
  const [identity, setIdentity] = useState<StaffIdentity | null>(null)

  const restore = useCallback(async () => {
    if (!loadStaffRefreshToken()) {
      setStatus('unauthenticated')
      setIdentity(null)
      return
    }
    setStatus('loading')
    // Single-flight in the client: StrictMode's double-mounted effect (and
    // any concurrent 401s) share one refresh call — spending the token twice
    // would trip the server's reuse detector.
    const pair = await restoreStaffSession()
    if (pair) {
      setIdentity(pair.identity)
      setStatus('authenticated')
    } else {
      setIdentity(null)
      setStatus('unauthenticated')
    }
  }, [])

  useEffect(() => {
    void restore()
  }, [restore])

  // The client module owns the 401-retry machinery and cannot touch the
  // router; this callback is how an unrecoverable session reaches the UI.
  const expired = useRef(false)
  useEffect(() => {
    onStaffSessionExpired(() => {
      if (expired.current) return
      expired.current = true
      setIdentity(null)
      setStatus('unauthenticated')
    })
    return () => onStaffSessionExpired(undefined)
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const pair = await apiPost<StaffTokenPair>('/api/v1/auth/login', { username, password })
    adoptStaffTokens(pair)
    expired.current = false
    setIdentity(pair.identity)
    setStatus('authenticated')
    return pair.identity
  }, [])

  const logout = useCallback(async () => {
    const refreshToken = loadStaffRefreshToken()
    if (refreshToken) {
      // Best-effort revoke; the local session is cleared either way.
      await apiPost<void>('/api/v1/auth/logout', { refreshToken }).catch(() => undefined)
    }
    clearStaffSession()
    expired.current = false
    setIdentity(null)
    setStatus('unauthenticated')
  }, [])

  return (
    <StaffAuthContext.Provider value={{ status, identity, login, logout, retry: () => void restore() }}>
      {children}
    </StaffAuthContext.Provider>
  )
}
