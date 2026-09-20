import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import { format, messagesFor } from '@/i18n'
import type { SharedJourney } from './queries'
import {
  clockLabel,
  shareExpiryLabel,
  sharedScreenModel,
  sharedStatusLabel,
  sharedUpdatedLabel,
  sharedWhereLine,
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

describe('sharedStatusLabel — exhaustive over the enum, both locales', () => {
  it.each([
    ['WAITING', 'Waiting', 'กำลังรอคิว'],
    ['IN_SERVICE', 'Being seen', 'กำลังรับบริการ'],
    ['DONE', 'Done', 'เสร็จเรียบร้อย'],
  ] as const)('maps %s', (status, enLabel, thLabel) => {
    expect(sharedStatusLabel(status, 'en')).toBe(enLabel)
    expect(sharedStatusLabel(status, 'th')).toBe(thLabel)
  })

  it('falls back to a neutral label rather than echoing an unknown value', () => {
    expect(sharedStatusLabel('SOMETHING_NEW' as SharedJourney['status'], 'en')).toBe('Awaiting update')
    expect(sharedStatusLabel('SOMETHING_NEW' as SharedJourney['status'], 'th')).toBe(
      messagesFor('th')['shared.status.awaiting'],
    )
  })
})

// The labels render wall-clock time in the viewer's timezone, so fixtures
// are built from local components (no Z suffix) — the assertions then hold
// on any runner, Bangkok or UTC.
const localIso = (hour: number, minute: number) => new Date(2026, 8, 20, hour, minute).toISOString()

describe('clock formatting', () => {
  it('renders a zero-padded hour; Thai keeps the น. suffix, English does not', () => {
    expect(clockLabel(localIso(7, 5), 'th')).toBe('07:05 น.')
    expect(clockLabel(localIso(7, 5), 'en')).toBe('07:05')
  })

  it('says nothing rather than inventing a time for an unparseable value', () => {
    expect(clockLabel('not-a-date', 'th')).toBe('—')
    expect(clockLabel('not-a-date', 'en')).toBe('—')
  })

  it('phrases expiry and recency in each locale', () => {
    expect(shareExpiryLabel(localIso(14, 30), 'en')).toBe('Valid until 14:30')
    expect(sharedUpdatedLabel(localIso(14, 30), 'en')).toBe('Updated 14:30')
    expect(shareExpiryLabel(localIso(14, 30), 'th')).toBe('ใช้ได้ถึง 14:30 น.')
    expect(sharedUpdatedLabel(localIso(14, 30), 'th')).toBe('อัปเดตล่าสุด 14:30 น.')
  })
})

describe('sharedWhereLine', () => {
  it('joins service point and floor; name alone when the floor is unknown', () => {
    expect(
      sharedWhereLine(
        { title: 'พบแพทย์', status: 'WAITING', servicePointName: 'อายุรกรรม', floorName: 'Ground Floor' },
        'th',
      ),
    ).toBe('อายุรกรรม · Ground Floor')
    expect(
      sharedWhereLine({ title: 'พบแพทย์', status: 'WAITING', servicePointName: 'อายุรกรรม' }, 'th'),
    ).toBe('อายุรกรรม')
  })

  it('waits for a confirmed place instead of naming one, in both locales', () => {
    expect(sharedWhereLine(null, 'en')).toBe('Confirming the service point')
    expect(sharedWhereLine({ title: 'พบแพทย์', status: 'WAITING' }, 'en')).toBe(
      'Confirming the service point',
    )
    expect(sharedWhereLine(null, 'th')).toBe(messagesFor('th')['shared.awaitingServicePoint'])
  })
})

describe('sharedScreenModel — the six states of /shared', () => {
  it('invalid before anything else: no token means no request was ever made', () => {
    expect(sharedScreenModel({ hasToken: false, isPending: true, error: null, locale: 'en' })).toEqual({
      kind: 'invalid',
    })
  })

  it('loading while the first payload is in flight', () => {
    expect(sharedScreenModel({ hasToken: true, isPending: true, error: null, locale: 'en' })).toEqual({
      kind: 'loading',
    })
  })

  it('a 401 is the expired/revoked case; anything else is unreachable, never shown as a raw error', () => {
    expect(
      sharedScreenModel({ hasToken: true, isPending: false, error: new ApiError(401, 'no'), locale: 'en' }),
    ).toEqual({ kind: 'expired' })
    expect(
      sharedScreenModel({
        hasToken: true,
        isPending: false,
        error: new ApiError(502, 'bad gateway'),
        locale: 'en',
      }),
    ).toEqual({ kind: 'unreachable' })
    expect(
      sharedScreenModel({
        hasToken: true,
        isPending: false,
        error: new ApiError(0, 'ECONNREFUSED'),
        locale: 'en',
      }),
    ).toEqual({ kind: 'unreachable' })
  })

  it('DONE is the home state — the relative is told to come pick the patient up', () => {
    expect(
      sharedScreenModel({
        hasToken: true,
        isPending: false,
        data: sharedJourney({ status: 'DONE' }),
        error: null,
        locale: 'en',
      }),
    ).toEqual({ kind: 'home' })
  })

  it('maps an in-flight visit to the current step with its localized status', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      locale: 'en',
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
      statusLabel: 'Being seen',
      updatedLabel: 'Updated 07:30',
      expiryLabel: 'Valid until 11:30',
    })
  })

  it('an active visit with no step yet still says something honest', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      locale: 'en',
      data: sharedJourney(),
      error: null,
    })
    expect(model.kind).toBe('active')
    if (model.kind === 'active') {
      expect(model.title).toBe('Preparing the next step')
      expect(model.statusLabel).toBe('Waiting')
    }
  })

  it('renders no domain vocabulary — the relative reads plain language only', () => {
    const model = sharedScreenModel({
      hasToken: true,
      isPending: false,
      locale: 'en',
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

describe('interpolation parity with the shared labels', () => {
  it('composes the Thai expiry line from the catalog template', () => {
    expect(shareExpiryLabel(localIso(9, 0), 'th')).toBe(
      format(messagesFor('th'), 'shared.validUntil', { time: '09:00 น.' }),
    )
  })
})
