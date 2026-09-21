import { renderToString } from 'react-dom/server'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import { AuthContext } from '@/auth/AuthContext'
import type { AuthContextValue } from '@/auth/types'
import { JourneyScreen, JourneyShell } from '@/routes/patient/-JourneyScreen'
import { NavigateScreen } from '@/routes/patient/-NavigateScreen'
import { ShareSheet } from '@/features/share'
import { VisitOutcomeCard } from '@/features/visit'
import { LanguageToggle, LocaleProvider } from '@/i18n'
import { en } from './locales/en'

/**
 * The #93 acceptance render: every patient-facing screen rendered with
 * locale 'en' contains no Thai characters. renderToString suffices — no
 * jsdom — because the claim is about composed text, not interactions;
 * effects (queries, map drawing) never run, which is exactly what keeps
 * the loading/pending branches deterministic here.
 *
 * /shared is covered by shared-view.test.ts instead: it reads the token
 * from window.location.hash at render time.
 */

const THAI = /[\u0E00-\u0E7F]/

// The floor-plan SVG used to have to be masked out of the scan: it was
// bundled into the app, and its room labels are Thai on purpose (that is
// what the walls say). Since ADR-0015 the drawing is fetched, so it is
// simply not in the rendered output here — the screen renders its
// map-placeholder instead, whose text is the app's own and must translate
// like everything else.

const anonymous: AuthContextValue = {
  status: 'unauthenticated',
  identity: null,
  retry: () => {},
}

const queryClient = () =>
  new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })

function english(node: ReactNode): string {
  return renderToString(
    <QueryClientProvider client={queryClient()}>
      <AuthContext.Provider value={anonymous}>
        <LocaleProvider initialLocale="en">{node}</LocaleProvider>
      </AuthContext.Provider>
    </QueryClientProvider>,
  )
}

function expectNoThai(html: string) {
  // The language switch (#94) names each option in its own script — the
  // Thai endonym 'ไทย' is expected text in every locale, like any language
  // picker; it is masked before scanning, same idea as the floor-plan asset.
  const target = html.split('ไทย').join('')
  expect(target, `Thai leaked into the English render:\n${html}`).not.toMatch(THAI)
}

describe('patient screens rendered in English (#93 AC)', () => {
  it('journey screen (loading state) is English-only', () => {
    const html = english(<JourneyScreen visitId="VISIT-001" />)
    expect(html).toContain(en['journey.loading'])
    expectNoThai(html)
  })

  it('journey shell with a live rail, progress line and sticky CTA is English-only', () => {
    const html = english(
      <JourneyShell
        visitRef="VISIT-001"
        steps={[
          { id: 'reg', state: 'done', title: en['step.title.REGISTRATION'], meta: en['step.meta.completed'] },
          {
            id: 'lab',
            state: 'current',
            title: en['step.title.LAB'],
            meta: `${en['sp.LAB']} · LAB-01`,
            queue: { label: 'Queue 12', wait: 'About an 8 minute wait' },
          },
        ]}
        next={{
          value: `${en['step.title.LAB']} · ${en['sp.LAB']}`,
          cta: en['journey.navigateCta'].replace('{title}', en['step.title.LAB']),
        }}
        progress={en['journey.progress'].replace('{done}', '1').replace('{total}', '2')}
      />,
    )
    expect(html).toContain(en['rail.currentStep'])
    expectNoThai(html)
  })

  it('navigate screen is English-only in the pending, no-destination and plan states', () => {
    const pending = english(<NavigateScreen plan={{ state: 'pending' }} floorLabel={() => 'Floor 1'} />)
    expect(pending).toContain(en['navigate.pendingNotice'])
    expectNoThai(pending)

    const empty = english(<NavigateScreen plan={{ state: 'no-destination' }} floorLabel={() => 'Floor 1'} />)
    expect(empty).toContain(en['navigate.noDestinationNotice'])
    expectNoThai(empty)

    const planned = english(
      <NavigateScreen
        plan={{
          state: 'plan',
          title: `Directions to ${en['step.title.PHARMACY']}`,
          name: 'Pharmacy',
          subtitle: 'Floor 1 · Pharmacy · PHARMACY-01',
          floorId: 'I-1301',
          floorLabel: 'Floor 1',
          placeId: 'PHARMACY-01',
          servicePointCode: 'PHARMACY',
          x: 885,
          y: 190,
        }}
        floorLabel={() => 'Floor 1'}
      />,
    )
    expect(planned).toContain('Directions to Medication pickup')
    // No planSvg was passed, so this is also the map-placeholder state the
    // screen shows while the drawing is on its way (ADR-0015): the
    // destination and the directions are there, only the picture is not.
    expect(planned).toContain(en['map.loading'])
    expectNoThai(planned)
  })

  it('visit outcome cards are English-only for both outcomes', () => {
    expectNoThai(english(<VisitOutcomeCard outcome="completed" />))
    expectNoThai(english(<VisitOutcomeCard outcome="cancelled" />))
  })

  it('share sheet chrome is English-only before any link exists', () => {
    const html = english(<ShareSheet visitId="VISIT-001" onClose={() => {}} />)
    expect(html).toContain(en['share.title'])
    expectNoThai(html)
  })

  it('provider-less rendering stays Thai — the /design specimen gallery relies on it', () => {
    const html = renderToString(<JourneyShell visitRef="VISIT-001" steps={[]} next={null} />)
    expect(html).toMatch(THAI)
  })

  it('language toggle: one button naming the target locale, never a pressed switch (#94, #108 header pass)', () => {
    const html = english(<LanguageToggle />)
    expect(html).toContain('ไทย')
    expect(html).toContain(en['common.language.switch'])
    expect(html).not.toContain('>EN<')

    // Switching is the provider's job (instant, persisted — browser-verified
    // for #94); here only the markup contract is asserted.
    const thai = renderToString(
      <LocaleProvider initialLocale="th">
        <LanguageToggle />
      </LocaleProvider>,
    )
    expect(thai).toContain('>EN<')
    expect(thai).not.toContain('ไทย')

    // A flip button carries no on/off state — that grammar belongs to the
    // A+ and voice switches beside it.
    expect(html).not.toContain('aria-pressed')
    expect(thai).not.toContain('aria-pressed')
  })
})
