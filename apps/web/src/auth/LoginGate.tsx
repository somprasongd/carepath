import type { ReactNode } from 'react'
import { Button, Card, Meta, Screen, ScreenBody, SectionTitle, Stack } from '@/design-system'
import { useAuth } from './AuthContext'

/**
 * Owns every non-authenticated UI state for the LIFF entry flow. Mounted once
 * above `routes/patient.tsx`'s `<Outlet/>`, above the defensive `RequireAuth`
 * gate — this is where the loading/error/redirecting copy actually lives.
 */
export function LoginGate({ children }: { children: ReactNode }) {
  const { status, error, retry } = useAuth()

  if (status === 'loading') {
    return (
      <Screen variant="patient">
        <ScreenBody className="flex flex-1 items-center justify-center text-center">
          <div role="status" aria-live="polite">
            <Meta>กำลังตรวจสอบสิทธิ์การเข้าใช้งาน...</Meta>
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
            <Meta>กำลังนำท่านไปยังหน้าเข้าสู่ระบบ LINE...</Meta>
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
              <SectionTitle>เข้าสู่ระบบไม่สำเร็จ</SectionTitle>
              <Meta>{error ?? 'เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ'}</Meta>
              <Button variant="ghost" onClick={retry}>
                ลองอีกครั้ง
              </Button>
            </Stack>
          </Card>
        </ScreenBody>
      </Screen>
    )
  }

  return <>{children}</>
}
