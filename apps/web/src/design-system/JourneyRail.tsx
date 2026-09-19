import { Card } from './Card'
import { CheckIcon } from './Icon'
import { QueuePill } from './StatusBadge'

export type JourneyStepState = 'done' | 'current' | 'next' | 'pending'

export type JourneyStep = {
  id: string
  state: JourneyStepState
  /** Plain-Thai step name — never a domain enum or `ServicePoint`. */
  title: string
  /** Place + code line, e.g. "ห้องเจาะเลือด · ชั้น 2 · LAB-01". */
  meta: string
  /** Current step only: the ticket card's queue number and wait estimate. */
  queue?: { number: string; wait: string }
}

const nodeClass: Record<JourneyStepState, string> = {
  done: 'size-6 bg-ink text-surface',
  /* The one glowing marker on the screen — it is where the patient is now. */
  current: 'size-7 -ml-0.5 bg-primary shadow-[0_0_0_6px_var(--primary-tint)]',
  next: 'size-6 border-2 border-primary bg-surface',
  pending: 'size-6 border-2 border-line bg-surface',
}

/**
 * The patient's visit as a vertical rail. The connector turns orange one step
 * ahead of "current", echoing the route line painted on the floor plan.
 */
export function JourneyRail({ steps }: { steps: JourneyStep[] }) {
  return (
    <ol className="m-0 flex list-none flex-col p-0">
      {steps.map((step, i) => (
        <li className="flex gap-[14px]" key={step.id}>
          <div className="flex w-6 shrink-0 flex-col items-center">
            <StepNode state={step.state} />
            {i < steps.length - 1 && <Connector step={step} nextStep={steps[i + 1]} />}
          </div>
          <div className="min-w-0 flex-1 pb-5">
            <StepBody step={step} />
          </div>
        </li>
      ))}
    </ol>
  )
}

function Connector({ step, nextStep }: { step: JourneyStep; nextStep: JourneyStep }) {
  if (step.state !== 'done') {
    return <span className="mt-1.5 min-h-[22px] flex-1 border-l-2 border-dashed border-line" />
  }

  return nextStep.state === 'done' ? (
    <span className="min-h-[22px] w-0.5 flex-1 bg-ink" />
  ) : (
    <span className="mt-1 min-h-[18px] w-[3px] flex-1 bg-primary" />
  )
}

function StepNode({ state }: { state: JourneyStepState }) {
  return (
    <span
      className={`flex shrink-0 items-center justify-center rounded-full ${nodeClass[state]}`}
    >
      {state === 'done' && <CheckIcon />}
    </span>
  )
}

function StepBody({ step }: { step: JourneyStep }) {
  if (step.state === 'current') {
    return (
      <Card radius="ticket" padding="md">
        <div className="mb-1 font-sans text-caption text-ink-muted">ขั้นตอนปัจจุบัน</div>
        <div className="mb-1 text-[18px] font-bold text-ink">{step.title}</div>
        <div className="mb-3 font-sans text-body-sm text-ink-muted">{step.meta}</div>
        {step.queue && (
          <div className="flex items-center gap-2.5">
            <QueuePill>คิวที่ {step.queue.number}</QueuePill>
            <span className="font-sans text-body-sm text-ink-muted">{step.queue.wait}</span>
          </div>
        )}
      </Card>
    )
  }

  if (step.state === 'next') {
    return (
      <>
        <div className="mb-0.5 font-sans text-caption font-bold text-primary">ขั้นตอนถัดไป</div>
        <div className="font-sans text-h2 text-ink">{step.title}</div>
        <StepMeta>{step.meta}</StepMeta>
      </>
    )
  }

  return (
    <>
      <div
        className={`font-sans text-[15px]/[1.6] font-semibold ${
          step.state === 'pending' ? 'text-ink-muted' : 'text-ink'
        }`}
      >
        {step.title}
      </div>
      <StepMeta>{step.meta}</StepMeta>
    </>
  )
}

function StepMeta({ children }: { children: string }) {
  return <div className="mt-0.5 font-sans text-body-sm text-ink-muted">{children}</div>
}
