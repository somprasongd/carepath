import {
  AppBar,
  Button,
  ChevronRightIcon,
  JourneyRail,
  Lead,
  PageTitle,
  RefPill,
  Screen,
  ScreenDock,
  ScreenScroll,
  StickyActionBar,
} from '@/design-system'
import { journeySteps, visitRef } from '@/mocks/demo-data'

/**
 * ผู้ป่วย · หน้าแรกเส้นทาง — the whole visit as one rail, one primary action.
 * No persistent nav: the journey is a linear flow, not a set of destinations.
 */
export function JourneyScreen({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <Screen variant="patient">
      <AppBar wordmark="CarePath" trailing={<RefPill>{visitRef}</RefPill>} />

      <ScreenScroll style={{ padding: '6px var(--cp-gutter) 158px' }}>
        <div style={{ margin: '14px 0 6px' }}>
          <PageTitle>การมาโรงพยาบาลของคุณวันนี้</PageTitle>
        </div>
        <div style={{ maxWidth: 300, marginBottom: 30 }}>
          <Lead>ติดตามขั้นตอนของคุณ แล้วไปยังจุดบริการถัดไปได้จากปุ่มด้านล่าง</Lead>
        </div>

        <JourneyRail steps={journeySteps} />
      </ScreenScroll>

      <ScreenDock>
        <StickyActionBar label="ขั้นตอนถัดไป" value="รับยา · ห้องยา ชั้น 1">
          <Button variant="primary" block onClick={onNavigate}>
            <span>นำทางไปห้องยา</span>
            <ChevronRightIcon />
          </Button>
        </StickyActionBar>
      </ScreenDock>
    </Screen>
  )
}
