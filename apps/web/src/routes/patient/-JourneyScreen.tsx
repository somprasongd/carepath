import type { ReactNode } from 'react'
import { useAuth } from '@/auth/AuthContext'
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
import {
  thaiStepTitle,
  toJourneySteps,
  useJourney,
  visitLoadErrorMessage,
  visitOutcome,
  journeyProgressLabel,
  VisitOutcomeCard,
} from '@/features/visit'

/**
 * ผู้ป่วย · หน้าแรกเส้นทาง — the whole visit as one rail, one primary action.
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
  const { data: journey, isPending, isError, error, refetch } = useJourney(visitId)

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
        <Meta>กำลังโหลดขั้นตอนของคุณ…</Meta>
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
            {visitLoadErrorMessage(error)}
          </div>
          <Button variant="ghost" onClick={() => void refetch()}>
            ลองใหม่
          </Button>
        </Card>
      </JourneyShell>
    )
  }

  const recommended = journey?.recommended
  const nextTitle = recommended ? thaiStepTitle(recommended) : undefined
  const outcome = journey ? visitOutcome(journey) : null
  // No route to offer once the visit ended — the projection may still carry
  // actionable steps for a CANCELLED visit, so the CTA is gated on the
  // outcome, not on `recommended` alone.
  const next =
    !outcome && recommended && nextTitle
      ? {
          value: `${nextTitle} · ${recommended.servicePoint?.name ?? ''}`.trim(),
          cta: `นำทางไป${nextTitle}`,
        }
      : null

  return (
    <JourneyShell
      visitRef={journey?.visitId ?? visitId}
      steps={journey ? toJourneySteps(journey) : []}
      next={next}
      progress={journey ? journeyProgressLabel(journey) : undefined}
      notice={outcome ? <VisitOutcomeCard outcome={outcome} /> : undefined}
      onNavigate={onNavigate}
      displayName={identity?.displayName}
    />
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
  /** Progress line under the lead, e.g. "ความคืบหน้า · เสร็จแล้ว 2 จาก 5 ขั้นตอน". */
  progress?: string
  /** End-of-visit summary card above the rail; absent mid-visit. */
  notice?: ReactNode
  onNavigate?: () => void
  /** LINE display name, when signed in via LIFF; omitted on /design's static reference. */
  displayName?: string
  /** Loading/error state replaces the rail. */
  children?: ReactNode
}) {
  return (
    <Screen variant="patient">
      <AppBar wordmark="CarePath" trailing={<RefPill>{visitRef}</RefPill>} />

      <ScreenBody className="pt-1.5 pb-40">
        <PageTitle className="mt-3.5 mb-1.5">การมาโรงพยาบาลของคุณวันนี้</PageTitle>
        <Lead className="mb-7 max-w-[300px]">
          ติดตามขั้นตอนของคุณ แล้วไปยังจุดบริการถัดไปได้จากปุ่มด้านล่าง
        </Lead>
        {displayName && <Meta className="-mt-5 mb-7">เข้าสู่ระบบด้วยไลน์ · {displayName}</Meta>}
        {progress && <Meta className="-mt-5 mb-7">{progress}</Meta>}

        {notice}
        {children ?? <JourneyRail steps={steps} />}
      </ScreenBody>

      {/* No READY step means nothing to navigate to — the one-primary-action
          rule says show no action bar at all rather than a disabled one. */}
      {next && (
        <ScreenDock>
          <StickyActionBar label="ขั้นตอนถัดไป" value={next.value}>
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
