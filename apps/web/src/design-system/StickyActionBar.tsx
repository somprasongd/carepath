import type { ReactNode } from 'react'

export type StickyActionBarProps = {
  /** Small label above the action, e.g. "ขั้นตอนถัดไป". */
  label: string
  /** What the action leads to, e.g. "รับยา · ห้องยา ชั้น 1". */
  value: string
  /** Exactly one primary action — this is the one-thumb zone. */
  children: ReactNode
}

export function StickyActionBar({ label, value, children }: StickyActionBarProps) {
  return (
    <div className="cp-actionbar">
      <div className="cp-actionbar__context">
        <span className="cp-actionbar__label">{label}</span>
        <span className="cp-actionbar__value">{value}</span>
      </div>
      {children}
    </div>
  )
}
