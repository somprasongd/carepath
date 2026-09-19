import type { ReactNode } from 'react'
import { InfoIcon, LocationIcon } from './Icon'

/** Quiet explanatory note — used to surface a domain rule, not to warn. */
export function InfoNote({ children }: { children: ReactNode }) {
  return (
    <div className="cp-note">
      <span className="cp-note__icon">
        <InfoIcon />
      </span>
      <div className="cp-note__text">{children}</div>
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
    <div className="cp-locate">
      <span className="cp-locate__icon">
        <LocationIcon />
      </span>
      <div className="cp-locate__text">{children}</div>
      {actionLabel && (
        <button type="button" className="cp-link" onClick={onAction}>
          {actionLabel}
        </button>
      )}
    </div>
  )
}
