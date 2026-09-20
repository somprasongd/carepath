import { Card } from './Card'

export type StatCardProps = {
  label: string
  value: string
  /** Small trailing unit inside the numeral, e.g. a persons or minutes word. */
  unit?: string
  note?: string
  /** `attention` is the one orange KPI allowed on a staff screen. */
  tone?: 'default' | 'attention'
  /** The value is a word, not a numeral — drops to a readable size. */
  textValue?: boolean
  /** Mobile density. */
  compact?: boolean
}

function valueSize(textValue: boolean, compact: boolean) {
  if (compact) return textValue ? 'text-[16px]' : 'text-[24px]'
  return textValue ? 'text-[26px]' : 'text-display-stat'
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
  const attention = tone === 'attention'

  return (
    <Card
      tone={attention ? 'attention' : 'surface'}
      radius={compact ? 'md' : 'lg'}
      padding={compact ? 'md' : 'xl'}
    >
      <div
        className={`font-sans text-caption ${compact ? 'mb-1.5' : 'mb-2.5'} ${
          attention ? 'text-warning' : 'text-ink-muted'
        }`}
      >
        {label}
      </div>
      <div
        className={`font-code font-bold ${valueSize(textValue, compact)} ${
          attention ? 'text-primary' : 'text-ink'
        }`}
      >
        {value}
        {unit && (
          <span
            className={`font-sans text-caption ${attention ? 'text-primary' : 'text-ink-muted'}`}
          >
            {' '}
            {unit}
          </span>
        )}
      </div>
      {note && (
        <div
          className={`mt-1.5 font-sans text-caption ${
            attention ? 'text-warning' : 'text-ink-muted'
          }`}
        >
          {note}
        </div>
      )}
    </Card>
  )
}
