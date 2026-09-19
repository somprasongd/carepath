import type { ReactNode } from 'react'
import { zones, type Zone } from './tokens'
import { Badge } from './ui/badge'

/*
 * Every zone is spelled out because Tailwind only emits classes it can read in
 * the source — `bg-zone-${zone}` would compile to nothing. These fills must
 * stay byte-identical to the `:root` values in
 * packages/floorplans/floors/*.svg (ADR-0003); src/styles/index.css holds the
 * single copy of the values themselves.
 */
const chipFill: Record<Zone, string> = {
  public: 'bg-zone-public',
  opd: 'bg-zone-opd',
  diagnostic: 'bg-zone-diagnostic',
  pharmacy: 'bg-zone-pharmacy',
  rehab: 'bg-zone-rehab',
  ipd: 'bg-zone-ipd',
  support: 'bg-zone-support',
}

const dotFill: Record<Zone, string> = {
  public: 'bg-zone-public border-zone-public-edge',
  opd: 'bg-zone-opd border-zone-opd-edge',
  diagnostic: 'bg-zone-diagnostic border-zone-diagnostic-edge',
  pharmacy: 'bg-zone-pharmacy border-zone-pharmacy-edge',
  rehab: 'bg-zone-rehab border-zone-rehab-edge',
  ipd: 'bg-zone-ipd border-zone-ipd-edge',
  support: 'bg-zone-support border-zone-support-edge',
}

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
    <Badge variant="zone" className={chipFill[zone]}>
      {children ?? zones[zone].label}
    </Badge>
  )
}

/** The chip reduced to its colour — for dense rows where the code carries the name. */
export function ZoneDot({ zone }: { zone: Zone }) {
  return (
    <span
      className={`inline-block size-2.5 shrink-0 rounded-full border ${dotFill[zone]}`}
      aria-hidden="true"
    />
  )
}

/**
 * Reminds staff that these are the same colours the patient sees on the map.
 * Shown on the console, never on a patient screen.
 */
export function ZoneLegend({ only }: { only?: Zone[] }) {
  const keys = (only ?? (Object.keys(zones) as Zone[])) as Zone[]

  return (
    <ul className="m-0 flex list-none flex-col gap-2 p-0">
      {keys.map((zone) => (
        <li className="flex items-center gap-2" key={zone}>
          <span
            className={`inline-block size-3 shrink-0 rounded-[4px] ${chipFill[zone]}`}
            aria-hidden="true"
          />
          <span className="font-sans text-caption font-normal text-ink-muted">
            {zones[zone].label} — {zones[zone].use}
          </span>
        </li>
      ))}
    </ul>
  )
}
