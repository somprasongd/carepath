import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import liff from '@line/liff'
import { apiPost } from '@/api/client'
import type { components } from '@/api/schema'
import { AuthContext } from './AuthContext'
import type { AuthContextValue, AuthIdentity, AuthStatus } from './types'

type CreateSessionResponse = components['schemas']['CreateSessionResponse']

const liffId = import.meta.env.VITE_LIFF_ID

function extractErrorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  if (typeof err === 'object' && err !== null && 'message' in err) {
    const message = (err as { message?: unknown }).message
    if (typeof message === 'string' && message.length > 0) return message
  }
  return 'LIFF initialization failed for an unknown reason.'
}

export function LiffAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [identity, setIdentity] = useState<AuthIdentity | null>(null)
  const [error, setError] = useState<string>()
  const initStarted = useRef(false)

  const runAuthFlow = useCallback(async () => {
    setStatus('loading')
    setError(undefined)
    setIdentity(null)

    if (!liffId) {
      setStatus('error')
      setError('VITE_LIFF_ID is not configured. Set it in .env to use VITE_AUTH_MODE=line.')
      return
    }

    try {
      await liff.init({ liffId })

      if (!liff.isLoggedIn()) {
        // liff.login() navigates away (in-client webview or full external-browser
        // redirect); it returns to this same LIFF URL, where liff.init() runs
        // again and isLoggedIn() re-evaluates to true. No callback route needed.
        setStatus('unauthenticated')
        liff.login()
        return
      }

      const idToken = liff.getIDToken()
      if (!idToken) {
        // Real failure mode: openid scope not enabled for this LIFF app in the
        // LINE Developers console. Login can "succeed" with no ID token.
        setStatus('error')
        setError(
          'Logged in but no ID token was returned. Confirm the "openid" scope is ' +
            'enabled for this LIFF app in the LINE Developers console.',
        )
        return
      }

      const session = await apiPost<CreateSessionResponse>('/api/v1/auth/session', {
        source: 'line',
        idToken,
      })
      setIdentity({
        sessionToken: session.sessionToken,
        source: 'line',
        externalId: session.identity.externalId,
        displayName: session.identity.displayName,
      })
      setStatus('authenticated')
    } catch (err) {
      setStatus('error')
      setError(extractErrorMessage(err))
    }
  }, [])

  useEffect(() => {
    // Guards against React StrictMode's dev-only double-invoke of effects
    // re-running liff.init() twice on the same mount.
    if (initStarted.current) return
    initStarted.current = true
    void runAuthFlow()
  }, [runAuthFlow])

  const retry = useCallback(() => {
    initStarted.current = false
    void runAuthFlow()
  }, [runAuthFlow])

  const value: AuthContextValue = { status, identity, error, retry }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
