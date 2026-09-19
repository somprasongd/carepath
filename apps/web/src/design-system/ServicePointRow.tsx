import { Card } from './Card'
import { RouteStatusBadge, type RouteStatus } from './StatusBadge'
import { ZoneDot } from './ZoneChip'
import type { Zone } from './tokens'

export type ServicePointRowProps = {
  zone: Zone
  /** Service point code in the signage numeral face, e.g. "LAB-01". */
  code: string
  name: string
  /** People currently waiting. */
  waiting: number
  /** Marks the busiest point — the one orange number in the list. */
  busy?: boolean
}

/** Dense staff list row: colour, code, name, count. Scannable against the map. */
export function ServicePointRow({ zone, code, name, waiting, busy }: ServicePointRowProps) {
  return (
    <div className="cp-sp-row">
      <ZoneDot zone={zone} />
      <div className="cp-sp-row__body">
        <div className="cp-sp-row__code">{code}</div>
        <div className="cp-sp-row__name">{name}</div>
      </div>
      <div className={`cp-sp-row__count ${busy ? 'cp-sp-row__count--busy' : ''}`}>{waiting}</div>
    </div>
  )
}

export type ServicePointMappingProps = {
  zone: Zone
  /** The clinical service, e.g. "LAB". */
  service: string
  serviceName: string
  /** Where it resolves to, e.g. "LAB-01 · ห้องเจาะเลือด · ชั้น 2". */
  target: string
  note?: string
  status: RouteStatus
}

/**
 * One care-step → place link, the mobile counterpart of a row in the staff
 * desktop mapping table.
 */
export function ServicePointMappingCard({
  zone,
  service,
  serviceName,
  target,
  note,
  status,
}: ServicePointMappingProps) {
  return (
    <Card radius="md" padding="md">
      <div className="cp-mapping__head">
        <ZoneDot zone={zone} />
        <span className="cp-sp-row__code">{service}</span>
        <span className="cp-mapping__target" style={{ margin: 0 }}>
          {serviceName}
        </span>
      </div>
      <div className="cp-mapping__target">→ {target}</div>
      {note && <div className="cp-mapping__note">{note}</div>}
      <RouteStatusBadge status={status} />
    </Card>
  )
}
