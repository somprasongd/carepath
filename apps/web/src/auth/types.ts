export type AuthStatus = 'loading' | 'unauthenticated' | 'authenticated' | 'error'

export type AuthSource = 'line' | 'demo'

export interface AuthIdentity {
  sessionToken: string
  source: AuthSource
  externalId: string
  displayName?: string
}

export interface AuthContextValue {
  status: AuthStatus
  identity: AuthIdentity | null
  error?: string
  retry: () => void
}
