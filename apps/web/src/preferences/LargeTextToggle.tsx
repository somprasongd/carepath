import { useState } from 'react'
import { cn } from 'cn'
import { useT } from '@/i18n'
import { LARGE_TEXT_STORAGE_KEY, applyLargeText, readStoredFlag, storeFlag } from './preferences'

/**
 * The large-text switch (#99 part 2, FR-20/US-09): one button that toggles
 * the reading-size CSS tokens document-wide via
 * `html.carepath-large-text` — the text grows everywhere at once, no
 * per-screen re-render. Lives beside LanguageToggle in the patient AppBar's
 * trailing slot; the choice persists like the locale (#94, same mechanism).
 *
 * "A+" reads in every language, so it is the visible label; the accessible
 * name is localized. Touch target matches LanguageToggle's 44px.
 */
export function LargeTextToggle() {
  const t = useT()
  const [on, setOn] = useState(() => readStoredFlag(LARGE_TEXT_STORAGE_KEY) ?? false)

  return (
    <button
      type="button"
      aria-pressed={on}
      aria-label={t('prefs.largeText')}
      title={t('prefs.largeText')}
      onClick={() => {
        const next = !on
        setOn(next)
        storeFlag(LARGE_TEXT_STORAGE_KEY, next)
        applyLargeText(next)
      }}
      className={cn(
        'flex min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-[10px] border px-3 font-sans text-body-sm font-semibold transition-colors',
        on ? 'border-primary bg-primary text-surface' : 'border-line bg-surface text-ink-muted',
      )}
    >
      A+
    </button>
  )
}
