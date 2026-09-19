import type { ReactNode } from 'react'

export function PageTitle({
  children,
  compact = false,
}: {
  children: ReactNode
  /** Staff mobile density. */
  compact?: boolean
}) {
  return <h1 className={`cp-h1 ${compact ? 'cp-h1--compact' : ''}`}>{children}</h1>
}

export function SectionTitle({
  children,
  compact = false,
}: {
  children: ReactNode
  compact?: boolean
}) {
  return (
    <h2
      className={`cp-h2 ${compact ? 'cp-h2--sm' : ''}`}
      style={{ marginBottom: 'var(--cp-space-md)' }}
    >
      {children}
    </h2>
  )
}

export function Lead({ children }: { children: ReactNode }) {
  return <p className="cp-lead">{children}</p>
}

export function Meta({ children }: { children: ReactNode }) {
  return <p className="cp-meta">{children}</p>
}

/** Short code or numeral — Space Grotesk, never used for prose. */
export function Code({ children }: { children: ReactNode }) {
  return <span className="cp-code">{children}</span>
}
