import { Code } from '@/design-system'
import type { StationQueueEntry } from '@/features/visit/queries'
import { waitedLabel } from '../labels'

/**
 * Who is still waiting at this point, in arrival order (#102). The row
 * number is that order — the position the patient would be called in, not a
 * printed ticket. The first row is marked as next because that is the only
 * one the staff member is about to act on; the rest are context,
 * deliberately quiet. `now` is the queue payload's asOf, so waits are
 * measured against the same instant the server took the picture: "—" means
 * the timeline has no arrival fact, an honest blank rather than a guessed
 * number.
 */
export function WaitingQueueList({
  entries,
  now,
}: {
  entries: StationQueueEntry[]
  now: number
}) {
  if (entries.length === 0) {
    return (
      <p className="m-0 font-sans text-body-sm text-ink-muted">
        ไม่มีผู้ป่วยรออยู่ที่จุดบริการนี้
      </p>
    )
  }

  return (
    <ol className="m-0 flex list-none flex-col p-0">
      {entries.map((entry, i) => (
        <li
          key={`${entry.visitId}:${entry.stepKey}`}
          className="flex items-center gap-3 border-b border-line py-2.5 last:border-b-0"
        >
          <span
            className={`w-8 shrink-0 font-code text-[18px] font-bold tabular-nums ${
              i === 0 ? 'text-ink' : 'text-ink-muted'
            }`}
          >
            {i + 1}
          </span>
          <span className="min-w-0 flex-1">
            <span className="font-sans text-body-sm font-bold text-ink">{entry.patientName}</span>
            <Code className="ml-2 text-[12px]/none text-ink-muted">{entry.visitId}</Code>
            {i === 0 && (
              <span className="ml-2 font-sans text-caption font-bold text-secondary">ถัดไป</span>
            )}
          </span>
          <span className="shrink-0 font-sans text-caption font-normal text-ink-muted">
            {waitedLabel(entry.readyAt, now)}
          </span>
        </li>
      ))}
    </ol>
  )
}
