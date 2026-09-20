import { useEffect, useRef, useState } from 'react'
import { messagesFor, useLocale, useT, type Locale } from '@/i18n'
import { Button, Divider, Meta } from '@/design-system'
import { useCreateShareLink, useStopSharing } from '../queries'
import { shareExpiryLabel } from '../shared-view'

/**
 * Share progress with family — the share bottom sheet (#90). Opened from the
 * journey screen's ghost button, never the sticky bar: the one orange action
 * on a patient screen stays the navigate CTA (apps/web AGENTS.md design
 * rules).
 *
 * The sheet mints a link on open, shows it with a copy affordance and its
 * expiry in the patient's language, and can stop sharing — revoking every
 * active link at once, then minting a fresh one so the cap (5 active) never
 * blocks the next share. The token is composed into a URL here and nowhere
 * else; it travels in the fragment after /shared so no server or proxy ever
 * sees it (ADR-0011 §5).
 */
export function ShareSheet({ visitId, onClose }: { visitId: string; onClose: () => void }) {
  const create = useCreateShareLink(visitId)
  const stop = useStopSharing(visitId)
  const { locale } = useLocale()
  const t = useT()
  const [copied, setCopied] = useState(false)
  const [stopped, setStopped] = useState(false)
  const minted = useRef(false)

  useEffect(() => {
    if (minted.current) return
    minted.current = true
    create.mutate()
  }, [create])

  const stopSharing = () => {
    stop.mutate(undefined, {
      onSuccess: () => {
        setStopped(true)
        // A stopped sheet mints a new link immediately — the patient asked to
        // stop, not to close the door on sharing again.
        create.mutate()
      },
    })
  }

  const link = create.data ? `${window.location.origin}/shared#${create.data.token}` : null

  const copy = async () => {
    if (!link) return
    if (await copyText(link)) {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col justify-end"
      role="dialog"
      aria-modal="true"
      aria-label={t('share.title')}
    >
      {/* The scrim is a button so tapping anywhere above the sheet closes it. */}
      <button
        type="button"
        aria-label={t('share.close')}
        className="flex-1 cursor-default bg-ink/45"
        onClick={onClose}
      />
      <div className="rounded-t-xl bg-surface px-gutter pt-3 pb-6 shadow-sheet">
        <div className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
        <div className="mt-3.5 font-sans text-body-md font-bold text-ink">{t('share.title')}</div>
        <Meta className="mt-1">{t('share.privacyNote')}</Meta>

        <Divider />

        {create.isPending && <Meta>{t('share.creating')}</Meta>}

        {create.isError && (
          <div className="flex flex-col gap-2.5">
            <Meta>{createErrorMessage(create.error, locale)}</Meta>
            {create.error.status === 409 ? (
              <Button variant="secondary" onClick={stopSharing} disabled={stop.isPending}>
                {stop.isPending ? t('share.stopping') : t('share.stopOldFirst')}
              </Button>
            ) : (
              <Button variant="ghost" onClick={() => create.mutate()}>
                {t('share.recreate')}
              </Button>
            )}
          </div>
        )}

        {link && create.data && (
          <div className="flex flex-col gap-3">
            <p className="m-0 break-all rounded-lg border border-line bg-neutral px-3 py-2.5 font-sans text-body-sm text-ink">
              {link}
            </p>
            <div className="flex flex-wrap items-center gap-2.5">
              <Button variant="secondary" onClick={() => void copy()}>
                {copied ? t('share.copied') : t('share.copyLink')}
              </Button>
              <Button variant="ghost" onClick={stopSharing} disabled={stop.isPending}>
                {stop.isPending ? t('share.stopping') : t('share.stop')}
              </Button>
            </div>
            <Meta>{shareExpiryLabel(create.data.expiresAt, locale)}</Meta>
            {stopped && !stop.isPending && (
              <Meta className="text-ink">{t('share.stoppedNote')}</Meta>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function createErrorMessage(error: { status: number }, locale: Locale): string {
  const t = messagesFor(locale)
  if (error.status === 409) return t['share.error.tooMany']
  if (error.status === 401) return t['share.error.signInFirst']
  return t['share.error.generic']
}

/** Clipboard with the legacy fallback for webviews without the async API. */
async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // Fall through to the legacy path.
  }
  try {
    const area = document.createElement('textarea')
    area.value = text
    area.style.position = 'fixed'
    area.style.opacity = '0'
    document.body.appendChild(area)
    area.select()
    const ok = document.execCommand('copy')
    area.remove()
    return ok
  } catch {
    return false
  }
}
