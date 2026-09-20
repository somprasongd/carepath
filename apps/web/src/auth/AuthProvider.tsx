import type { ReactNode } from 'react'
import { authMode } from './auth-mode'
import { DemoAuthProvider } from './DemoAuthProvider'
import { LiffAuthProvider } from './LiffAuthProvider'

export function AuthProvider({ children }: { children: ReactNode }) {
  if (authMode === 'line') {
    return <LiffAuthProvider>{children}</LiffAuthProvider>
  }
  return <DemoAuthProvider>{children}</DemoAuthProvider>
}
