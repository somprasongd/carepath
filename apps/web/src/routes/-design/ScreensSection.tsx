import { JourneyScreen } from '../patient/-JourneyScreen'
import { NavigateScreen } from '../patient/-NavigateScreen'
import { OverviewDesktop } from '../staff/-OverviewDesktop'
import { OverviewMobile } from '../staff/-OverviewMobile'
import { ServicePointsDesktop } from '../staff/-ServicePointsDesktop'
import { ServicePointsMobile } from '../staff/-ServicePointsMobile'
import { ScreenFrame, Section, SubHead } from './CatalogueParts'

const MOBILE = { width: 390, height: 844, scale: 0.82 }
const DESKTOP = { width: 1440, height: 900, scale: 0.62 }

export function ScreensSection() {
  return (
    <Section
      id="screens"
      index="03"
      title="Reference screens"
      description={
        <>
          หกหน้าจออ้างอิงจาก <code>docs/designs/patient-staff-ui.html</code> ประกอบใหม่เป็น React
          จริงใน <code>src/routes/</code> — เลื่อนภายในกรอบได้ และกดเมนูของเจ้าหน้าที่ได้
          (เมนูจะพาไปที่ URL จริง)
        </>
      }
    >
      <SubHead>ผู้ป่วย · มือถือ (LINE LIFF)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="หน้าแรกเส้นทาง" {...MOBILE}>
          <JourneyScreen />
        </ScreenFrame>
        <ScreenFrame title="นำทางไปจุดบริการ" {...MOBILE}>
          <NavigateScreen />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · มือถือ</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...MOBILE}>
          <OverviewMobile />
        </ScreenFrame>
        <ScreenFrame title="ผังจุดบริการ" {...MOBILE}>
          <ServicePointsMobile />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · เดสก์ท็อป (คอนโซลหน้าเคาน์เตอร์)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...DESKTOP}>
          <OverviewDesktop />
        </ScreenFrame>
      </div>
      <div className="ds-frames" style={{ marginTop: 'var(--cp-space-2xl)' }}>
        <ScreenFrame title="ผังจุดบริการ" {...DESKTOP}>
          <ServicePointsDesktop />
        </ScreenFrame>
      </div>
    </Section>
  )
}
