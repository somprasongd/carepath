import { Card, RouteStatusBadge, ZoneDot, type RouteStatus, type Zone } from '@/design-system'

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
    <div className="flex items-center gap-3 border-b border-line py-2.5 last:border-b-0">
      <ZoneDot zone={zone} />
      <div className="min-w-0 flex-1">
        <div className="font-code text-[12px]/none font-bold text-ink">{code}</div>
        <div className="font-sans text-[11px] font-normal text-ink-muted">{name}</div>
      </div>
      <div className={`font-code text-label-code ${busy ? 'text-primary' : 'text-ink'}`}>
        {waiting}
      </div>
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
 * desktop mapping table. Care Graph and Navigation Graph stay separate models
 * (ADR-0002); this card is where a human joins them.
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
      <div className="mb-2 flex items-center gap-2">
        <ZoneDot zone={zone} />
        <span className="font-code text-[12px]/none font-bold text-ink">{service}</span>
        <span className="font-sans text-caption font-normal text-ink-muted">{serviceName}</span>
      </div>
      <div className="mb-2.5 font-sans text-caption font-normal text-ink-muted">→ {target}</div>
      {note && (
        <div className="-mt-1.5 mb-2.5 text-[11px] leading-[1.5] text-ink-muted">{note}</div>
      )}
      <RouteStatusBadge status={status} />
    </Card>
  )
}
