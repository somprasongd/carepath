import { NavigateScreen } from '../patient/-NavigateScreen'
import { Overview } from '../staff/-Overview'
import { ServicePoints } from '../staff/-ServicePoints'
import { ReferenceJourney } from './ReferenceJourney'
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
          (เมนูจะพาไปที่ URL จริง) หน้าเจ้าหน้าที่เป็นคอมโพเนนต์เดียว ปรับเองตามความกว้างของกรอบ
          ด้วย container query — ไม่ใช่สองหน้าที่แยกกัน
        </>
      }
    >
      <SubHead>ผู้ป่วย · มือถือ (LINE LIFF)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="หน้าแรกเส้นทาง" {...MOBILE}>
          <ReferenceJourney />
        </ScreenFrame>
        <ScreenFrame title="นำทางไปจุดบริการ" {...MOBILE}>
          <NavigateScreen />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · มือถือ</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...MOBILE}>
          <Overview />
        </ScreenFrame>
        <ScreenFrame title="ผังจุดบริการ" {...MOBILE}>
          <ServicePoints />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · เดสก์ท็อป (คอนโซลหน้าเคาน์เตอร์)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...DESKTOP}>
          <Overview />
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="ผังจุดบริการ" {...DESKTOP}>
          <ServicePoints />
        </ScreenFrame>
      </div>
    </Section>
  )
}
