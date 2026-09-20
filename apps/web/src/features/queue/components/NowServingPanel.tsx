import { Card } from '@/design-system'
import { clockLabel } from '@/i18n/time'
import type { ReactNode } from 'react'
import { stepTitle } from '@/features/visit/journey'
import type { StationQueueEntry } from '@/features/visit/queries'
import { positionLabel } from '../labels'

export type NowServingPanelProps = {
  serving: StationQueueEntry | null
  /** Rendered under the patient — the transition controls, when serving. */
  children?: ReactNode
}

/**
 * The one thing a service point needs at a glance: who is at the counter now
 * (#102, wired to the real queue). The visit reference is the single orange
 * element on this screen. DESIGN.md reserves orange for "your route, right
 * now"; the called patient is exactly that person, and every action here
 * stays staff blue. The patient's own screen shows the same fact the moment
 * the transition lands.
 */
export function NowServingPanel({ serving, children }: NowServingPanelProps) {
  if (!serving) {
    return (
      <Card radius="md" padding="xl">
        <div className="font-sans text-h2 text-ink">ยังไม่ได้เรียกคิว</div>
        <p className="mt-1.5 mb-0 font-sans text-body-sm text-ink-muted">
          กดเรียกคิวถัดไปเพื่อเริ่มให้บริการ ผู้ป่วยจะเห็นสถานะของตัวเองเปลี่ยนบนหน้าจอทันที
        </p>
        {children}
      </Card>
    )
  }

  return (
    <Card radius="md" padding="xl">
      <div className="flex items-baseline justify-between gap-3">
        <span className="font-sans text-caption font-semibold text-ink-muted">กำลังให้บริการ</span>
        {serving.startedAt && (
          <span className="font-sans text-caption font-normal text-ink-muted">
            เรียกเมื่อ {clockLabel(serving.startedAt, 'th')}
          </span>
        )}
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-3">
        <div>
          <div className="font-sans text-caption font-normal text-ink-muted">หมายเลข VN</div>
          <div className="font-code text-display-queue text-primary break-all">
            {serving.visitId}
          </div>
        </div>
        <div className="min-w-0">
          <div className="mt-1 font-sans text-body-md font-bold text-ink">{serving.patientName}</div>
          <div className="mt-0.5 font-sans text-body-sm text-ink-muted">
            {stepTitle(serving, 'th')} · {positionLabel(serving.sequence, serving.totalSteps)}
          </div>
        </div>
      </div>

      {children ? (
        <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-line pt-4">
          {children}
        </div>
      ) : null}
    </Card>
  )
}
