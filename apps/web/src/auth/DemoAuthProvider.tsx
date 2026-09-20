import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { apiPost, setApiAuthToken } from '@/api/client'
import type { components } from '@/api/schema'
import { AuthContext } from './AuthContext'
import type { AuthContextValue, AuthIdentity, AuthStatus } from './types'

type CreateSessionResponse = components['schemas']['CreateSessionResponse']

const DEMO_FALLBACK_IDENTITY: AuthIdentity = {
  source: 'demo',
  externalId: 'demo-user',
  displayName: 'Demo User',
}

export function DemoAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [identity, setIdentity] = useState<AuthIdentity | null>(null)

  const resolve = useCallback(async () => {
    // No LIFF SDK — demo mode asks the API for a real patient session
    // (ALLOW_DEMO_AUTH, ADR-0010) so bearer-secured actions like sharing a
    // visit (#90) work outside LINE's client. The API being down must not
    // take the journey screens with it: they render tokenless, and the share
    // button disables itself with a reason instead of breaking the page.
    setStatus('loading')
    setIdentity(null)
    setApiAuthToken(undefined)

    try {
      const session = await apiPost<CreateSessionResponse>('/api/v1/auth/session', {
        source: 'demo',
      })
      setApiAuthToken(session.sessionToken)
      setIdentity({
        sessionToken: session.sessionToken,
        source: 'demo',
        externalId: session.identity.externalId,
        displayName: session.identity.displayName,
      })
    } catch {
      setApiAuthToken(undefined)
      setIdentity(DEMO_FALLBACK_IDENTITY)
    }
    setStatus('authenticated')
  }, [])

  useEffect(() => {
    void resolve()
  }, [resolve])

  const value: AuthContextValue = { status, identity, error: undefined, retry: () => void resolve() }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
