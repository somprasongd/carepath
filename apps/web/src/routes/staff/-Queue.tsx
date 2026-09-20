import { useEffect } from 'react'
import {
  Card,
  Code,
  Meta,
  PageTitle,
  SectionTitle,
  SkipIcon,
  ZoneChip,
} from '@/design-system'
import {
  NowServingPanel,
  StationPicker,
  WaitingQueueList,
  positionLabel,
  waitedLabel,
} from '@/features/queue'
import { zoneForService } from '@/features/servicepoint/mappings'
import { useMyServicePoints, type ServicePoint } from '@/features/servicepoint/queries'
import { StepTransitionControls } from '@/features/visit/components/StepTransitionControls'
import { stepTitle } from '@/features/visit/journey'
import {
  useStationQueue,
  useTransitionStep,
  type StationQueueEntry,
} from '@/features/visit/queries'
import { transitionErrorText, type StepAction } from '@/features/visit/staff'
import { TaskHeader } from './-TaskHeader'

/**
 * The station console's own transition button — the queue screen reuses the
 * patients' screen's mechanism (#38, #102): the same endpoint, the same
 * controls, only the verb differs ("เรียกคิวถัดไป" is a READY → STARTED
 * transition on the head of the waiting list). One instance per entry, so
 * pending/error state lands on the row that caused it.
 */
function EntryTransition({
  visitId,
  stepKey,
  action,
}: {
  visitId: string
  stepKey: string
  action: StepAction
}) {
  const transition = useTransitionStep(visitId)

  return (
    <StepTransitionControls
      action={action}
      pending={transition.isPending && transition.variables?.stepKey === stepKey}
      errorText={
        transition.isError && transition.variables?.stepKey === stepKey
          ? transitionErrorText(transition.error.status, transition.error.message)
          : null
      }
      onTransition={(to) => transition.mutate({ stepKey, to })}
    />
  )
}

/** A serving entry beyond the latest call — same facts, one row instead of a stage. */
function ServingRow({ entry }: { entry: StationQueueEntry }) {
  return (
    <Card radius="md" padding="lg">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <div className="min-w-0">
          <span className="font-sans text-body-sm font-bold text-ink">{entry.patientName}</span>
          <Code className="ml-2 text-[12px]/none text-ink-muted">{entry.visitId}</Code>
          <div className="mt-0.5 font-sans text-caption text-ink-muted">
            {stepTitle(entry, 'th')} · {positionLabel(entry.sequence, entry.totalSteps)}
          </div>
        </div>
        <EntryTransition
          visitId={entry.visitId}
          stepKey={entry.stepKey}
          action={{ to: 'COMPLETED', label: 'ทำเสร็จแล้ว' }}
        />
      </div>
    </Card>
  )
}

/**
 * เจ้าหน้าที่ · เรียกคิว (FR-15/FR-16, US-14/US-15, #102) — one dedicated
 * workstation view per service point, on the real queue: the visit's
 * actionable steps at this point, arrival-ordered, called through the same
 * staff transition the patient screen reflects. No tickets exist in the
 * model — the arrival order and the VN are the honest identifiers.
 */
