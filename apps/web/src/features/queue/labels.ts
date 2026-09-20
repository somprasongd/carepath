/**
 * Staff-pinned label helpers for the station console (#102). Like
 * features/visit/staff.ts, this surface pins locale 'th' by design
 * (ADR-0012 §2) — the strings here are staff vocabulary, not catalog keys.
 */

/** "รอ 12 นาที" since the step became READY — the arrival that queued them. */
export function waitedLabel(readyAt: string | null | undefined, now: number): string {
  if (!readyAt) return '—'
  const minutes = Math.floor((now - Date.parse(readyAt)) / 60_000)
  if (minutes < 1) return 'รอไม่ถึงนาที'
  return `รอ ${minutes} นาที`
}

/** "ขั้นที่ 3 จาก 5" — where this step sits in its visit's plan. */
export function positionLabel(sequence: number, totalSteps: number): string {
  return `ขั้นที่ ${sequence} จาก ${totalSteps}`
}
