import type { ReactNode } from 'react'
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
  serviceCodeOf,
  thaiStepTitle,
  toJourneySteps,
  useVisit,
  visitLoadErrorMessage,
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
  const { data: visit, isPending, isError, error, refetch } = useVisit(visitId)

  // Early returns keep `children` undefined in the success path so the shell
  // falls back to the rail — a `false` fragment child would suppress it.
  if (isPending) {
    return (
      <JourneyShell visitRef={visitId} steps={[]} next={null} onNavigate={onNavigate}>
        <Meta>กำลังโหลดขั้นตอนของคุณ…</Meta>
      </JourneyShell>
    )
  }

  if (isError) {
    return (
      <JourneyShell visitRef={visitId} steps={[]} next={null} onNavigate={onNavigate}>
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

  const next = visit?.next
  const nextTitle = visit && next ? thaiStepTitle(serviceCodeOf(visit, next.sequence)) : undefined

  return (
    <JourneyShell
      visitRef={visit?.visitId ?? visitId}
      steps={visit ? toJourneySteps(visit) : []}
      next={
        next && nextTitle
          ? {
              value: `${nextTitle} · ${next.servicePoint?.name ?? next.servicePoint?.code ?? ''}`.trim(),
              cta: `นำทางไป${nextTitle}`,
            }
          : null
      }
      onNavigate={onNavigate}
    />
  )
}

export function JourneyShell({
  visitRef,
  steps,
  next,
  onNavigate,
  children,
}: {
  visitRef: string
  steps: JourneyStep[]
  /** The sticky CTA; null when nothing is actionable (visit finished). */
  next: { value: string; cta: string } | null
  onNavigate?: () => void
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
