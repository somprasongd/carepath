import { NavigateScreen } from '../patient/-NavigateScreen'
import { LoginScreen } from '../-LoginScreen'
import { Overview } from '../staff/-Overview'
import { Queue } from '../staff/-Queue'
import { Register } from '../staff/-Register'
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
          หกหน้าจอฐานจาก <code>docs/designs/patient-staff-ui.html</code> ประกอบใหม่เป็น React จริง
          ใน <code>src/routes/</code> — เลื่อนภายในกรอบได้ และกดเมนูของเจ้าหน้าที่ได้ (เมนูจะพาไปที่ URL
          จริง) หน้าเจ้าหน้าที่เป็นคอมโพเนนต์เดียว ปรับเองตามความกว้างของกรอบด้วย container query —
          ไม่ใช่สองหน้าที่แยกกัน ต่อจากนั้นคือ 3 หน้าจอที่ปิดช่องว่างใน{' '}
          <code>docs/deliverables/03-prototype-wireframe.md</code> §3.3 (เข้าสู่ระบบ, ลงทะเบียน,
          เรียกคิว) — ออกแบบไว้ก่อนใน canvas เดียวกัน แล้วนำมาสร้างเป็นคอมโพเนนต์จริงที่นี่ ปรับตาม
          ความกว้างของกรอบด้วย container query เหมือนหน้าเจ้าหน้าที่ชุดเดิม จึงมีทั้งเวอร์ชันมือถือและ
          เดสก์ท็อป
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

      <SubHead>
        เจ้าหน้าที่ · เข้าสู่ระบบ, ลงทะเบียน, เรียกคิว · มือถือ (FR-13, FR-14, FR-15, FR-16, FR-18)
      </SubHead>
      <div className="ds-frames">
        <ScreenFrame title="เข้าสู่ระบบ · เลือกบทบาท" {...MOBILE}>
          <LoginScreen onSignIn={() => {}} />
        </ScreenFrame>
        <ScreenFrame title="ลงทะเบียนผู้ป่วย" {...MOBILE}>
          <Register />
        </ScreenFrame>
        <ScreenFrame title="เรียกคิว" {...MOBILE}>
          <Queue />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · เข้าสู่ระบบ, ลงทะเบียน, เรียกคิว · เดสก์ท็อป</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="เข้าสู่ระบบ · เลือกบทบาท" {...DESKTOP}>
          <LoginScreen onSignIn={() => {}} />
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="ลงทะเบียนผู้ป่วย" {...DESKTOP}>
          <Register />
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="เรียกคิว" {...DESKTOP}>
          <Queue />
        </ScreenFrame>
      </div>
    </Section>
  )
}
