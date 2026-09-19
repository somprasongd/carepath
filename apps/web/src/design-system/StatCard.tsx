import { Card } from './Card'

export type StatCardProps = {
  label: string
  value: string
  /** Small trailing unit inside the numeral, e.g. "ราย", "นาที". */
  unit?: string
  note?: string
  /** `attention` is the one orange KPI allowed on a staff screen. */
  tone?: 'default' | 'attention'
  /** The value is a word, not a numeral — drops to a readable size. */
  textValue?: boolean
  /** Mobile density. */
  compact?: boolean
}

export function StatCard({
  label,
  value,
  unit,
  note,
  tone = 'default',
  textValue = false,
  compact = false,
}: StatCardProps) {
  return (
    <Card
      tone={tone === 'attention' ? 'attention' : 'surface'}
      radius={compact ? 'md' : 'lg'}
      padding={compact ? 'md' : 'xl'}
      className={`cp-stat ${compact ? 'cp-stat--compact' : ''} ${
        tone === 'attention' ? 'cp-stat--attention' : ''
      }`}
    >
      <div className="cp-stat__label">{label}</div>
      <div className={`cp-stat__value ${textValue ? 'cp-stat__value--text' : ''}`}>
        {value}
        {unit && <span className="cp-stat__unit"> {unit}</span>}
      </div>
      {note && <div className="cp-stat__note">{note}</div>}
    </Card>
  )
}
