import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button, InfoNote, Meta, PageTitle, staffTitle } from '@/design-system'
import {
  journeyQueryOptions,
  useStaffVisits,
} from '@/features/visit'
import { StaffVisitDetail } from '@/features/visit/components/StaffVisitDetail'
import { StaffVisitList } from '@/features/visit/components/StaffVisitList'
import type { JourneyStep } from '@/features/visit'
import { StaffShell } from './-StaffShell'

/**
 * ผู้ป่วยวันนี้ · the staff visit monitor (#37): every projected visit on
 * the left, the selected visit's full journey on the right. Reads the
 * CarePath projection via the staff list + journey endpoints — on a phone
 * the list and the detail swap instead of sitting side by side.
 */
export function Patients() {
  const visits = useStaffVisits()
  const journeys = visits.data ?? []
  const [explicitId, setExplicitId] = useState<string | null>(null)
  const selectedId = explicitId ?? journeys[0]?.visitId ?? null

  const detail = useQuery({
    ...journeyQueryOptions(selectedId ?? ''),
    enabled: selectedId !== null,
  })

  const refreshing = visits.isFetching || detail.isFetching

  return (
    <StaffShell>
      <div className="flex items-end justify-between gap-3">
        <div>
          <PageTitle className={staffTitle}>ผู้ป่วยวันนี้</PageTitle>
          <Meta className="mt-1.5">
            {visits.isLoading
              ? 'กำลังโหลดรายการ…'
              : visits.isError
                ? 'โหลดรายการไม่สำเร็จ'
                : `${journeys.length} การมารับบริการในระบบ`}
          </Meta>
        </div>
        <Button
          variant="ghost"
          onClick={() => {
            void visits.refetch()
            if (selectedId) void detail.refetch()
          }}
          disabled={refreshing}
        >
          {refreshing ? 'กำลังรีเฟรช…' : 'รีเฟรช'}
        </Button>
      </div>

      <div className="mt-4 @7xl:mt-5">
        {visits.isError ? (
          <InfoNote>
            โหลดรายการผู้ป่วยไม่สำเร็จ (สถานะ {visits.error.status}) — ตรวจว่า API
            เปิดอยู่แล้วกดรีเฟรชอีกครั้ง
          </InfoNote>
        ) : journeys.length === 0 ? (
          <InfoNote>
            ยังไม่มีการมารับบริการในระบบ — เปิด visit ใน Mock HIS console แล้วรอสักครู่
            ให้ ingest ดึงเข้า projection
          </InfoNote>
        ) : (
          <div className="grid grid-cols-1 items-start gap-4 @6xl:grid-cols-[minmax(300px,360px)_1fr] @7xl:gap-6">
            <div className={explicitId ? 'hidden @6xl:block' : 'block'}>
              <StaffVisitList
                journeys={journeys}
                selectedId={selectedId}
                onSelect={setExplicitId}
              />
            </div>
            <div className={explicitId ? 'block' : 'hidden @6xl:block'}>
              {detail.data ? (
                <StaffVisitDetail
                  journey={detail.data}
                  onBack={() => setExplicitId(null)}
                  renderStepActions={stepControlsPlaceholder}
                />
              ) : detail.isError ? (
                <InfoNote>
                  อ่าน journey ของ visit นี้ไม่สำเร็จ (สถานะ {detail.error.status})
                  {detail.error.status === 404 ? ' — projection ยังไม่ถูกสร้าง' : ''}
                </InfoNote>
              ) : (
                <Meta>กำลังโหลด journey…</Meta>
              )}
            </div>
          </div>
        )}
      </div>
    </StaffShell>
  )
}

/**
 * #37 AC4: the per-step control slot exists and is reachable; the live
 * start/complete buttons land with #38 — an honest placeholder, not a fake
 * button.
 */
function stepControlsPlaceholder(step: JourneyStep) {
  if (step.status !== 'READY' && step.status !== 'STARTED') return null
  return (
    <Meta className="m-0 w-fit rounded-sm bg-zone-support px-2 py-1">
      ปุ่มเปลี่ยนสถานะเปิดใช้งานในขั้นถัดไป (#38)
    </Meta>
  )
}
