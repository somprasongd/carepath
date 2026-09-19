import { journeySteps, journeyStepsDone, visitRef } from '@/mocks/demo-data'
import { VisitOutcomeCard } from '@/features/visit'
import { JourneyShell } from '../patient/-JourneyScreen'

/**
 * The journey screen wearing its static /design data — the catalogue must
 * never depend on the API, so it feeds the shell directly instead of going
 * through useJourney.
 */
export function ReferenceJourney({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <JourneyShell
      visitRef={visitRef}
      steps={journeySteps}
      progress="ความคืบหน้า · เสร็จแล้ว 2 จาก 5 ขั้นตอน"
      next={{ value: 'รับยา · ห้องยา ชั้น 1', cta: 'นำทางไปห้องยา' }}
      onNavigate={onNavigate}
    />
  )
}

/** #34's completed state, same shell: outcome card, all-done rail, no CTA. */
export function ReferenceJourneyDone() {
  return (
    <JourneyShell
      visitRef={visitRef}
      steps={journeyStepsDone}
      progress="ความคืบหน้า · เสร็จแล้ว 5 จาก 5 ขั้นตอน"
      notice={<VisitOutcomeCard outcome="completed" />}
      next={null}
    />
  )
}
