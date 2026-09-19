import {
  AppBar,
  BottomSheet,
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

/**
 * ผู้ป่วย · นำทางไปจุดบริการ — schematic map plus turn-by-turn text, so the
 * route is readable without reading the map.
 */
export function NavigateScreen({ onBack }: { onBack?: () => void }) {
  return (
    <Screen variant="patient">
      <AppBar
        onBack={onBack}
        backLabel="ย้อนกลับไปหน้าเส้นทาง"
        title="เส้นทางไปห้องยา"
        subtitle="ชั้น 1 · Pharmacy · PHARMACY-01"
      />

      <div style={{ margin: '0 var(--cp-gutter) 14px', flexShrink: 0 }}>
        <LocationBanner actionLabel="สแกนใหม่">
          ตำแหน่งล่าสุดจากการสแกน QR ที่ทางลงชั้น 1 · 2 นาทีที่แล้ว
        </LocationBanner>
      </div>

      <div style={{ flex: '1 1 auto', padding: '4px var(--cp-gutter) 0', overflow: 'hidden' }}>
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
          footer={
            <button type="button" className="cp-link">
              แจ้งเจ้าหน้าที่หากหลงทาง
            </button>
          }
        />
      </ScreenDock>
    </Screen>
  )
}
