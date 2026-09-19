import type { ReactNode } from 'react'

export type RouteStatus = 'routable' | 'not-routable'

/**
 * Whether a Place currently has a navigation-graph entry node. A Place without
 * one stays visible and selectable — it just says so (DESIGN.md, and
 * packages/floorplans/README.md for the real cases: Rehab, IPD).
 */
export function RouteStatusBadge({ status }: { status: RouteStatus }) {
  return status === 'routable' ? (
    <span className="cp-badge cp-badge--routable">เดินทางได้</span>
  ) : (
    <span className="cp-badge cp-badge--blocked">ยังไม่รองรับเส้นทาง</span>
  )
}

/** Operational load on a service point — never a red/green alert colour. */
export function LoadBadge({ load }: { load: 'normal' | 'busy' }) {
  return load === 'busy' ? (
    <span className="cp-badge cp-badge--busy">หนาแน่น</span>
  ) : (
    <span className="cp-badge cp-badge--quiet">ปกติ</span>
  )
}

/** Queue number in the signage numeral face. */
export function QueuePill({ children }: { children: ReactNode }) {
  return <span className="cp-queue-pill">{children}</span>
}

/** Visit / patient reference code, shown as a quiet outlined pill. */
export function RefPill({ children }: { children: ReactNode }) {
  return <span className="cp-ref-pill">{children}</span>
}
