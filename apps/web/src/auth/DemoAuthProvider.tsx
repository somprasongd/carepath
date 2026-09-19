import { useCallback, useEffect, useState, type ReactNode } from 'react'
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
    setIdentity(DEMO_IDENTITY)
    setStatus('authenticated')
  }, [])

  useEffect(() => {
    resolve()
  }, [resolve])

  const value: AuthContextValue = { status, identity, error: undefined, retry: resolve }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
