import type { ReactNode } from 'react'
import { zones, type Zone } from './tokens'

export type ZoneChipProps = {
  zone: Zone
  /** Defaults to the zone's own label. */
  children?: ReactNode
}

/**
 * The atomic unit that ties a staff list row to an area on the map. Text is
 * always `ink` — every zone tint is pale (DESIGN.md).
 */
export function ZoneChip({ zone, children }: ZoneChipProps) {
  return (
    <span className="cp-chip" data-zone={zone}>
      {children ?? zones[zone].label}
    </span>
  )
}

/** The chip reduced to its colour — for dense rows where the code carries the name. */
export function ZoneDot({ zone }: { zone: Zone }) {
  return <span className="cp-dot" data-zone={zone} aria-hidden="true" />
}

/**
 * Reminds staff that these are the same colours the patient sees on the map.
 * Shown on the console, never on a patient screen.
 */
export function ZoneLegend({ only }: { only?: Zone[] }) {
  const keys = (only ?? (Object.keys(zones) as Zone[])) as Zone[]
  return (
    <ul className="cp-legend">
      {keys.map((zone) => (
        <li className="cp-legend__row" key={zone}>
          <span className="cp-swatch-box" data-zone={zone} aria-hidden="true" />
          <span className="cp-legend__label">
            {zones[zone].label} — {zones[zone].use}
          </span>
        </li>
      ))}
    </ul>
  )
}
