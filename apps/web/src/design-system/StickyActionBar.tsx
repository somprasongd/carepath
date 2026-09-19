import type { ReactNode } from 'react'

export type StickyActionBarProps = {
  /** Small label above the action, e.g. "ขั้นตอนถัดไป". */
  label: string
  /** What the action leads to, e.g. "รับยา · ห้องยา ชั้น 1". */
  value: string
  /** Exactly one primary action — this is the one-thumb zone. */
  children: ReactNode
}

/** The other element allowed a shadow: it floats over the scrolling journey. */
export function StickyActionBar({ label, value, children }: StickyActionBarProps) {
  return (
    <div className="flex flex-col gap-2.5 border-t border-line bg-surface px-gutter pt-[14px] pb-[22px] shadow-bar">
      <div className="flex items-baseline gap-2">
        <span className="font-sans text-caption font-normal text-ink-muted">{label}</span>
        <span className="font-sans text-body-md font-bold text-ink">{value}</span>
      </div>
      {children}
    </div>
  )
}
