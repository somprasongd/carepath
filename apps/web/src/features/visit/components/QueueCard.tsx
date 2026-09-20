import { useT } from '@/i18n'
import { Card } from '@/design-system'
import type { VisitQueueStep } from '../queries'

/**
 * The next step's queue picture on the patient home screen (FR-17, #101).
 * Honest by construction: waitingAhead is a live count and always shown —
 * 0 reads as "you are next", not as a missing number — while the estimate
 * appears only when the API has real samples. null avgWaitMinutes means
 * "no data yet for this point today" and renders as words; rendering a
 * zero-minute wait there would claim a measurement that never happened,
 * and a wrong guess sends the patient walking. Informational only — ink
 * on surface, matching VisitOutcomeCard (the CTA stays the route button).
 */
export function QueueCard({ step }: { step: VisitQueueStep }) {
  const t = useT()

  return (
    <Card radius="md" padding="md" className="mb-5">
      <div className="mb-1 font-sans text-caption text-ink-muted">{t('queue.title')}</div>
      <div className="text-[18px] font-bold text-ink">
        {step.waitingAhead === 0
          ? t('queue.youAreNext')
          : t('queue.peopleAhead', { count: step.waitingAhead })}
      </div>
      <p className="m-0 mt-1 font-sans text-body-sm text-ink-muted">
        {step.estimatedWaitMinutes == null
          ? t('queue.noData')
          : t('queue.estimate', { minutes: Math.round(step.estimatedWaitMinutes) })}
      </p>
    </Card>
  )
}
