import type { Locale } from '@/i18n'
import { clockLabel } from '@/i18n/time'
import type { Zone } from '@/design-system'
import { zoneForService } from '@/features/servicepoint/mappings'
import type { AnalyticsOverview, ServicePointMetrics } from './queries'

/**
 * Pure response → props mapping for /staff/overview (#87). All metric
 * formatting lives here so the screen stays markup-only, and the null-≠-0
 * contract (#86) is honoured in exactly one place: a missing metric renders
 * as "—", never as a fabricated 0 or NaN (NFR-10).
 */

/** One table row — metrics already rendered to strings, zone already resolved. */
export type ServicePointLoadView = {
  servicePointId: string
  code: string
  name: string
  zone: Zone
  waitingNow: number
  longestWaiting: string
  avgWait: string
  /** True only for the bottleneck the API named — the one orange row. */
  busy: boolean
}

/** KPI values pre-rendered as the strings StatCard takes. */
export type OverviewCards = {
  activeVisits: string
  /** "—" when the API has no bottleneck to name. */
  bottleneckName: string
  bottleneckNote?: string
  avgWait: string
  /** The longest wait still in queue — the honest companion to the average. */
  longestWaiting: string
  longestWaitingNote?: string
}

export type OverviewView = {
  cards: OverviewCards
  rows: ServicePointLoadView[]
  updatedAt: string
}

/** The pending frame: full layout, honest em dashes — no blank flash, no fake zeros. */
export const emptyOverviewCards: OverviewCards = {
  activeVisits: '—',
  bottleneckName: '—',
  avgWait: '—',
  longestWaiting: '—',
}

/** Minutes with at most one decimal — the console never shows false precision. */
export function minutesLabel(value: number | null): string {
  if (value == null) return '—'
  return `${Math.round(value * 10) / 10}`
}

/**
 * Clock time of the aggregate snapshot, e.g. "14:05 น." (the API stamps
 * Asia/Bangkok). The console pins 'th' (ADR-0012 §2); the locale parameter
 * keeps it off hardcoded `toLocaleTimeString` (#94's one-clock rule).
 */
export function asOfLabel(asOf: string, locale: Locale): string {
  return clockLabel(asOf, locale)
}

function bottleneckRow(overview: AnalyticsOverview): ServicePointMetrics | null {
  const id = overview.summary.bottleneckServicePointId
  return overview.servicePoints.find((sp) => sp.servicePointId === id) ?? null
}

function longestWaitingRow(points: ServicePointMetrics[]): ServicePointMetrics | null {
  let longest: ServicePointMetrics | null = null
  for (const sp of points) {
    if (sp.longestWaitingMinutes == null) continue
    if (longest == null || sp.longestWaitingMinutes > (longest.longestWaitingMinutes ?? 0)) {
      longest = sp
    }
  }
  return longest
}

export function toOverviewView(overview: AnalyticsOverview, locale: Locale): OverviewView {
  const bottleneck = bottleneckRow(overview)
  const longest = longestWaitingRow(overview.servicePoints)
  const bottleneckId = overview.summary.bottleneckServicePointId

  return {
    cards: {
      activeVisits: `${overview.summary.activeVisits}`,
      bottleneckName: bottleneck?.name ?? '—',
      bottleneckNote: bottleneck ? `${bottleneck.waitingNow} คนในคิว` : undefined,
      avgWait: minutesLabel(overview.summary.avgWaitMinutes),
      longestWaiting: minutesLabel(longest?.longestWaitingMinutes ?? null),
      longestWaitingNote: longest ? `ที่ ${longest.name}` : undefined,
    },
    rows: overview.servicePoints.map((sp) => ({
      servicePointId: sp.servicePointId,
      code: sp.code,
      name: sp.name,
      zone: zoneForService(sp.code),
      waitingNow: sp.waitingNow,
      longestWaiting: minutesLabel(sp.longestWaitingMinutes),
      avgWait: minutesLabel(sp.avgWaitMinutes),
      busy: sp.servicePointId === bottleneckId,
    })),
    updatedAt: asOfLabel(overview.asOf, locale),
  }
}
