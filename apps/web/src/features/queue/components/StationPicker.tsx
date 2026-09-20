import type { ServicePoint } from '@/features/servicepoint/queries'

/**
 * The workstation's station switcher (FR-15, #102): a picker over the
 * signed-in staff member's assigned points — never free text, so a typo
 * can't point the console at a station that doesn't exist. Admins see the
 * full list here; the id, not the label, is what rides the URL (?sp=).
 */
export function StationPicker({
  points,
  value,
  onChange,
}: {
  points: ServicePoint[]
  value: string | null
  onChange: (servicePointId: string) => void
}) {
  return (
    <label className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 font-sans text-caption font-normal text-ink-muted">จุดบริการ</span>
      <select
        className="min-w-0 cursor-pointer rounded-full border border-line bg-surface px-3.5 py-1.5 font-sans text-body-sm text-ink focus:outline-2 focus:outline-offset-1 focus:outline-secondary"
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={points.length === 0}
      >
        {points.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name} · {p.code}
          </option>
        ))}
      </select>
    </label>
  )
}
