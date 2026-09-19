export type AuthStatus = 'loading' | 'unauthenticated' | 'authenticated' | 'error'

export type AuthSource = 'line' | 'demo'

export interface AuthIdentity {
  idToken: string
  source: AuthSource
  displayName?: string
}

export interface AuthContextValue {
  status: AuthStatus
  identity: AuthIdentity | null
  error?: string
  retry: () => void
}
