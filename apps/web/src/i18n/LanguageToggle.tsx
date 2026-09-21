import { useLocale, useT, type Locale } from '.'

/** A tap flips the locale — the button names the language it switches to. */
const TARGET: Record<Locale, Locale> = { th: 'en', en: 'th' }
const TARGET_LABEL: Record<Locale, 'common.locale.th' | 'common.locale.en'> = {
  th: 'common.locale.en',
  en: 'common.locale.th',
}

/**
 * The Thai ⇄ EN toggle (#94, FR-19; single-button form since #108's header
 * pass). One button that flips between the two locales — not a two-segment
 * switch: next to the A+ and voice switches in the patient AppBar's trailing
 * slot, a filled active segment read as "this switch is on", and at phone
 * widths the pill was eating ~110px the title needs. The visible label is
 * the target locale's endonym (common.locale.*), so a user who lands in
 * the wrong language still recognises their way back; the accessible name
 * says what a tap does. Touch target stays 44px
 * (FR-20); patient screens have no persistent nav, so this is the one
 * affordance that follows the user across the journey.
 */
export function LanguageToggle() {
  const { locale, setLocale } = useLocale()
  const t = useT()

  return (
    <button
      type="button"
      lang={TARGET[locale]}
      onClick={() => setLocale(TARGET[locale])}
      aria-label={t('common.language.switch')}
      title={t('common.language.switch')}
      className="flex min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-[10px] border border-line bg-surface px-3 font-sans text-body-sm font-semibold text-ink-muted transition-colors"
    >
      {t(TARGET_LABEL[locale])}
    </button>
  )
}
