import type { ReactNode } from 'react'

export type BottomSheetProps = {
  /** Headline figure — walking time in the signage numeral face. */
  primary: string
  /** Supporting figure, e.g. "· 65 เมตร". */
  secondary?: string
  /** Turn-by-turn text: the accessible alternative to reading the map. */
  steps: string[]
  footer?: ReactNode
}

/**
 * One of only two elements allowed a shadow — it genuinely floats over the map
 * beneath it.
 */
export function BottomSheet({ primary, secondary, steps, footer }: BottomSheetProps) {
  return (
    <div className="cp-sheet">
      <div className="cp-sheet__handle" aria-hidden="true" />
      <div className="cp-sheet__summary">
        <span className="cp-sheet__primary">{primary}</span>
        {secondary && <span className="cp-sheet__secondary">{secondary}</span>}
      </div>
      <hr className="cp-divider" />
      <ol className="cp-sheet__steps">
        {steps.map((text, i) => (
          <li className="cp-sheet__step" key={text}>
            <span className="cp-sheet__step-num">{i + 1}</span>
            <span className="cp-sheet__step-text">{text}</span>
          </li>
        ))}
      </ol>
      {footer}
    </div>
  )
}
