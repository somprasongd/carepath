import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import type { SharedJourney } from './queries'
import {
  shareExpiryLabel,
  sharedScreenModel,
  sharedStatusLabel,
  sharedUpdatedLabel,
  sharedWhereLine,
  thaiClock,
} from './shared-view'

function sharedJourney(overrides: Partial<SharedJourney> = {}): SharedJourney {
  return {
    status: 'WAITING',
    currentStep: null,
    updatedAt: '2026-09-20T07:30:00Z',
    expiresAt: '2026-09-20T11:30:00Z',
    ...overrides,
  }
}

describe('sharedStatusLabel — plain Thai, exhaustive over the enum', () => {
  it.each([
    ['WAITING', 'กำลังรอคิว'],
    ['IN_SERVICE', 'กำลังรับบริการ'],
    ['DONE', 'เสร็จเรียบร้อย'],
  ] as const)('maps %s', (status, label) => {
    expect(sharedStatusLabel(status)).toBe(label)
  })

  it('falls back to a neutral label rather than echoing an unknown value', () => {
    expect(sharedStatusLabel('SOMETHING_NEW' as SharedJourney['status'])).toBe('รออัปเดต')
  })
})

// The labels render wall-clock time in the viewer's timezone, so fixtures
// are built from local components (no Z suffix) — the assertions then hold
// on any runner, Bangkok or UTC.
const localIso = (hour: number, minute: number) => new Date(2026, 8, 20, hour, minute).toISOString()

describe('clock formatting', () => {
  it('renders HH:MM น. with a zero-padded hour', () => {
    expect(thaiClock(localIso(7, 5))).toBe('07:05 น.')
  })

  it('says nothing rather than inventing a time for an unparseable value', () => {
    expect(thaiClock('not-a-date')).toBe('—')
  })

  it('phrases expiry and recency the way the issue words them', () => {
    expect(shareExpiryLabel(localIso(14, 30))).toMatch(/^ใช้ได้ถึง \d{2}:\d{2} น\.$/)
    expect(sharedUpdatedLabel(localIso(14, 30))).toMatch(/^อัปเดตล่าสุด \d{2}:\d{2} น\.$/)
  })
})

describe('sharedWhereLine', () => {
  it('joins service point and floor; name alone when the floor is unknown', () => {
    expect(
      sharedWhereLine({ title: 'พบแพทย์', status: 'WAITING', servicePointName: 'อายุรกรรม', floorName: 'Ground Floor' }),
    ).toBe('อายุรกรรม · Ground Floor')
    expect(
      sharedWhereLine({ title: 'พบแพทย์', status: 'WAITING', servicePointName: 'อายุรกรรม' }),
    ).toBe('อายุรกรรม')
  })

  it('waits for a confirmed place instead of naming one', () => {
    expect(sharedWhereLine(null)).toBe('รอยืนยันจุดบริการ')
    expect(sharedWhereLine({ title: 'พบแพทย์', status: 'WAITING' })).toBe('รอยืนยันจุดบริการ')
  })
})

describe('sharedScreenModel — the six states of /shared', () => {
  it('invalid before anything else: no token means no request was ever made', () => {
    expect(sharedScreenModel({ hasToken: false, isPending: true, error: null })).toEqual({
      kind: 'invalid',
    })
  })

  it('loading while the first payload is in flight', () => {
    expect(sharedScreenModel({ hasToken: true, isPending: true, error: null })).toEqual({
      kind: 'loading',
    })
  })

  it('a 401 is the expired/revoked case; anything else is unreachable, never shown as a raw error', () => {
    expect(
      sharedScreenModel({ hasToken: true, isPending: false, error: new ApiError(401, 'no') }),
    ).toEqual({ kind: 'expired' })
    expect(
      sharedScreenModel({ hasToken: true, isPending: false, error: new ApiError(502, 'bad gateway') }),
    ).toEqual({ kind: 'unreachable' })
    expect(
      sharedScreenModel({ hasToken: true, isPending: false, error: new ApiError(0, 'ECONNREFUSED') }),
    ).toEqual({ kind: 'unreachable' })
  })

  it('DONE is the home state — the relative is told to come pick the patient up', () => {
    expect(
      sharedScreenModel({ hasToken: true, isPending: false, data: sharedJourney({ status: 'DONE' }), error: null }),
    ).toEqual({ kind: 'home' })
  })

  it('maps an in-flight visit to the current step with its Thai status', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      data: sharedJourney({
        status: 'IN_SERVICE',
        updatedAt: localIso(7, 30),
        expiresAt: localIso(11, 30),
        currentStep: {
          title: 'พบแพทย์',
          status: 'IN_SERVICE',
          servicePointName: 'อายุรกรรม',
          floorName: 'Ground Floor',
        },
      }),
      error: null,
    })
    expect(model).toEqual({
      kind: 'active',
      title: 'พบแพทย์',
      where: 'อายุรกรรม · Ground Floor',
      statusLabel: 'กำลังรับบริการ',
      updatedLabel: 'อัปเดตล่าสุด 07:30 น.',
      expiryLabel: 'ใช้ได้ถึง 11:30 น.',
    })
  })

  it('an active visit with no step yet still says something honest', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      data: sharedJourney(),
      error: null,
    })
    expect(model.kind).toBe('active')
    if (model.kind === 'active') {
      expect(model.title).toBe('ระหว่างเตรียมขั้นตอนถัดไป')
      expect(model.statusLabel).toBe('กำลังรอคิว')
    }
  })

  it('renders no domain vocabulary — the relative reads plain Thai only', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      data: sharedJourney({
        currentStep: { title: 'เจาะเลือด', status: 'WAITING', servicePointName: 'Laboratory' },
      }),
      error: null,
    })
    const text = JSON.stringify(model)
    for (const banned of ['READY', 'STARTED', 'WAITING', 'IN_SERVICE', 'CLINIC', 'ServicePoint', 'stepKey', 'VISIT']) {
      expect(text).not.toContain(banned)
    }
  })
})
