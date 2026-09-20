import { describe, expect, it } from 'vitest'
import type { AnalyticsOverview, ServicePointMetrics } from './queries'
import { minutesLabel, toOverviewView } from './overview'

function point(overrides: Partial<ServicePointMetrics> = {}): ServicePointMetrics {
  return {
    servicePointId: 'SP-OPD',
    code: 'DOCTOR',
    name: 'OPD Exam',
    waitingNow: 2,
    inProgressNow: 1,
    longestWaitingMinutes: 9,
    avgWaitMinutes: 10,
    avgServiceMinutes: 8,
    completedCount: 4,
    ...overrides,
  }
}

function overview(overrides: Partial<AnalyticsOverview> = {}): AnalyticsOverview {
  return {
    asOf: '2026-09-20T04:05:06Z',
    window: 'today',
    summary: {
      activeVisits: 7,
      avgVisitMinutes: 41,
      avgWaitMinutes: 12.5,
      bottleneckServicePointId: 'SP-LAB',
    },
    servicePoints: [
      point(),
      point({
        servicePointId: 'SP-LAB',
        code: 'LAB',
        name: 'Blood Collection',
        waitingNow: 5,
        longestWaitingMinutes: 23,
        avgWaitMinutes: 12.5,
      }),
    ],
    ...overrides,
  }
}

describe('minutesLabel', () => {
  it('renders null as an em dash, never 0', () => {
    expect(minutesLabel(null)).toBe('—')
  })

  it('keeps at most one decimal', () => {
    expect(minutesLabel(8)).toBe('8')
    expect(minutesLabel(12.5)).toBe('12.5')
    expect(minutesLabel(12.34)).toBe('12.3')
  })
})

describe('toOverviewView', () => {
  it('maps cards from the summary: bottleneck by id, longest wait across rows', () => {
    const { cards } = toOverviewView(overview(), 'th')

    expect(cards.activeVisits).toBe('7')
    expect(cards.avgWait).toBe('12.5')
    expect(cards.bottleneckName).toBe('Blood Collection')
    expect(cards.bottleneckNote).toBe('5 คนในคิว')
    expect(cards.longestWaiting).toBe('23')
    expect(cards.longestWaitingNote).toBe('ที่ Blood Collection')
  })

  it('marks only the bottleneck row busy and resolves zones per service code', () => {
    const { rows } = toOverviewView(overview(), 'th')

    expect(rows).toHaveLength(2)
    const lab = rows.find((row) => row.code === 'LAB')
    const opd = rows.find((row) => row.code === 'DOCTOR')
    expect(lab?.busy).toBe(true)
    expect(lab?.zone).toBe('diagnostic')
    expect(opd?.busy).toBe(false)
    expect(opd?.zone).toBe('opd')
  })

  it('renders every metric as "—" when the API reports no data yet — no 0, no NaN', () => {
    const view = toOverviewView(
      overview({
        summary: {
          activeVisits: 0,
          avgVisitMinutes: null,
          avgWaitMinutes: null,
          bottleneckServicePointId: null,
        },
        servicePoints: [
          point({
            waitingNow: 0,
            longestWaitingMinutes: null,
            avgWaitMinutes: null,
          }),
        ],
      }),
      'th',
    )

    expect(view.cards.activeVisits).toBe('0')
    expect(view.cards.avgWait).toBe('—')
    expect(view.cards.bottleneckName).toBe('—')
    expect(view.cards.bottleneckNote).toBeUndefined()
    expect(view.cards.longestWaiting).toBe('—')
    expect(view.cards.longestWaitingNote).toBeUndefined()
    expect(view.rows[0]?.busy).toBe(false)
    expect(view.rows[0]?.longestWaiting).toBe('—')
    expect(view.rows[0]?.avgWait).toBe('—')
  })

  it('picks the longest wait from the row that has one, skipping nulls', () => {
    const { cards } = toOverviewView(
      overview({
        servicePoints: [
          point({ servicePointId: 'SP-XRAY', longestWaitingMinutes: null }),
          point({ servicePointId: 'SP-LAB', longestWaitingMinutes: 17.5 }),
        ],
      }),
      'th',
    )

    expect(cards.longestWaiting).toBe('17.5')
  })

  it('formats the snapshot clock time through the shared clock, น. and all', () => {
    const { updatedAt } = toOverviewView(overview(), 'th')

    // Timezone-dependent like the old assertion, but now asserts the Thai
    // marker that the one clock formatter (i18n/time.ts) appends.
    expect(updatedAt).toMatch(/^[0-9]{2}:[0-9]{2} น\.$/)
  })

  it('renders the snapshot clock bare in English', () => {
    const { updatedAt } = toOverviewView(overview(), 'en')

    expect(updatedAt).toMatch(/^[0-9]{2}:[0-9]{2}$/)
    expect(updatedAt).not.toContain('น.')
  })
})
