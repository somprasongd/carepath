import type { ReactNode } from 'react'
import { Divider } from './Card'

export type BottomSheetProps = {
  /** Headline figure — walking time in the signage numeral face. */
  primary: string
  /** Supporting figure, e.g. "· 65 m". */
  secondary?: string
  /** Turn-by-turn text: the accessible alternative to reading the map. */
  steps: string[]
  footer?: ReactNode
}

/**
 * One of only two elements allowed a shadow — it genuinely floats over the map
 * beneath it.
 *
 * Deliberately not shadcn's Drawer: this sheet is part of the layout and is
 * always visible, not a modal with an overlay that dismisses the map behind
 * it. DESIGN.md fixes its anatomy — rounded top corners, drag handle, a
 * display-stat summary line, then the numbered steps.
 */
export function BottomSheet({ primary, secondary, steps, footer }: BottomSheetProps) {
  return (
    <div className="flex flex-col gap-[14px] rounded-t-xl bg-surface px-gutter pt-3 pb-6 shadow-sheet">
      <div className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
      <div className="flex items-baseline gap-2">
        <span className="font-code text-[22px] font-bold text-ink">{primary}</span>
        {secondary && <span className="font-sans text-body-sm text-ink-muted">{secondary}</span>}
      </div>
      <Divider />
      <ol className="m-0 flex list-none flex-col gap-3 p-0">
        {steps.map((text, i) => (
          <li className="flex items-start gap-2.5" key={text}>
            <span className="flex size-[22px] shrink-0 items-center justify-center rounded-full border border-line bg-neutral font-sans text-caption font-bold text-ink-muted">
              {i + 1}
            </span>
            <span className="pt-0.5 font-sans text-body-sm text-ink">{text}</span>
          </li>
        ))}
      </ol>
      {footer}
    </div>
  )
}
