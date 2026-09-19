import type { ReactNode } from 'react'

/** A numbered chapter of the catalogue. */
export function Section({
  id,
  index,
  title,
  description,
  children,
}: {
  id: string
  index: string
  title: string
  description: ReactNode
  children: ReactNode
}) {
  return (
    <section className="ds-section" id={id}>
      <header className="ds-section__head">
        <div className="ds-section__index">{index}</div>
        <h2 className="ds-section__title">{title}</h2>
        <p className="ds-section__desc">{description}</p>
      </header>
      {children}
    </section>
  )
}

export function SubHead({ children }: { children: ReactNode }) {
  return <h3 className="ds-subhead">{children}</h3>
}

/** One component shown on a stage, with its name and the rule that governs it. */
export function Specimen({
  name,
  note,
  paper = false,
  block = false,
  children,
}: {
  name: string
  note?: ReactNode
  /** Put the specimen on the page background instead of a white surface. */
  paper?: boolean
  /** Stack contents instead of laying them out in a row. */
  block?: boolean
  children: ReactNode
}) {
  return (
    <div className="ds-specimen">
      <div
        className={`ds-specimen__stage ${paper ? 'ds-specimen__stage--paper' : ''} ${
          block ? 'ds-specimen__stage--block' : ''
        }`}
      >
        {children}
      </div>
      <div className="ds-specimen__foot">
        <div className="ds-specimen__name">{name}</div>
        {note && <p className="ds-specimen__note">{note}</p>}
      </div>
    </div>
  )
}

export function Swatch({ value, name, use }: { value: string; name: string; use: string }) {
  return (
    <div className="ds-swatch">
      <div className="ds-swatch__chip" style={{ background: value }} />
      <div className="ds-swatch__body">
        <div className="ds-swatch__name">{name}</div>
        <div className="ds-swatch__hex">{value.toUpperCase()}</div>
        <p className="ds-swatch__use">{use}</p>
      </div>
    </div>
  )
}

/**
 * A reference screen at its real pixel size, optionally scaled down so a
 * 1440px console fits on this page.
 */
export function ScreenFrame({
  title,
  width,
  height,
  scale = 1,
  children,
}: {
  title: string
  width: number
  height: number
  scale?: number
  children: ReactNode
}) {
  return (
    <figure style={{ margin: 0 }}>
      <figcaption className="ds-frame__caption">
        <span className="ds-frame__title">{title}</span>
        <span className="ds-frame__dims">
          {width}×{height}
          {scale !== 1 && ` · ${Math.round(scale * 100)}%`}
        </span>
      </figcaption>
      <div
        className="ds-frame__viewport"
        style={{ width: width * scale, height: height * scale }}
      >
        {/* The frame states the mockup's dimensions — the screen inside fills them. */}
        <div className="ds-frame__inner" style={{ transform: `scale(${scale})`, width, height }}>
          {children}
        </div>
      </div>
    </figure>
  )
}

export function RuleList({
  kind,
  title,
  rules,
}: {
  kind: 'do' | 'dont'
  title: string
  rules: string[]
}) {
  return (
    <div>
      <h3 className="ds-subhead" style={{ marginTop: 0 }}>
        {title}
      </h3>
      <ul className="ds-rule-list">
        {rules.map((rule) => (
          <li className="ds-rule" key={rule}>
            <span className={`ds-rule__mark ds-rule__mark--${kind}`} aria-hidden="true">
              {kind === 'do' ? '✓' : '✕'}
            </span>
            <span>{rule}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
