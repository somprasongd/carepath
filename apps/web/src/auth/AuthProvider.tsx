import type { ReactNode } from 'react'
import { DemoAuthProvider } from './DemoAuthProvider'
import { LiffAuthProvider } from './LiffAuthProvider'

const rawAuthMode = import.meta.env.VITE_AUTH_MODE

if (rawAuthMode !== undefined && rawAuthMode !== 'line' && rawAuthMode !== 'demo') {
  console.warn(`Unknown VITE_AUTH_MODE "${rawAuthMode}", falling back to "demo".`)
}

const authMode: 'line' | 'demo' = rawAuthMode === 'line' ? 'line' : 'demo'

export function AuthProvider({ children }: { children: ReactNode }) {
  if (authMode === 'line') {
    return <LiffAuthProvider>{children}</LiffAuthProvider>
  }
  return <DemoAuthProvider>{children}</DemoAuthProvider>
}
