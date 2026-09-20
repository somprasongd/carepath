import type { RouteStatus, Zone } from '@/design-system'
import type { ServicePoint } from './queries'

/**
 * What one service point looks like in the staff console — the same props the
 * mapping card takes, so the screen and the card stay one model. Translated
 * here (not in the design system) because zone/route status are domain calls:
 * zone follows the service code's functional area, routable follows the
 * place's navigation entry node.
 */
export type ServicePointView = {
  zone: Zone
  /** The clinical service code in HIS vocabulary, e.g. "LAB". */
  service: string
  serviceName: string
  /** Where it resolves to, e.g. "LAB-01 · Blood Collection · ชั้น 2". */
  target: string
  status: RouteStatus
}

/**
 * Functional area per service code; unknown codes fall back to public.
 * Binding codes reuse the bare vocabulary behind a kind prefix (ADR-0009):
 * "ORDERTYPE:LAB" classifies like LAB, and any "CLINIC:*" is a doctor
 * visit. Exported because the overview table (#87) colours its rows with
 * the same call — the two screens can't disagree about a zone.
 */
export function zoneForService(code: string): Zone {
  const base = code.replace(/^(ORDERTYPE|CLINIC):/, '')
  switch (code.startsWith('CLINIC:') ? 'DOCTOR' : base) {
    case 'DOCTOR':
      return 'opd'
    case 'LAB':
    case 'XRAY':
    case 'EKG':
    case 'ULTRASOUND':
      return 'diagnostic'
    case 'PHARMACY':
      return 'pharmacy'
    default:
      return 'public'
  }
}

/**
 * A service point is routable when its place is in the map and carries a
 * navigation-graph entry node (DESIGN.md: never fake a route — the explicit
 * "ยังไม่รองรับเส้นทาง" state is the honest one).
 */
export function toServicePointView(sp: ServicePoint): ServicePointView {
  const place = sp.place ?? null
  return {
    zone: zoneForService(sp.code),
    service: sp.code,
    serviceName: sp.name,
    target: place
      ? `${place.id} · ${place.name} · ชั้น ${place.floor.code}`
      : 'ยังไม่อยู่ในผังอาคาร',
    status: place?.entryNodeId ? 'routable' : 'not-routable',
  }
}
