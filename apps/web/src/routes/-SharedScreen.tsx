import { useEffect, useState } from 'react'
import { setApiShareToken } from '@/api/client'
import { useLocale, useT } from '@/i18n'
import { AppBar, Card, InfoNote, Lead, Meta, PageTitle, Screen, ScreenBody } from '@/design-system'
import { sharedScreenModel, useSharedJourney } from '@/features/share'

/**
 * Relative · treatment follow-up — the read-only relative view (#90,
 * ADR-0011).
 *
 * The token arrives in the URL fragment (`/shared#<token>`): browsers never
 * send a fragment to a server, so the credential itself never leaves the
 * device except as the bearer of the one GET this screen makes. It is read
 * once, on mount, and handed to api/client.ts — the only module allowed to
 * hold it. Nothing identifying the patient is rendered here: no name, no
 * visit ref — the payload itself is already redacted server-side.
 */
export function SharedScreen() {
  const { locale } = useLocale()
  const t = useT()
  const [token] = useState(() => window.location.hash.replace(/^#/, '').trim())

  useEffect(() => {
    setApiShareToken(token || undefined)
    return () => setApiShareToken(undefined)
  }, [token])

  const query = useSharedJourney(Boolean(token))
  // defaultError is registered as ApiError (features/visit/queries.ts), so
  // query.error arrives already narrowed.
  const model = sharedScreenModel({
    hasToken: Boolean(token),
    isPending: query.isPending,
    data: query.data,
    error: query.error,
    locale,
  })

  return (
    <Screen variant="patient">
      <AppBar wordmark="CarePath" />
      <ScreenBody className="pt-1.5 pb-10">
        {model.kind === 'loading' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <Meta>{t('shared.checkingLink')}</Meta>
          </>
        )}

        {model.kind === 'active' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <Lead className="mb-6">{t('shared.autoRefresh')}</Lead>
            <Card radius="md" padding="md">
              <div className="font-sans text-body-md font-bold text-ink">{model.title}</div>
              <Meta className="mt-1">{model.where}</Meta>
              <div className="mt-3 font-sans text-caption font-bold text-secondary">
                {model.statusLabel}
              </div>
            </Card>
            <Meta className="mt-3">
              {model.updatedLabel} · {model.expiryLabel}
            </Meta>
          </>
        )}

        {model.kind === 'home' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <InfoNote>{t('shared.done')}</InfoNote>
          </>
        )}

        {model.kind === 'expired' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <InfoNote>{t('shared.expired')}</InfoNote>
          </>
        )}

        {model.kind === 'invalid' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <InfoNote>{t('shared.invalid')}</InfoNote>
          </>
        )}

        {model.kind === 'unreachable' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">{t('shared.title')}</PageTitle>
            <InfoNote>{t('shared.unreachable')}</InfoNote>
          </>
        )}
      </ScreenBody>
    </Screen>
  )
}
