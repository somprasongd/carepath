import {
  AppBar,
  BottomSheet,
  InfoNote,
  LinkButton,
  LocationBanner,
  SchematicMap,
  Screen,
  ScreenDock,
} from '@/design-system'
import {
  currentLocation,
  floor1Corridor,
  floor1Rooms,
  pharmacyRoute,
  pharmacyRouteEnd,
  walkingSteps,
} from '@/mocks/demo-data'

export type NavigateDestination = {
  /** AppBar title, e.g. "เส้นทางไปเจาะเลือด". */
  title: string
  /** Place line under it, e.g. "Laboratory · LAB-01". */
  subtitle: string
  /** When set and not PHARMACY-01 there is no schematic plan yet — see below. */
  placeId?: string
}

/**
 * ผู้ป่วย · นำทางไปจุดบริการ — schematic map plus turn-by-turn text, so the
 * route is readable without reading the map.
 *
 * The only schematic plan that exists so far is the pharmacy one inherited
 * from the reference screens; /api/v1/navigation/route is still reserved in
 * the contract. Until it lands, a destination with another placeId gets the
 * honest "ยังไม่รองรับเส้นทาง" state instead of a wrong map (DESIGN.md).
 * Omitting `destination` renders the static reference screens on /design.
 */
export function NavigateScreen({
  destination,
  onBack,
}: {
  destination?: NavigateDestination
  onBack?: () => void
}) {
  const hasPlan = destination?.placeId === undefined || destination.placeId === 'PHARMACY-01'
  const fallback = { title: 'เส้นทางไปห้องยา', subtitle: 'ชั้น 1 · Pharmacy · PHARMACY-01' }

  return (
    <Screen variant="patient">
      <AppBar
        onBack={onBack}
        backLabel="ย้อนกลับไปหน้าเส้นทาง"
        title={destination?.title ?? fallback.title}
        subtitle={destination?.subtitle ?? fallback.subtitle}
      />

      {hasPlan ? (
        <>
          <div className="mx-gutter mb-3.5 shrink-0">
            <LocationBanner actionLabel="สแกนใหม่">
              ตำแหน่งล่าสุดจากการสแกน QR ที่ทางลงชั้น 1 · 2 นาทีที่แล้ว
            </LocationBanner>
          </div>

          <div className="flex-1 overflow-hidden px-gutter pt-1">
            <SchematicMap
              rooms={floor1Rooms}
              corridor={floor1Corridor}
              route={pharmacyRoute}
              routeEnd={pharmacyRouteEnd}
              you={currentLocation}
            />
          </div>

          <ScreenDock>
            <BottomSheet
              primary="3 นาที"
              secondary="· 65 เมตร"
              steps={walkingSteps}
              footer={<LinkButton>แจ้งเจ้าหน้าที่หากหลงทาง</LinkButton>}
            />
          </ScreenDock>
        </>
      ) : (
        <div className="px-gutter pt-5">
          <InfoNote>
            ระบบยังไม่รองรับเส้นทางในอาคารสำหรับจุดบริการนี้ — โปรดถามเจ้าหน้าที่ที่จุดรับลงทะเบียน
          </InfoNote>
        </div>
      )}
    </Screen>
  )
}
