import type { ReactNode } from 'react'
import { LinkButton } from './Button'
import { InfoIcon, LocationIcon } from './Icon'

/*
 * Not shadcn's Alert: that component is a `role="alert"` live region, and
 * neither of these warns about anything — one states a domain rule, the other
 * reports where the patient is. Announcing either as an alert would interrupt
 * a screen-reader user for no reason.
 */

/** Quiet explanatory note — used to surface a domain rule, not to warn. */
export function InfoNote({ children }: { children: ReactNode }) {
  return (
    <div className="flex gap-2.5 rounded-md border border-line bg-surface p-3">
      <span className="mt-px shrink-0 text-ink-muted">
        <InfoIcon />
      </span>
      <div className="font-sans text-body-sm text-ink-muted">{children}</div>
    </div>
  )
}

export type LocationBannerProps = {
  /** How CarePath currently believes it knows where the patient is (ADR-0004). */
  children: ReactNode
  actionLabel?: string
  onAction?: () => void
}

/** Current-location provider status, in the navigation-node blue. */
export function LocationBanner({ children, actionLabel, onAction }: LocationBannerProps) {
  return (
    <div className="flex items-center gap-2.5 rounded-md border border-secondary-edge bg-secondary-tint px-3 py-2.5">
      <span className="shrink-0 text-secondary">
        <LocationIcon />
      </span>
      <div className="flex-1 font-sans text-caption font-normal leading-[1.5] text-secondary">
        {children}
      </div>
      {actionLabel && <LinkButton onClick={onAction}>{actionLabel}</LinkButton>}
    </div>
  )
}
