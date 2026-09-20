import { useT } from '@/i18n'
import { Meta } from '@/design-system'
import { useSetVisitNotificationPref, useVisitNotificationPref } from '../notifications'

/**
 * The queue-proximity notification toggle (FR-21, #104): one pressed-style
 * button like the app bar's toggles, driven by the server's per-visit
 * preference rather than local storage — the backend sweep reads the same
 * row. Rendered only while the visit is still going: a finished visit has
 * no queue to be near.
 */
export function NotificationToggle({ visitId }: { visitId: string }) {
  const t = useT()
  const { data: pref, isPending, isError } = useVisitNotificationPref(visitId)
  const setPref = useSetVisitNotificationPref(visitId)

  if (isPending || isError) {
    // The toggle is an enhancement, not a gate: unreadable state simply
    // renders nothing rather than a broken control. A failed flip keeps
    // the button at the last known state.
    return null
  }

  const on = pref.enabled
  return (
    <div className="flex flex-col items-start">
      <button
        type="button"
        aria-pressed={on}
        disabled={setPref.isPending}
        onClick={() => setPref.mutate(!on)}
        className={
          'flex min-h-11 cursor-pointer items-center justify-center rounded-[10px] border px-4 font-sans text-body-sm font-semibold transition-colors disabled:cursor-wait ' +
          (on
            ? 'border-primary bg-primary text-surface'
            : 'border-line bg-surface text-ink-muted')
        }
      >
        {on ? t('journey.notifyOn') : t('journey.notifyOff')}
      </button>
      <Meta className="mt-1.5">{t('journey.notifyToggleLabel')}</Meta>
    </div>
  )
}
