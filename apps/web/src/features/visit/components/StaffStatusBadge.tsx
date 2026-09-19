import { Badge } from '@/design-system'
import { stepStatusLabel, stepTone, visitStatusLabel, visitTone } from '../staff'

/**
 * A status badge for the staff monitor. Unlike the patient screens, staff
 * sees translated statuses on the badge tones the design system already
 * owns; 'ready' borrows the ZoneChip grammar (caller supplies the fill).
 */
export function StaffStatusBadge({ status, kind }: { status: string; kind: 'step' | 'visit' }) {
  const tone = kind === 'step' ? stepTone(status) : visitTone(status)
  const label = kind === 'step' ? stepStatusLabel(status) : visitStatusLabel(status)

  if (tone === 'ready') {
    return (
      <Badge variant="zone" className="bg-primary-tint">
        {label}
      </Badge>
    )
  }
  return <Badge variant={tone}>{label}</Badge>
}
