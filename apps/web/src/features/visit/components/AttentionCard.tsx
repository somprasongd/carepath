import { Button, Card } from '@/design-system'

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
      <div className="mb-1 flex items-baseline justify-between">
        <span className="font-code text-[12px]/none font-bold text-ink">{visitRef}</span>
        <span className="font-sans text-[11px] font-normal text-ink-muted">{step}</span>
      </div>
      <div
        className={`mb-2.5 font-sans text-caption font-normal ${
          urgent ? 'text-primary' : 'text-ink-muted'
        }`}
      >
        {reason}
      </div>
      <Button variant="ghost" onClick={onAction}>
        {actionLabel}
      </Button>
    </Card>
  )
}
