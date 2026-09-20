import { useEffect, useState } from 'react'
import { setApiShareToken } from '@/api/client'
import { AppBar, Card, InfoNote, Lead, Meta, PageTitle, Screen, ScreenBody } from '@/design-system'
import { sharedScreenModel, useSharedJourney } from '@/features/share'

/**
 * ญาติ · ติดตามการรักษา — the read-only relative view (#90, ADR-0011).
 *
 * The token arrives in the URL fragment (`/shared#<token>`): browsers never
 * send a fragment to a server, so the credential itself never leaves the
 * device except as the bearer of the one GET this screen makes. It is read
 * once, on mount, and handed to api/client.ts — the only module allowed to
 * hold it. Nothing identifying the patient is rendered here: no name, no
 * visit ref — the payload itself is already redacted server-side.
 */
export function SharedScreen() {
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
  })

  return (
    <Screen variant="patient">
      <AppBar wordmark="CarePath" />
      <ScreenBody className="pt-1.5 pb-10">
        {model.kind === 'loading' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <Meta>กำลังตรวจสอบลิงก์…</Meta>
          </>
        )}

        {model.kind === 'active' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <Lead className="mb-6">หน้านี้อัปเดตอัตโนมัติทุก 15 วินาที</Lead>
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
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <InfoNote>ผู้ป่วยเสร็จการรักษาแล้ว — กลับบ้านได้เลย</InfoNote>
          </>
        )}

        {model.kind === 'expired' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <InfoNote>ลิงก์นี้หมดอายุแล้ว — ขอลิงก์ใหม่จากผู้ป่วยได้เลย</InfoNote>
          </>
        )}

        {model.kind === 'invalid' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <InfoNote>เปิดหน้านี้ด้วยลิงก์ที่ผู้ป่วยแชร์มาเท่านั้น — ขอลิงก์จากผู้ป่วยได้เลย</InfoNote>
          </>
        )}

        {model.kind === 'unreachable' && (
          <>
            <PageTitle className="mt-3.5 mb-1.5">ติดตามการรักษา</PageTitle>
            <InfoNote>เชื่อมต่อไม่สำเร็จ — ลองปิดแล้วเปิดหน้านี้ใหม่อีกครั้ง</InfoNote>
          </>
        )}
      </ScreenBody>
    </Screen>
  )
}
