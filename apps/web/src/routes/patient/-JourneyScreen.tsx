import { useState, type ReactNode } from 'react'
import { useAuth } from '@/auth/AuthContext'
import { useLocale, useT } from '@/i18n'
import {
  AppBar,
  Button,
  Card,
  ChevronRightIcon,
  JourneyRail,
  Lead,
  Meta,
  PageTitle,
  RefPill,
  Screen,
  ScreenBody,
  ScreenDock,
  StickyActionBar,
  type JourneyStep,
} from '@/design-system'
import { ShareSheet } from '@/features/share'
import {
  journeyProgressLabel,
  servicePointLabel,
  stepTitle,
  toJourneySteps,
  useJourney,
  visitLoadErrorMessage,
  visitOutcome,
  VisitOutcomeCard,
} from '@/features/visit'

/**
 * Patient · journey home — the whole visit as one rail, one primary action.
 * No persistent nav: the journey is a linear flow, not a set of destinations.
 *
 * Live data only; the static /design reference variant is ReferenceJourney
 * in routes/-design/. The shell is exported for it to reuse.
 */
export function JourneyScreen({
  visitId,
  onNavigate,
}: {
  visitId: string
  onNavigate?: () => void
}) {
  const { identity } = useAuth()
  const { locale } = useLocale()
  const t = useT()
  const { data: journey, isPending, isError, error, refetch } = useJourney(visitId)
  const [shareOpen, setShareOpen] = useState(false)
  // Sharing needs a real patient session (ADR-0011). In demo fallback — API
  // unreachable, no token — the page stays fully usable and the button says
  // why it is off rather than breaking anything (#90 NFR).
  const shareReady = Boolean(identity?.sessionToken)

  // Early returns keep `children` undefined in the success path so the shell
  // falls back to the rail — a `false` fragment child would suppress it.
  if (isPending) {
    return (
      <JourneyShell
        visitRef={visitId}
        steps={[]}
        next={null}
        onNavigate={onNavigate}
        displayName={identity?.displayName}
      >
        <Meta>{t('journey.loading')}</Meta>
      </JourneyShell>
    )
  }

  if (isError) {
    return (
      <JourneyShell
        visitRef={visitId}
        steps={[]}
        next={null}
        onNavigate={onNavigate}
        displayName={identity?.displayName}
      >
        <Card radius="md" padding="md">
          <div className="mb-2.5 font-sans text-body-md text-ink">
            {visitLoadErrorMessage(error, locale)}
          </div>
          <Button variant="ghost" onClick={() => void refetch()}>
            {t('journey.retry')}
          </Button>
        </Card>
      </JourneyShell>
    )
  }

  const recommended = journey?.recommended
  const nextTitle = recommended ? stepTitle(recommended, locale) : undefined
  const outcome = journey ? visitOutcome(journey) : null
  const rail = journey ? toJourneySteps(journey, locale) : []
  // No route to offer once the visit ended — the projection may still carry
  // actionable steps for a CANCELLED visit, so the CTA is gated on the
  // outcome, not on `recommended` alone.
  const next =
    !outcome && recommended && nextTitle
      ? {
          value: `${nextTitle} · ${
            recommended.servicePoint ? servicePointLabel(recommended.servicePoint, locale) : ''
          }`.trim(),
          cta: t('journey.navigateCta', { title: nextTitle }),
        }
      : null
  // Nothing to follow once the visit was cancelled — the shared view would
  // have nothing honest to say.
  const shareable = outcome !== 'cancelled'

  return (
    <JourneyShell
      visitRef={journey?.visitId ?? visitId}
      steps={rail}
      next={next}
      progress={journey ? journeyProgressLabel(journey, locale) : undefined}
      notice={outcome ? <VisitOutcomeCard outcome={outcome} /> : undefined}
      onNavigate={onNavigate}
      displayName={identity?.displayName}
    >
      <JourneyRail
        steps={rail}
        labels={{ currentStep: t('rail.currentStep'), nextStep: t('rail.nextStep') }}
      />
      {shareable && (
        <div className="mt-5 flex flex-col items-start">
          <Button variant="ghost" onClick={() => setShareOpen(true)} disabled={!shareReady}>
            {t('journey.shareWithFamily')}
          </Button>
          {!shareReady && <Meta className="mt-1">{t('journey.shareDemoHint')}</Meta>}
        </div>
      )}
      {shareOpen && (
        <ShareSheet visitId={journey?.visitId ?? visitId} onClose={() => setShareOpen(false)} />
      )}
    </JourneyShell>
  )
}

export function JourneyShell({
  visitRef,
  steps,
  next,
  progress,
  notice,
  onNavigate,
  displayName,
  children,
}: {
  visitRef: string
  steps: JourneyStep[]
  /** The sticky CTA; null when nothing is actionable (visit finished). */
  next: { value: string; cta: string } | null
  /** Progress line under the lead, e.g. "Progress · 2 of 5 steps done". */
  progress?: string
  /** End-of-visit summary card above the rail; absent mid-visit. */
  notice?: ReactNode
  onNavigate?: () => void
  /** LINE display name, when signed in via LIFF; omitted on /design's static reference. */
  displayName?: string
  /** Loading/error state replaces the rail. */
  children?: ReactNode
}) {
  const t = useT()
  return (
    <Screen variant="patient">
      <AppBar wordmark="CarePath" trailing={<RefPill>{visitRef}</RefPill>} />

      <ScreenBody className="pt-1.5 pb-40">
        <PageTitle className="mt-3.5 mb-1.5">{t('journey.title')}</PageTitle>
        <Lead className="mb-7 max-w-[300px]">{t('journey.lead')}</Lead>
        {displayName && (
          <Meta className="-mt-5 mb-7">{t('journey.signedInWithLine', { name: displayName })}</Meta>
        )}
        {progress && <Meta className="-mt-5 mb-7">{progress}</Meta>}

        {notice}
        {children ?? (
          <JourneyRail
            steps={steps}
            labels={{ currentStep: t('rail.currentStep'), nextStep: t('rail.nextStep') }}
          />
        )}
      </ScreenBody>

      {/* No READY step means nothing to navigate to — the one-primary-action
          rule says show no action bar at all rather than a disabled one. */}
      {next && (
        <ScreenDock>
          <StickyActionBar label={t('journey.nextStep')} value={next.value}>
            <Button variant="primary" block onClick={onNavigate}>
              <span>{next.cta}</span>
              <ChevronRightIcon />
            </Button>
          </StickyActionBar>
        </ScreenDock>
      )}
    </Screen>
  )
}
