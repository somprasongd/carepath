import { useRef, useState } from 'react'
import { ApiError } from '@/api/client'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import {
  Button,
  Card,
  FloorPlanMap,
  InfoNote,
  Meta,
  PageTitle,
  RefPill,
  SectionTitle,
  staffSection,
  staffTitle,
} from '@/design-system'
import {
  floorLabelFor,
  planUrlFor,
  useActivateFloorPlan,
  useFloorHealth,
  useFloorPlan,
  useFloorPlans,
  useFloors,
  useUploadFloorPlan,
  type FloorPlanWarning,
  type StoredFloorPlan,
} from '@/features/floorplan'
import { StaffShell } from './-StaffShell'

/**
 * เจ้าหน้าที่ · ผังอาคาร — the floor plans the patient app draws, and, for an
 * ADMIN, the controls to replace one (FR-11 / ADR-0015).
 *
 * Plans are append-only artifacts: uploading adds a version and moves the
 * floor's pointer, so "roll back" is picking an older row rather than
 * restoring anything. That is why this screen shows a history rather than a
 * single current file.
 *
 * Warnings are shown as prominently as the plan itself. They are the map
 * model and the drawing disagreeing — a place the plan does not draw is one
 * a patient will be told is not routable — and the whole point of catching
 * them at upload is that somebody sees them here rather than in a corridor.
 */
export function FloorPlan() {
  const { identity } = useStaffAuth()
  const isAdmin = Boolean(identity?.roles?.includes('ADMIN'))

  const floors = useFloors()
  const [picked, setPicked] = useState<string | null>(null)
  // Derived, not latched: the floors query can resolve after this mounts,
  // and a useState initial value would pin to null forever.
  const floorId = picked ?? floors.data?.[0]?.floorId ?? null

  return (
    <StaffShell>
      <div className="mb-4 @7xl:mb-5">
        <PageTitle className={staffTitle}>ผังอาคาร</PageTitle>
        <Meta className="mt-1.5">
          {floors.isLoading
            ? 'กำลังโหลดรายชื่อชั้น…'
            : floors.isError
              ? 'โหลดรายชื่อชั้นไม่สำเร็จ'
              : `${floors.data?.length ?? 0} ชั้นในระบบ`}
        </Meta>
      </div>

      {(floors.data?.length ?? 0) > 1 && (
        <div className="mb-4 flex flex-wrap gap-1.5">
          {floors.data?.map((floor) => (
            <button
              key={floor.floorId}
              type="button"
              onClick={() => setPicked(floor.floorId)}
              className={`cursor-pointer rounded-full border px-3.5 py-1.5 font-sans text-caption font-bold ${
                floor.floorId === floorId
                  ? 'border-primary bg-primary-tint text-ink'
                  : 'border-line bg-surface text-ink-muted'
              }`}
            >
              {floorLabelFor(floor.floorId, 'th', floors.data ?? [])}
            </button>
          ))}
        </div>
      )}

      {floorId && <FloorPanel key={floorId} floorId={floorId} isAdmin={isAdmin} />}
    </StaffShell>
  )
}