export function Queue({
  servicePointId,
  onServicePointChange,
}: {
  /** The ?sp= from the URL — the point this console is working. */
  servicePointId: string | undefined
  onServicePointChange: (servicePointId: string) => void
}) {
  const points = useMyServicePoints()
  const assigned = points.data ?? []
  const selected: ServicePoint | null =
    assigned.find((p) => p.id === servicePointId) ?? assigned[0] ?? null

  // The URL follows the picker (and self-heals a stale ?sp=), so a
  // workstation's screen is bookmarkable and shareable as-is.
  useEffect(() => {
    if (selected && selected.id !== servicePointId) onServicePointChange(selected.id)
  }, [selected, servicePointId, onServicePointChange])

  const queue = useStationQueue(selected?.id ?? '', { enabled: selected !== null })
  // Waits are measured against the server's asOf — the same instant the
  // queue picture was taken — not this render's clock.
  const asOf = queue.data ? Date.parse(queue.data.asOf) : 0

  return (
    <div className="@container flex h-dvh flex-col bg-neutral">
      <TaskHeader
        role="เจ้าหน้าที่จุดบริการ"
        station={
          selected && (
            <span className="flex min-w-0 flex-wrap items-center gap-3.5">
              <StationPicker points={assigned} value={selected.id} onChange={onServicePointChange} />
              <span className="hidden h-5 w-px shrink-0 bg-line @sm:inline-block" />
              <ZoneChip zone={zoneForService(selected.code)}>
                {selected.name} · {selected.code}
                {selected.place?.floor ? ` · ${selected.place.floor.name}` : ''}
              </ZoneChip>
            </span>
          )
        }
      />

      <main className="flex-1 overflow-y-auto px-gutter py-8 @7xl:px-gutter-desktop">
        {points.isPending ? (
          <p className="m-auto font-sans text-body-md text-ink-muted">กำลังโหลดจุดบริการ…</p>
        ) : points.isError ? (
          <Card radius="md" padding="xl">
            <PageTitle className="mb-1.5 text-[16px]">โหลดจุดบริการไม่สำเร็จ</PageTitle>
            <Meta>สถานะ {points.error.status} — ตรวจว่า API ทำงานอยู่แล้วลองใหม่</Meta>
          </Card>
        ) : assigned.length === 0 ? (
          <Card radius="md" padding="xl">
            <PageTitle className="mb-1.5 text-[16px]">ยังไม่มีจุดบริการที่ได้รับมอบหมาย</PageTitle>
            <Meta>
              บัญชีนี้ยังไม่ได้รับมอบหมายจุดบริการ — ติดต่อผู้ดูแลระบบเพื่อมอบหมายก่อนเริ่มใช้หน้าจอคิว
            </Meta>
          </Card>
        ) : queue.isPending ? (
          <p className="m-auto font-sans text-body-md text-ink-muted">กำลังโหลดคิว…</p>
        ) : queue.isError ? (
          <Card radius="md" padding="xl">
            <PageTitle className="mb-1.5 text-[16px]">โหลดคิวไม่สำเร็จ</PageTitle>
            <Meta>
              สถานะ {queue.error.status}
              {queue.error.status === 403
                ? ' — บัญชีนี้ไม่ได้รับมอบหมายจุดบริการที่เลือก ลองเลือกจุดอื่น'
                : ' — ตรวจว่า API ทำงานอยู่แล้วลองใหม่'}
            </Meta>
          </Card>
        ) : queue.data ? (
          <div className="mx-auto grid max-w-[1240px] grid-cols-1 items-start gap-6 @6xl:grid-cols-[1fr_360px]">
            <div className="flex flex-col gap-6">
              <NowServingPanel serving={queue.data.serving[0] ?? null}>
                {queue.data.serving[0] && (
                  <EntryTransition
                    visitId={queue.data.serving[0].visitId}
                    stepKey={queue.data.serving[0].stepKey}
                    action={{ to: 'COMPLETED', label: 'ทำเสร็จแล้ว' }}
                  />
                )}
              </NowServingPanel>

              {queue.data.serving.slice(1).map((entry) => (
                <ServingRow key={`${entry.visitId}:${entry.stepKey}`} entry={entry} />
              ))}

              {queue.data.waiting[0] ? (
                <Card radius="md" padding="lg">
                  <SectionTitle>คิวถัดไป</SectionTitle>
                  <div className="mb-3.5 mt-2 flex flex-wrap items-baseline gap-x-3 gap-y-1">
                    <span className="font-sans text-body-md font-bold text-ink">
                      {queue.data.waiting[0].patientName}
                    </span>
                    <Code className="text-[12px]/none text-ink-muted">
                      {queue.data.waiting[0].visitId}
                    </Code>
                    <span className="font-sans text-caption text-ink-muted">
                      {waitedLabel(queue.data.waiting[0].readyAt, asOf)}
                    </span>
                  </div>
                  <EntryTransition
                    visitId={queue.data.waiting[0].visitId}
                    stepKey={queue.data.waiting[0].stepKey}
                    action={{ to: 'STARTED', label: 'เรียกคิวถัดไป' }}
                  />
                </Card>
              ) : (
                <Card radius="md" padding="xl">
                  <PageTitle className="mb-1.5 text-[16px]">ไม่มีผู้ป่วยรอเพิ่มเติม</PageTitle>
                  <Meta className="flex items-center gap-1.5">
                    <SkipIcon size={14} /> ให้บริการครบทุกคิวของช่วงเวลานี้แล้ว
                  </Meta>
                </Card>
              )}
            </div>

            <Card radius="md" padding="lg">
              <SectionTitle>คิวที่กำลังรอ</SectionTitle>
              <Meta className="-mt-2 mb-3.5">{queue.data.waiting.length} คน</Meta>
              <WaitingQueueList entries={queue.data.waiting} now={asOf} />
            </Card>
          </div>
        ) : null}
      </main>
    </div>
  )
}
