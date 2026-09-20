import type { ReactNode } from 'react'
import { Card, Code, Meta, RefPill, SectionTitle } from '@/design-system'
import { stepTitle } from '../journey'
import type { Journey, JourneyStep } from '../queries'
import { StaffStatusBadge } from './StaffStatusBadge'
import { syncedAtLabel } from '../staff'

/**
 * The selected visit's journey (#37 AC2–AC3): every ordered step with its
 * status, the resolved current/next line, and a per-step action slot — the
 * transition controls (#38) render into it without this layout knowing them.
 */
export function StaffVisitDetail({
  journey,
  onBack,
  renderStepActions,
}: {
  journey: Journey
  onBack?: () => void
  renderStepActions?: (step: JourneyStep) => ReactNode
}) {
  return (
    <Card radius="md" padding="lg" className="@7xl:p-5">
      {onBack && (
        <button
          type="button"
          onClick={onBack}
          className="mb-3 cursor-pointer border-0 bg-none p-0 font-sans text-caption font-bold text-secondary @6xl:hidden"
        >
          ← ผู้ป่วยทั้งหมด
        </button>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <RefPill>{journey.visitId}</RefPill>
          <StaffStatusBadge status={journey.status} kind="visit" />
        </div>
        <Meta className="m-0">
          {journey.patientRef} · sync {syncedAtLabel(journey.syncedAt)} น.
        </Meta>
      </div>

      <div className="mt-3 flex flex-col gap-1 rounded-md border border-line bg-neutral p-3 @7xl:flex-row @7xl:gap-6">
        <PositionLine
          label="กำลังทำอยู่"
          step={journey.steps.find((step) => step.status === 'STARTED')}
        />
        <PositionLine label="แนะนำตอนนี้" step={journey.recommended} />
      </div>

      <SectionTitle className="mt-4 mb-2 text-[14px]/[1.3] font-bold @7xl:mb-3 @7xl:text-h2">
        ขั้นตอนทั้งหมด ({journey.steps.length})
      </SectionTitle>

      <ol className="m-0 flex list-none flex-col gap-2 p-0">
        {journey.steps.map((step) => {
          const actions = renderStepActions?.(step)
          return (
            <li
              key={step.stepKey}
              className="flex flex-col rounded-sm border border-line bg-surface px-3 py-2.5"
            >
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                <Code className="text-ink-muted">{step.sequence}</Code>
                <div className="min-w-0 flex-1">
                  <div className="font-sans text-body-sm font-semibold text-ink">
                    {stepTitle(step, 'th')}
                  </div>
                  <Meta className="m-0">
                    {step.stepKey}
                    {step.servicePoint ? ` · ${step.servicePoint.name}` : ' · ไม่มีจุดบริการที่ผูกไว้'}
                  </Meta>
                </div>
                <StaffStatusBadge status={step.status} kind="step" />
              </div>
              {actions ? <div className="mt-2">{actions}</div> : null}
            </li>
          )
        })}
      </ol>
    </Card>
  )
}

function PositionLine({ label, step }: { label: string; step?: JourneyStep | null }) {
  return (
    <Meta className="m-0">
      <span className="text-ink-muted">{label}: </span>
      <span className="font-semibold text-ink">{step ? stepTitle(step, 'th') : '—'}</span>
    </Meta>
  )
}
