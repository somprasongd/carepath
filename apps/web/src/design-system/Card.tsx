import type { HTMLAttributes, ReactNode } from 'react'

export type CardProps = HTMLAttributes<HTMLDivElement> & {
  /**
   * `ticket` is the asymmetric radius reserved for the patient's current step —
   * the card they are "holding" right now. `md` is the flatter staff grammar.
   */
  radius?: 'md' | 'lg' | 'ticket'
  /** `attention` is the orange-tinted KPI/alert surface. */
  tone?: 'surface' | 'attention'
  padding?: 'none' | 'md' | 'lg' | 'xl'
  children: ReactNode
}

export function Card({
  radius = 'lg',
  tone = 'surface',
  padding = 'lg',
  className,
  children,
  ...rest
}: CardProps) {
  const classes = [
    'cp-card',
    radius === 'lg' ? '' : `cp-card--${radius}`,
    tone === 'attention' ? 'cp-card--attention' : '',
    padding === 'lg' ? '' : `cp-card--pad-${padding}`,
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div className={classes} {...rest}>
      {children}
    </div>
  )
}

export function Divider() {
  return <hr className="cp-divider" />
}
