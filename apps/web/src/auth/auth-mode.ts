const rawAuthMode = import.meta.env.VITE_AUTH_MODE

if (rawAuthMode !== undefined && rawAuthMode !== 'line' && rawAuthMode !== 'demo') {
  console.warn(`Unknown VITE_AUTH_MODE "${rawAuthMode}", falling back to "demo".`)
}

/**
 * The patient surface's auth mode, resolved once at load: `line` (LIFF login,
 * production) or `demo` (VN entry, ALLOW_DEMO_AUTH deployments). Split out of
 * AuthProvider.tsx so non-React modules (visit queries) can branch on it
 * without importing a component tree.
 */
export const authMode: 'line' | 'demo' = rawAuthMode === 'line' ? 'line' : 'demo'
