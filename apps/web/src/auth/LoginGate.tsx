import type { ReactNode } from 'react'
import { useT } from '@/i18n'
import { Button, Card, Meta, Screen, ScreenBody, SectionTitle, Stack } from '@/design-system'
import { useAuth } from './AuthContext'

/**
 * Owns every non-authenticated UI state for the LIFF entry flow. Mounted once
 * above `routes/patient.tsx`'s `<Outlet/>`, above the defensive `RequireAuth`
 * gate — this is where the loading/error/redirecting copy actually lives.
 */
export function LoginGate({ children }: { children: ReactNode }) {
  const { status, error, retry } = useAuth()
  const t = useT()

  if (status === 'loading') {
    return (
      <Screen variant="patient">
        <ScreenBody className="flex flex-1 items-center justify-center text-center">
          <div role="status" aria-live="polite">
            <Meta>{t('auth.checking')}</Meta>
          </div>
        </ScreenBody>
      </Screen>
    )
  }

  if (status === 'unauthenticated') {
    return (
      <Screen variant="patient">
        <ScreenBody className="flex flex-1 items-center justify-center text-center">
          <div role="status" aria-live="polite">
            <Meta>{t('auth.redirecting')}</Meta>
          </div>
        </ScreenBody>
      </Screen>
    )
  }

  if (status === 'error') {
    return (
      <Screen variant="patient">
        <ScreenBody className="flex flex-1 items-center justify-center">
          <Card radius="md" padding="md" role="alert" className="w-full">
            <Stack>
              <SectionTitle>{t('auth.errorTitle')}</SectionTitle>
              <Meta>{error ?? t('auth.errorUnknown')}</Meta>
              <Button variant="ghost" onClick={retry}>
                {t('auth.retry')}
              </Button>
            </Stack>
          </Card>
        </ScreenBody>
      </Screen>
    )
  }

  return <>{children}</>
}
