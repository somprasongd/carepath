import { format, messagesFor, type Locale } from '.'

/**
 * The app's one clock formatter (#94): a zero-padded 24h clock, plus the
 * locale's marker from the catalog — the Thai am/pm abbreviation in Thai,
 * nothing in English. Every rendered time goes through here — the patient
 * share surfaces, the staff console's sync stamp, the analytics snapshot —
 * so no call site hand-rolls `toLocaleTimeString('th-TH', …)` or pastes a
 * marker suffix next to the result again (the pattern this replaced).
 *
 * Manual padding instead of `toLocaleTimeString` keeps the output stable
 * across engines; the locale's only stake is the suffix, which rides the
 * catalog (`common.clock`).
 */
export function clockLabel(iso: string, locale: Locale): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const clock = `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
  return format(messagesFor(locale), 'common.clock', { time: clock })
}
