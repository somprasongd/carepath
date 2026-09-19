import { Button } from './Button'
import { Card } from './Card'

export type AttentionCardProps = {
  /** Visit reference, e.g. "V-2210". */
  visitRef: string
  /** The step the patient is stuck on, in plain Thai. */
  step: string
  reason: string
  /** Orange reason text is reserved for "we have lost this patient". */
  urgent?: boolean
  actionLabel: string
  onAction?: () => void
}

/** A patient the front desk should do something about. */
export function AttentionCard({
  visitRef,
  step,
  reason,
  urgent = false,
  actionLabel,
  onAction,
}: AttentionCardProps) {
  return (
    <Card radius="md" padding="md">
      <div className="cp-attention__head">
        <span className="cp-sp-row__code">{visitRef}</span>
        <span className="cp-sp-row__name">{step}</span>
      </div>
      <div className={`cp-attention__reason ${urgent ? 'cp-attention__reason--urgent' : ''}`}>
        {reason}
      </div>
      <Button variant="ghost" onClick={onAction}>
        {actionLabel}
      </Button>
    </Card>
  )
}
