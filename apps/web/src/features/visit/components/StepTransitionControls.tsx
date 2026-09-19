import { Button, Meta } from '@/design-system'
import type { StepTargetStatus } from '../queries'
import type { StepAction } from '../staff'

/**
 * The live transition buttons for one step (#38) — the piece that rendered
 * as an honest placeholder in #37. Blue `secondary` is the staff primary;
 * the orange patient CTA is never a staff action (DESIGN.md). The error is
 * a live region so a rejected transition is announced, not just painted.
 */
export function StepTransitionControls({
  action,
  pending,
  errorText,
  onTransition,
}: {
  action: StepAction | null
  pending: boolean
  errorText: string | null
  onTransition: (to: StepTargetStatus) => void
}) {
  if (!action) return null

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        variant="secondary"
        disabled={pending}
        onClick={() => onTransition(action.to)}
      >
        {pending ? 'กำลังบันทึก…' : action.label}
      </Button>
      {errorText ? (
        <span role="alert">
          <Meta className="m-0 text-warning">{errorText}</Meta>
        </span>
      ) : null}
    </div>
  )
}
