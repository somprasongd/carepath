import { Button, Card, CallIcon, PlusIcon, SkipIcon } from '@/design-system'
import type { ReactNode } from 'react'

/** FR-15's status vocabulary, in the order a ticket moves through it. */
export const servingStages = ['called', 'arrived', 'in-progress'] as const
export type ServingStage = (typeof servingStages)[number]

const stageLabel: Record<ServingStage, string> = {
  called: 'เรียกแล้ว',
  arrived: 'มาถึง',
  'in-progress': 'กำลังให้บริการ',
}

export type NowServing = {
  ticket: string
  visitRef: string
  patient: string
  /** The step this point is performing, in plain Thai. */
  step: string
  /** Where it sits in the visit, e.g. "ขั้นที่ 3 จาก 5". */
  position: string
  calledAt: string
}

export type NowServingPanelProps = {
  serving: NowServing | null
  stage: ServingStage
  onAdvance: () => void
  onSkip: () => void
  onInsertStep: () => void
  /** Rendered under the actions — the unplanned-step picker, when open. */
  children?: ReactNode
}

/**
 * The one thing a service point needs at a glance: who is at the counter now.
 *
 * The ticket numeral is the single orange element on this screen. DESIGN.md
 * reserves orange for "your route, right now"; the called ticket is exactly
 * that patient, and every action here stays staff blue. If that reading is
 * wrong, the fix is to make the numeral `ink` — not to spread orange around.
 */
export function NowServingPanel({
  serving,
  stage,
  onAdvance,
  onSkip,
  onInsertStep,
  children,
}: NowServingPanelProps) {
  if (!serving) {
    return (
      <Card radius="md" padding="xl">
        <div className="font-sans text-h2 text-ink">ยังไม่ได้เรียกคิว</div>
        <p className="mt-1.5 mb-0 font-sans text-body-sm text-ink-muted">
          กดเรียกคิวถัดไปเพื่อเริ่มให้บริการ ผู้ป่วยจะเห็นหมายเลขคิวของตัวเองบนหน้าจอทันที
        </p>
      </Card>
    )
  }

  return (
    <Card radius="md" padding="xl">
      <div className="flex items-baseline justify-between gap-3">
        <span className="font-sans text-caption font-semibold text-ink-muted">กำลังให้บริการ</span>
        <span className="font-sans text-caption font-normal text-ink-muted">
          เรียกเมื่อ {serving.calledAt}
        </span>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-3">
        <div>
          <div className="font-sans text-caption font-normal text-ink-muted">คิวที่</div>
          <div className="font-code text-display-queue text-primary tabular-nums">
            {serving.ticket}
          </div>
        </div>
        <div className="min-w-0">
          <div className="font-code text-label-code text-ink">{serving.visitRef}</div>
          <div className="mt-1 font-sans text-body-md font-bold text-ink">{serving.patient}</div>
          <div className="mt-0.5 font-sans text-body-sm text-ink-muted">
            {serving.step} · {serving.position}
          </div>
        </div>
      </div>

      <StageTrack stage={stage} />

      <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-line pt-4">
        {stage === 'in-progress' ? (
          <Button variant="secondary" onClick={onAdvance}>
            เสร็จสิ้น
          </Button>
        ) : (
          <Button variant="secondary" onClick={onAdvance}>
            <CallIcon />
            {stage === 'called' ? 'ผู้ป่วยมาถึงแล้ว' : 'เริ่มให้บริการ'}
          </Button>
        )}
        <Button variant="ghost" onClick={onSkip}>
          <SkipIcon />
          ข้ามคิวนี้
        </Button>
        <Button variant="ghost" onClick={onInsertStep}>
          <PlusIcon />
          ส่งต่อไปขั้นตอนเพิ่มเติม
        </Button>
      </div>

      {children}
    </Card>
  )
}

/** Where this ticket is in FR-15's arrived → in progress → completed run. */
function StageTrack({ stage }: { stage: ServingStage }) {
  const reached = servingStages.indexOf(stage)

  return (
    <ol className="m-0 mt-4 flex list-none gap-1 p-0">
      {servingStages.map((s, i) => (
        <li key={s} className="flex-1">
          <span
            className={`block h-1 rounded-full ${i <= reached ? 'bg-secondary' : 'bg-line'}`}
            aria-hidden="true"
          />
          <span
            className={`mt-1.5 block font-sans text-caption ${
              i === reached ? 'font-bold text-secondary' : 'font-normal text-ink-muted'
            }`}
          >
            {stageLabel[s]}
          </span>
        </li>
      ))}
    </ol>
  )
}
