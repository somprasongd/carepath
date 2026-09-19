import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { setApiAuthToken } from '@/api/client'
import { AuthContext } from './AuthContext'
import type { AuthContextValue, AuthIdentity, AuthStatus } from './types'

const DEMO_IDENTITY: AuthIdentity = {
  sessionToken: 'demo-session-token',
  source: 'demo',
  externalId: 'demo-user',
  displayName: 'Demo User',
}

export function DemoAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [identity, setIdentity] = useState<AuthIdentity | null>(null)

  const resolve = useCallback(() => {
    // No LIFF SDK, no network calls — synthesizes a fixed identity so
    // `npm run dev` works outside LINE's client and without a LIFF ID.
    setStatus('loading')
    setIdentity(null)
    // Placeholder token — never validates server-side; demo mode is
    // deliberately backendless. Attached anyway so both providers behave
    // identically once bearer-secured endpoints appear.
    setApiAuthToken(DEMO_IDENTITY.sessionToken)
    setIdentity(DEMO_IDENTITY)
    setStatus('authenticated')
  }, [])

  useEffect(() => {
    resolve()
  }, [resolve])

  const value: AuthContextValue = { status, identity, error: undefined, retry: resolve }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
