import {
  AppBar,
  Divider,
  FloorPlanMap,
  InfoNote,
  LinkButton,
  Screen,
  ScreenDock,
} from '@/design-system'
import {
  floorPlanFor,
  type DestinationPlan,
  type NavigatePlan,
} from '@/features/floorplan'

/**
 * Static reference for /design: the pharmacy on the real ground-floor plan —
 * the same asset and highlight as the live screen, no fabricated route.
 */
const REFERENCE_PLAN: DestinationPlan = {
  title: 'เส้นทางไปรับยา',
  name: 'Pharmacy',
  subtitle: 'ชั้น 1 · Pharmacy · PHARMACY-01',
  floorId: 'I-1301',
  floorLabel: 'ชั้น 1',
  placeId: 'PHARMACY-01',
  x: 885,
  y: 190,
}

/**
 * ผู้ป่วย · นำทางไปจุดบริการ — the real floor-plan asset with the
 * destination room highlighted and named, focused so it reads at phone
 * width. Turn-by-turn text and the route line wait for the navigation API
 * (#28/#29); nothing about the route is invented here (DESIGN.md).
 * Omitting `plan` renders the static reference screens on /design.
 */
export function NavigateScreen({
  plan,
  onBack,
}: {
  plan?: NavigatePlan
  onBack?: () => void
}) {
  const resolved = plan ?? { state: 'plan' as const, ...REFERENCE_PLAN }

  return (
    <Screen variant="patient">
      <AppBar
        onBack={onBack}
        backLabel="ย้อนกลับไปหน้าเส้นทาง"
        title={appBarTitle(resolved)}
        subtitle={resolved.state === 'plan' ? resolved.subtitle : undefined}
      />

      {resolved.state === 'plan' && floorPlanFor(resolved.floorId) ? (
        <>
          <div className="flex-1 overflow-hidden px-gutter pt-1 pb-2">
            <FloorPlanMap
              svg={floorPlanFor(resolved.floorId) ?? ''}
              floorLabel={resolved.floorLabel}
              destination={{
                placeId: resolved.placeId,
                name: resolved.name,
                x: resolved.x,
                y: resolved.y,
              }}
            />
          </div>

          <ScreenDock>
            <DestinationPanel plan={resolved} />
          </ScreenDock>
        </>
      ) : (
        <div className="px-gutter pt-5">
          <InfoNote>{noticeFor(resolved)}</InfoNote>
        </div>
      )}
    </Screen>
  )
}

function appBarTitle(plan: NavigatePlan): string {
  switch (plan.state) {
    case 'pending':
      return 'กำลังโหลดจุดหมาย…'
    case 'no-destination':
      return 'จุดบริการของคุณ'
    default:
      return plan.title
  }
}

function noticeFor(plan: NavigatePlan): string {
  switch (plan.state) {
    case 'pending':
      return 'กำลังโหลดจุดหมายของคุณ…'
    case 'no-destination':
      return 'ยังไม่มีจุดบริการถัดไปในการมาโรงพยาบาลครั้งนี้'
    default:
      return 'ระบบยังไม่รองรับเส้นทางในอาคารสำหรับจุดบริการนี้ — โปรดถามเจ้าหน้าที่ที่จุดรับลงทะเบียน'
  }
}

/**
 * The sheet under the map carries the destination facts patients need —
 * name, floor, place — instead of the demo's invented walking time. It
 * keeps the BottomSheet's anatomy (handle, summary, divider) so the real
 * turn-by-turn steps can drop straight in with #28/#29.
 */
function DestinationPanel({ plan }: { plan: DestinationPlan }) {
  return (
    <div className="flex flex-col gap-3.5 rounded-t-xl bg-surface px-gutter pt-3 pb-6 shadow-sheet">
      <div className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
      <div className="flex items-baseline justify-between gap-3">
        <div className="min-w-0">
          <div className="font-sans text-body-md font-bold text-ink">{plan.name}</div>
          <div className="font-sans text-body-sm text-ink-muted">{plan.subtitle}</div>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-neutral px-2.5 py-1 font-sans text-caption font-bold text-ink-muted">
          {plan.floorLabel}
        </span>
      </div>
      <Divider />
      <LinkButton>แจ้งเจ้าหน้าที่หากหลงทาง</LinkButton>
      <p className="m-0 font-sans text-caption text-ink-muted">
        เส้นทางเดินแบบทีละจุดจะแสดงที่นี่เมื่อระบบนำทางในอาคารพร้อมใช้งาน
      </p>
    </div>
  )
}
