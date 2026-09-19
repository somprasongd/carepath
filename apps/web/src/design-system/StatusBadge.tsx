import type { ReactNode } from 'react'
import { Badge } from './ui/badge'

export type RouteStatus = 'routable' | 'not-routable'

/**
 * Whether a Place currently has a navigation-graph entry node. A Place without
 * one stays visible and selectable — it just says so (DESIGN.md, and
 * packages/floorplans/README.md for the real cases: Rehab, IPD).
 */
export function RouteStatusBadge({ status }: { status: RouteStatus }) {
  return status === 'routable' ? (
    <Badge variant="routable">เดินทางได้</Badge>
  ) : (
    <Badge variant="blocked">ยังไม่รองรับเส้นทาง</Badge>
  )
}

/** Operational load on a service point — never a red/green alert colour. */
export function LoadBadge({ load }: { load: 'normal' | 'busy' }) {
  return load === 'busy' ? (
    <Badge variant="busy">หนาแน่น</Badge>
  ) : (
    <Badge variant="quiet">ปกติ</Badge>
  )
}

/** Queue number in the signage numeral face. */
export function QueuePill({ children }: { children: ReactNode }) {
  return <Badge variant="queue">{children}</Badge>
}

/** Visit / patient reference code, shown as a quiet outlined pill. */
export function RefPill({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <Badge variant="ref" className={className}>
      {children}
    </Badge>
  )
}
