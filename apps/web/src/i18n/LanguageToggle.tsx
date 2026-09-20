import { cn } from 'cn'
import { useLocale, useT, type Locale } from '.'

const OPTIONS: { value: Locale; labelKey: 'common.locale.th' | 'common.locale.en' }[] = [
  { value: 'th', labelKey: 'common.locale.th' },
  { value: 'en', labelKey: 'common.locale.en' },
]

/**
 * The Thai ⇄ EN switch (#94, FR-19). Two values as side-by-side buttons —
 * with exactly two languages a dropdown would only add a click. Lives in
 * the patient AppBar's trailing slot; patient screens have no persistent
 * nav by design, so this is the one affordance that follows the user
 * across the journey.
 *
 * Touch targets match the AppBar's back chevron (44px, FR-20's elderly
 * accessibility bar — a tiny corner toggle does not pass). Each option is
 * named in its own language (endonyms, per common.locale.*) in every
 * locale: a user who lands in the wrong language must still recognize
 * their way back.
 */
export function LanguageToggle() {
  const { locale, setLocale } = useLocale()
  const t = useT()

  return (
    <div
      role="group"
      aria-label={t('common.language')}
      className="flex items-center gap-0.5 rounded-[10px] border border-line bg-surface p-0.5"
    >
      {OPTIONS.map(({ value, labelKey }) => (
        <button
          key={value}
          type="button"
          lang={value}
          onClick={() => setLocale(value)}
          aria-pressed={locale === value}
          className={cn(
            'flex min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-[8px] px-3 font-sans text-body-sm font-semibold transition-colors',
            locale === value ? 'bg-primary text-surface' : 'text-ink-muted',
          )}
        >
          {t(labelKey)}
        </button>
      ))}
    </div>
  )
}
