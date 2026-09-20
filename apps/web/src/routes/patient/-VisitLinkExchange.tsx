import { useEffect, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { LanguageToggle, useT } from '@/i18n'
import { LargeTextToggle } from '@/preferences'
import { Button, Card, PageTitle } from '@/design-system'
import { ApiError } from '@/api/client'
import { redeemVisitLinkToken } from '@/features/visit/queries'
import { clearVisitLinkToken, readVisitLinkToken } from '@/features/visit/visit-link'

/**
 * The slip-link exchange (#136), and the line-mode front door it becomes.
 * Arriving on /patient/journey without ?visit= but with a stashed #vt= token
 * (see visit-link.ts) redeems the token for the visit id — the URL then
 * carries only the id, and the token never does again. A 404 (unknown,
 * rotated, cancelled, past-grace link) clears the stash: the dead link must
 * not wedge every future navigation into its error screen.
 */
type ExchangeState = 'exchanging' | 'invalid' | 'error'

export function VisitLinkExchange() {
  const navigate = useNavigate()
  const t = useT()
  // No token at mount (the parent only renders this screen when one existed
  // — but a stash cleared since that read) starts at the dead-link answer.
  const [state, setState] = useState<ExchangeState>(() =>
    readVisitLinkToken() === undefined ? 'invalid' : 'exchanging',
  )
  // Bumped by the retry button — the effect re-runs and re-POSTs (redeem is
  // idempotent, and the stash survives an error for exactly this retry).
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      const token = readVisitLinkToken()
      if (!token) return
      try {
        const visitId = await redeemVisitLinkToken(token)
        if (!cancelled) {
          // The token has done its job — the claim outlives it. Clearing on
          // success too means a later bare visit never replays a link that
          // has since expired into its error screen.
          clearVisitLinkToken()
          // Replace: the fragment URL in the QR is history the patient
          // should not land back on.
          void navigate({ to: '/patient/journey', search: { visit: visitId }, replace: true })
        }
      } catch (err) {
        if (cancelled) return
        if (err instanceof ApiError && err.status === 404) {
          clearVisitLinkToken()
          setState('invalid')
        } else {
          // Network or server hiccup — the stash stays for the retry.
          setState('error')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [navigate, attempt])

  return (
    <div className="@container h-full w-full">
      <div className="flex h-full items-center justify-center overflow-y-auto bg-neutral px-gutter py-12">
        <Card radius="lg" padding="xl" className="w-full max-w-[440px]">
          <div className="mb-9 flex items-center justify-between gap-4">
            <div className="flex items-baseline gap-2">
              <span className="font-code text-[20px] font-bold text-ink">CarePath</span>
              <span className="font-sans text-body-sm text-ink-muted">{t('entry.role')}</span>
            </div>
            <div className="flex items-center gap-2">
              <LargeTextToggle />
              <LanguageToggle />
            </div>
          </div>

          {state === 'exchanging' && (
            <>
              <PageTitle className="mb-1.5">{t('entry.exchanging')}</PageTitle>
              <p className="mt-0 font-sans text-body-sm text-ink-muted">{t('entry.exchangingLead')}</p>
            </>
          )}
          {state === 'invalid' && (
            <>
              <PageTitle className="mb-1.5">{t('entry.linkInvalidTitle')}</PageTitle>
              <p className="mt-0 mb-7 font-sans text-body-sm text-ink-muted">
                {t('entry.linkInvalidLead')}
              </p>
            </>
          )}
          {state === 'error' && (
            <>
              <PageTitle className="mb-1.5">{t('entry.linkErrorTitle')}</PageTitle>
              <p className="mt-0 mb-7 font-sans text-body-sm text-ink-muted">
                {t('entry.linkErrorLead')}
              </p>
              <Button
                variant="secondary"
                block
                onClick={() => {
                  setState('exchanging')
                  setAttempt((n) => n + 1)
                }}
              >
                {t('journey.retry')}
              </Button>
            </>
          )}
        </Card>
      </div>
    </div>
  )
}

/**
 * Line mode's answer to "no visit and no token": the VN entry screen is a
 * demo-only surface (typing another patient's VN must never open their
 * journey in production), so the patient is pointed back to the slip the
 * hospital printed.
 */
export function NoVisitScreen() {
  const t = useT()

  return (
    <div className="@container h-full w-full">
      <div className="flex h-full items-center justify-center overflow-y-auto bg-neutral px-gutter py-12">
        <Card radius="lg" padding="xl" className="w-full max-w-[440px]">
          <div className="mb-9 flex items-center justify-between gap-4">
            <div className="flex items-baseline gap-2">
              <span className="font-code text-[20px] font-bold text-ink">CarePath</span>
              <span className="font-sans text-body-sm text-ink-muted">{t('entry.role')}</span>
            </div>
            <div className="flex items-center gap-2">
              <LargeTextToggle />
              <LanguageToggle />
            </div>
          </div>

          <PageTitle className="mb-1.5">{t('entry.noVisitTitle')}</PageTitle>
          <p className="mt-0 font-sans text-body-sm text-ink-muted">{t('entry.noVisitLead')}</p>
        </Card>
      </div>
    </div>
  )
}
