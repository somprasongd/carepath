import {
  AppBar,
  Button,
  ChevronRightIcon,
  JourneyRail,
  Lead,
  PageTitle,
  RefPill,
  Screen,
  ScreenBody,
  ScreenDock,
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

      <ScreenBody className="pt-1.5 pb-40">
        <PageTitle className="mt-3.5 mb-1.5">การมาโรงพยาบาลของคุณวันนี้</PageTitle>
        <Lead className="mb-7 max-w-[300px]">
          ติดตามขั้นตอนของคุณ แล้วไปยังจุดบริการถัดไปได้จากปุ่มด้านล่าง
        </Lead>

        <JourneyRail steps={journeySteps} />
      </ScreenBody>

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
