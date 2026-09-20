import { useState } from 'react'
import { cn } from 'cn'
import { Button, InfoNote, Meta, PageTitle, staffTitle } from '@/design-system'
import {
  stepAction,
  transitionErrorText,
  useStaffVisits,
  useTransitionStep,
} from '@/features/visit'
import { StepTransitionControls } from '@/features/visit/components/StepTransitionControls'
import { StaffVisitDetail } from '@/features/visit/components/StaffVisitDetail'
import { StaffVisitList } from '@/features/visit/components/StaffVisitList'
import { StaffShell } from './-StaffShell'

/**
 * ผู้ป่วยวันนี้ · the staff visit monitor (#37): every projected visit on
 * the left, the selected visit's full journey on the right. Both sides read
 * the staff list endpoint, which returns the same per-visit shape as the
 * single-journey read — since #96 the patient journey read takes the
 * patient's own session, so the staff detail must not ride it (#96). On a
 * phone the list and the detail swap instead of sitting side by side. The
 * per-step controls (#38) command transitions through the application API.
 */
export function Patients() {
  const visits = useStaffVisits()
  const journeys = visits.data ?? []
  const [explicitId, setExplicitId] = useState<string | null>(null)
  const selectedId = explicitId ?? journeys[0]?.visitId ?? null
  const detail = { data: journeys.find((j) => j.visitId === selectedId) ?? null }

  const transition = useTransitionStep(selectedId ?? '')

  const refreshing = visits.isFetching

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
          <div className="grid grid-cols-1 items-start gap-4 @6xl:grid-cols-[minmax(300px,360px)_1fr] @7xl:grid-cols-[minmax(460px,560px)_1fr] @7xl:gap-6">
            <div className={cn('min-w-0', explicitId ? 'hidden @6xl:block' : 'block')}>
              <StaffVisitList
                journeys={journeys}
                selectedId={selectedId}
                onSelect={setExplicitId}
              />
            </div>
            <div className={cn('min-w-0', explicitId ? 'block' : 'hidden @6xl:block')}>
              {detail.data ? (
                <StaffVisitDetail
                  journey={detail.data}
                  onBack={() => setExplicitId(null)}
                  renderStepActions={(step) => (
                    <StepTransitionControls
                      action={stepAction(step.status)}
                      pending={
                        transition.isPending && transition.variables?.stepKey === step.stepKey
                      }
                      errorText={
                        transition.isError && transition.variables?.stepKey === step.stepKey
                          ? transitionErrorText(
                              transition.error.status,
                              transition.error.message,
                            )
                          : null
                      }
                      onTransition={(to) =>
                        transition.mutate({ stepKey: step.stepKey, to })
                      }
                    />
                  )}
                />
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
