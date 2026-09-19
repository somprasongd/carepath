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

/**
 * The patient's visit as a vertical rail. The connector turns orange one step
 * ahead of "current", echoing the route line painted on the floor plan.
 */
export function JourneyRail({ steps }: { steps: JourneyStep[] }) {
  return (
    <ol className="cp-rail" style={{ listStyle: 'none', margin: 0, padding: 0 }}>
      {steps.map((step, i) => (
        <li className="cp-rail__step" key={step.id}>
          <div className="cp-rail__gutter">
            <StepNode state={step.state} />
            {i < steps.length - 1 && (
              <span className={`cp-rail__line ${connectorClass(step, steps[i + 1])}`} />
            )}
          </div>
          <div className="cp-rail__body">
            <StepBody step={step} />
          </div>
        </li>
      ))}
    </ol>
  )
}

function connectorClass(step: JourneyStep, nextStep: JourneyStep) {
  if (step.state !== 'done') return 'cp-rail__line--pending'
  return nextStep.state === 'done' ? '' : 'cp-rail__line--route'
}

function StepNode({ state }: { state: JourneyStepState }) {
  return (
    <span className={`cp-rail__node cp-rail__node--${state}`}>
      {state === 'done' && <CheckIcon />}
    </span>
  )
}

function StepBody({ step }: { step: JourneyStep }) {
  if (step.state === 'current') {
    return (
      <Card radius="ticket" padding="md">
        <div className="cp-rail__eyebrow">ขั้นตอนปัจจุบัน</div>
        <div className="cp-ticket__name">{step.title}</div>
        <div className="cp-ticket__place">{step.meta}</div>
        {step.queue && (
          <div className="cp-ticket__queue">
            <QueuePill>คิวที่ {step.queue.number}</QueuePill>
            <span className="cp-ticket__wait">{step.queue.wait}</span>
          </div>
        )}
      </Card>
    )
  }

  if (step.state === 'next') {
    return (
      <>
        <div className="cp-rail__eyebrow cp-rail__eyebrow--next">ขั้นตอนถัดไป</div>
        <div className="cp-rail__next-name">{step.title}</div>
        <div className="cp-rail__meta">{step.meta}</div>
      </>
    )
  }

  return (
    <>
      <div className={`cp-rail__title ${step.state === 'pending' ? 'cp-rail__title--muted' : ''}`}>
        {step.title}
      </div>
      <div className="cp-rail__meta">{step.meta}</div>
    </>
  )
}
