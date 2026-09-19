import { journeySteps, visitRef } from '@/mocks/demo-data'
import { JourneyShell } from '../patient/-JourneyScreen'

/**
 * The journey screen wearing its static /design data — the catalogue must
 * never depend on the API, so it feeds the shell directly instead of going
 * through useVisit.
 */
export function ReferenceJourney({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <JourneyShell
      visitRef={visitRef}
      steps={journeySteps}
      next={{ value: 'รับยา · ห้องยา ชั้น 1', cta: 'นำทางไปห้องยา' }}
      onNavigate={onNavigate}
    />
  )
}
