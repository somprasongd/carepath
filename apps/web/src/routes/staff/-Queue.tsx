import { useState } from 'react'
import { Card, Meta, PageTitle, PlusIcon, SectionTitle, SkipIcon, ZoneChip } from '@/design-system'
import { NowServingPanel, WaitingQueueList, type ServingStage } from '@/features/queue'
import { station, unplannedStepOptions } from '@/mocks/demo-data'
import { TaskHeader } from './-TaskHeader'

const stageOrder: ServingStage[] = ['called', 'arrived', 'in-progress']

/** The full name a visit ref stands in for, keyed for this screen's demo ticket run. */
const patientsByVisit: Record<string, string> = {
  'V-2384': 'นายสมชาย ใจดี',
  'V-2385': 'นางสมศรี มีสุข',
  'V-2390': 'นายวิชัย รักเรียน',
  'V-2401': 'นางสาวพิมพ์ใจ แสงทอง',
  'V-2404': 'นายประเสริฐ ยิ้มแย้ม',
  'V-2412': 'นางบุญมี ศรีสุข',
}

const ticketRun = [
  { ticket: '12', visitRef: 'V-2384', waited: '' },
  { ticket: '13', visitRef: 'V-2385', waited: '4 นาที' },
  { ticket: '14', visitRef: 'V-2390', waited: '6 นาที' },
  { ticket: '15', visitRef: 'V-2401', waited: '9 นาที' },
  { ticket: '16', visitRef: 'V-2404', waited: '11 นาที' },
  { ticket: '17', visitRef: 'V-2412', waited: '14 นาที' },
]

/**
 * เจ้าหน้าที่ · เรียกคิว (FR-15, FR-16, US-14, US-15) — one dedicated
 * workstation view per service point. Calling next advances a pointer through
 * a fixed demo ticket run; no queue-call endpoint exists yet.
 */
export function Queue() {
  const [queueIndex, setQueueIndex] = useState(0)
  const [stageIndex, setStageIndex] = useState(0)
  const [showInsert, setShowInsert] = useState(false)

  const hasServing = queueIndex < ticketRun.length
  const current = hasServing ? ticketRun[queueIndex] : null
  const stage = stageOrder[stageIndex]

  const advance = () => {
    if (stageIndex < stageOrder.length - 1) {
      setStageIndex(stageIndex + 1)
      return
    }
    setStageIndex(0)
    setShowInsert(false)
    setQueueIndex((i) => i + 1)
  }

  const skip = () => {
    setStageIndex(0)
    setShowInsert(false)
    setQueueIndex((i) => i + 1)
  }

  const waiting = ticketRun.slice(queueIndex + 1)

  return (
    <div className="@container flex h-dvh flex-col bg-neutral">
      <TaskHeader
        role="เจ้าหน้าที่จุดบริการ"
        station={
          <span className="flex min-w-0 items-center gap-3.5">
            <span className="hidden h-5 w-px shrink-0 bg-line @sm:inline-block" />
            <ZoneChip zone={station.zone}>
              {station.name} · {station.code} · {station.floor}
            </ZoneChip>
          </span>
        }
      />

      <main className="flex-1 overflow-y-auto px-gutter py-8 @7xl:px-gutter-desktop">
        <div className="mx-auto grid max-w-[1240px] grid-cols-1 items-start gap-6 @6xl:grid-cols-[1fr_360px]">
          {hasServing && current ? (
            <NowServingPanel
              serving={{
                ticket: current.ticket,
                visitRef: current.visitRef,
                patient: patientsByVisit[current.visitRef],
                step: 'เจาะเลือด',
                position: 'ขั้นที่ 3 จาก 5',
                calledAt: '09:41 น.',
              }}
              stage={stage}
              onAdvance={advance}
              onSkip={skip}
              onInsertStep={() => setShowInsert((v) => !v)}
            >
              {showInsert && (
                <div className="mt-4 rounded-md border border-line bg-neutral p-4">
                  <div className="mb-2.5 font-sans text-caption font-semibold text-ink">
                    เพิ่มขั้นตอนที่ไม่ได้วางแผนไว้ให้ผู้ป่วยรายนี้
                  </div>
                  <div className="flex flex-col gap-2">
                    {unplannedStepOptions.map((opt) => (
                      <div
                        key={opt.code}
                        className="flex items-center justify-between rounded-sm border border-line bg-surface px-3 py-2.5"
                      >
                        <div>
                          <span className="font-code text-[12px] font-bold text-ink">
                            {opt.code}
                          </span>
                          <span className="ml-2 font-sans text-body-sm text-ink">{opt.title}</span>
                          <div className="mt-0.5 font-sans text-[11px] text-ink-muted">
                            แทรกก่อนขั้นตอน "{opt.insertBefore}"
                          </div>
                        </div>
                        <button
                          type="button"
                          className="cursor-pointer border-0 bg-none p-0 font-sans text-caption font-bold text-secondary"
                        >
                          <PlusIcon size={12} /> เพิ่ม
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </NowServingPanel>
          ) : (
            <Card radius="md" padding="xl">
              <PageTitle className="mb-1.5 text-[16px]">ไม่มีผู้ป่วยรอเพิ่มเติม</PageTitle>
              <Meta className="flex items-center gap-1.5">
                <SkipIcon size={14} /> ให้บริการครบทุกคิวของช่วงเวลานี้แล้ว
              </Meta>
            </Card>
          )}

          <Card radius="md" padding="lg">
            <SectionTitle>คิวที่กำลังรอ</SectionTitle>
            <Meta className="-mt-2 mb-3.5">{waiting.length} คน</Meta>
            <WaitingQueueList tickets={waiting} />
          </Card>
        </div>
      </main>
    </div>
  )
}