function FloorPanel({ floorId, isAdmin }: { floorId: string; isAdmin: boolean }) {
  const floors = useFloors()
  const floor = floors.data?.find((f) => f.floorId === floorId)
  const planUrl = planUrlFor(floorId, floors.data ?? [])
  const plan = useFloorPlan(planUrl)
  // The history and the health check are ADMIN-only on the server; asking
  // as STAFF would only 403.
  const history = useFloorPlans(isAdmin ? floorId : undefined)
  const health = useFloorHealth(isAdmin ? floorId : undefined)

  return (
    <div className="grid gap-4 @7xl:max-w-[1100px] @7xl:grid-cols-[minmax(0,1fr)_320px]">
      <Card>
        <SectionTitle className={staffSection}>
          {floorLabelFor(floorId, 'th', floors.data ?? [])}
        </SectionTitle>
        <Meta className="mt-1 mb-3">
          {floor?.viewBox
            ? `พื้นที่พิกัด ${floor.viewBox} · ทุกจุดบริการและโหนดนำทางของชั้นนี้อ้างอิงพื้นที่นี้`
            : 'ยังไม่มีผังสำหรับชั้นนี้'}
        </Meta>

        <div className="h-[420px] w-full">
          {plan.data ? (
            <FloorPlanMap
              svg={plan.data}
              floorLabel={floorLabelFor(floorId, 'th', floors.data ?? [])}
              ariaLabel={`ผัง${floorLabelFor(floorId, 'th', floors.data ?? [])}`}
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center rounded-lg border border-line bg-surface">
              <span className="font-sans text-caption text-ink-muted">
                {!planUrl
                  ? 'ยังไม่มีผังที่อัปโหลดไว้'
                  : plan.isError
                    ? 'โหลดผังไม่สำเร็จ'
                    : 'กำลังโหลดผัง…'}
              </span>
            </div>
          )}
        </div>
      </Card>

      <div className="flex flex-col gap-4">
        {isAdmin ? (
          <>
            <UploadCard floorId={floorId} />
            <HealthCard
              hasActivePlan={Boolean(floor?.activePlanId)}
              warnings={health.data ?? []}
              loading={health.isLoading}
            />
            <HistoryCard
              floorId={floorId}
              plans={history.data ?? []}
              activePlanId={floor?.activePlanId}
            />
          </>
        ) : (
          <InfoNote>
            การเปลี่ยนผังอาคารทำได้เฉพาะผู้ดูแลระบบ — ผังที่เปลี่ยนมีผลกับผู้ป่วยทุกคนในอาคารทันที
          </InfoNote>
        )}
      </div>
    </div>
  )
}

function UploadCard({ floorId }: { floorId: string }) {
  const upload = useUploadFloorPlan(floorId)
  const fileRef = useRef<HTMLInputElement>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  // file.text() can reject on its own — the file was deleted or moved after
  // being picked, or the browser denies access to it — separately from
  // anything the upload mutation covers, so it needs its own error state
  // rather than leaving the rejection unhandled.
  const [readError, setReadError] = useState<string | null>(null)

  const pick = async (file: File) => {
    setFileName(file.name)
    setReadError(null)
    upload.reset()
    let svg: string
    try {
      svg = await file.text()
    } catch {
      setReadError('อ่านไฟล์ไม่สำเร็จ ลองเลือกไฟล์นี้อีกครั้ง')
      return
    }
    upload.mutate(svg)
  }

  return (
    <Card>
      <SectionTitle className={staffSection}>อัปโหลดผังใหม่</SectionTitle>
      <Meta className="mt-1 mb-3">
        ไฟล์ SVG · ต้องใช้พื้นที่พิกัดเดิมของชั้นนี้ · ไฟล์เดิมที่อัปซ้ำจะไม่สร้างเวอร์ชันใหม่
      </Meta>

      <input
        ref={fileRef}
        type="file"
        accept=".svg,image/svg+xml"
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0]
          if (file) void pick(file)
          // Clear the input so picking the same file twice still fires.
          event.target.value = ''
        }}
      />
      <Button
        variant="secondary"
        disabled={upload.isPending}
        onClick={() => fileRef.current?.click()}
      >
        {upload.isPending ? 'กำลังตรวจและอัปโหลด…' : 'เลือกไฟล์ SVG'}
      </Button>
      {fileName && <Meta className="mt-2">{fileName}</Meta>}

      {readError && (
        <p role="alert" className="mt-3 mb-0 font-sans text-caption text-warning">
          {readError}
        </p>
      )}
      {/* A refusal is the expected path, not a crash: the server names the
          rule that broke so the admin can fix the file. Showing a generic
          "upload failed" would throw away the only useful part. */}
      {upload.isError && (
        <p role="alert" className="mt-3 mb-0 font-sans text-caption text-warning">
          {upload.error instanceof ApiError ? upload.error.message : 'อัปโหลดไม่สำเร็จ'}
        </p>
      )}
      {upload.isSuccess && (
        <p className="mt-3 mb-0 font-sans text-caption text-ink">
          อัปโหลดแล้ว · ใช้ผังนี้อยู่
          {upload.data.warnings.length > 0 && ` · มีข้อสังเกต ${upload.data.warnings.length} รายการ`}
        </p>
      )}
    </Card>
  )
}

