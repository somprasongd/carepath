export type AuthStatus = 'loading' | 'unauthenticated' | 'authenticated' | 'error'

export type AuthSource = 'line' | 'demo'

export interface AuthIdentity {
  /**
   * The patient session token (ADR-0010). Absent only in demo mode when the
   * session endpoint could not be reached: the journey screens keep working
   * (they fall back to no bearer), while share actions that need a real
   * session disable themselves (#90).
   */
  sessionToken?: string
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
