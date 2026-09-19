import { Code } from '@/design-system'

export type WaitingTicket = {
  ticket: string
  visitRef: string
  waited: string
}

/**
 * Who is still waiting at this point. The first row is marked as next because
 * that is the only one the staff member is about to act on; the rest are
 * context, deliberately quiet.
 */
export function WaitingQueueList({ tickets }: { tickets: WaitingTicket[] }) {
  if (tickets.length === 0) {
    return (
      <p className="m-0 font-sans text-body-sm text-ink-muted">
        ไม่มีผู้ป่วยรออยู่ที่จุดบริการนี้
      </p>
    )
  }

  return (
    <ol className="m-0 flex list-none flex-col p-0">
      {tickets.map((t, i) => (
        <li
          key={t.ticket}
          className="flex items-center gap-3 border-b border-line py-2.5 last:border-b-0"
        >
          <span
            className={`w-8 shrink-0 font-code text-[18px] font-bold tabular-nums ${
              i === 0 ? 'text-ink' : 'text-ink-muted'
            }`}
          >
            {t.ticket}
          </span>
          <span className="min-w-0 flex-1">
            <Code className="text-[12px]/none text-ink-muted">{t.visitRef}</Code>
            {i === 0 && (
              <span className="ml-2 font-sans text-caption font-bold text-secondary">ถัดไป</span>
            )}
          </span>
          <span className="shrink-0 font-sans text-caption font-normal text-ink-muted">
            รอ {t.waited}
          </span>
        </li>
      ))}
    </ol>
  )
}