/**
 * What the active plan and the map model disagree about — checked live
 * against the model as it stands right now (ADR-0015), not against the
 * snapshot the plan was accepted with at upload time: a place added after
 * the upload shows up here even though the plan itself never changed.
 * Nothing here blocks a plan — a floor missing one room is still worth
 * showing — but each line is a place or a node a patient will be told is
 * not routable.
 */
function HealthCard({
  hasActivePlan,
  warnings,
  loading,
}: {
  hasActivePlan: boolean
  warnings: FloorPlanWarning[]
  loading: boolean
}) {
  const groups = groupWarnings(warnings)

  return (
    <Card>
      <SectionTitle className={staffSection}>ตรวจสุขภาพผัง</SectionTitle>
      {loading ? (
        <Meta className="mt-1">กำลังตรวจ…</Meta>
      ) : !hasActivePlan ? (
        <Meta className="mt-1">ยังไม่มีผังให้ตรวจ</Meta>
      ) : warnings.length === 0 ? (
        <Meta className="mt-1">ผังนี้ครอบคลุมทุกจุดบริการและโหนดนำทางของชั้น</Meta>
      ) : (
        <div className="mt-2 flex flex-col gap-3">
          {groups.map(({ code, label, refs }) => (
            <div key={code}>
              <p className="m-0 font-sans text-caption font-bold text-ink">
                {label} · {refs.length}
              </p>
              <div className="mt-1.5 flex flex-wrap gap-1">
                {refs.map((ref) => (
                  <RefPill key={ref}>{ref}</RefPill>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  )
}

function HistoryCard({
  floorId,
  plans,
  activePlanId,
}: {
  floorId: string
  plans: StoredFloorPlan[]
  activePlanId?: string
}) {
  const activate = useActivateFloorPlan(floorId)

  return (
    <Card>
      <SectionTitle className={staffSection}>ประวัติผัง</SectionTitle>
      <Meta className="mt-1 mb-3">ผังไม่เคยถูกลบ — การย้อนกลับคือเลือกเวอร์ชันเดิมมาใช้</Meta>

      <div className="flex flex-col gap-2">
        {plans.length === 0 && <Meta>ยังไม่มีประวัติ</Meta>}
        {plans.map((plan) => {
          const isActive = plan.planId === activePlanId
          return (
            <div
              key={plan.planId}
              className="flex items-baseline justify-between gap-3 rounded-lg border border-line px-3 py-2"
            >
              <div className="min-w-0">
                <p className="m-0 truncate font-sans text-caption font-bold text-ink">
                  {plan.sha256.slice(0, 8)}
                  {isActive && ' · ใช้อยู่'}
                </p>
                <p className="m-0 font-sans text-caption text-ink-muted">
                  {plan.createdBy ?? 'ติดตั้งพร้อมระบบ'} · {thaiDateTime(plan.createdAt)}
                  {plan.warnings.length > 0 && ` · ข้อสังเกต ${plan.warnings.length}`}
                </p>
              </div>
              {!isActive && (
                <Button
                  variant="ghost"
                  disabled={activate.isPending}
                  onClick={() => activate.mutate(plan.planId)}
                >
                  ใช้ผังนี้
                </Button>
              )}
            </div>
          )
        })}
      </div>

      {activate.isError && (
        <p role="alert" className="mt-3 mb-0 font-sans text-caption text-warning">
          {activate.error instanceof ApiError ? activate.error.message : 'เปลี่ยนผังไม่สำเร็จ'}
        </p>
      )}
    </Card>
  )
}

const WARNING_LABELS: Record<string, string> = {
  PLACE_MISSING: 'จุดที่ผังยังไม่ได้วาด',
  PLACE_UNKNOWN: 'จุดในผังที่ระบบยังไม่รู้จัก',
  NODE_MISSING: 'โหนดนำทางที่ผังยังไม่ได้วาด',
}

export function groupWarnings(warnings: FloorPlanWarning[]) {
  const byCode = new Map<string, string[]>()
  for (const warning of warnings) {
    byCode.set(warning.code, [...(byCode.get(warning.code) ?? []), warning.ref])
  }
  return [...byCode.entries()].map(([code, refs]) => ({
    code,
    label: WARNING_LABELS[code] ?? code,
    refs,
  }))
}

function thaiDateTime(iso: string): string {
  return new Date(iso).toLocaleString('th-TH', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
