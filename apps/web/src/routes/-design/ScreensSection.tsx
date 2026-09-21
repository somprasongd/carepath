import type { ReactNode } from 'react'
import { floorLabelFor, planUrlFor, useFloorPlan, useFloors } from '@/features/floorplan'
import { NavigateScreen } from '../patient/-NavigateScreen'
import { StaffAuthProvider } from '@/auth/StaffAuthProvider'
import { LoginScreen } from '../-LoginScreen'
import { Overview } from '../staff/-Overview'
import { Queue } from '../staff/-Queue'
import { PathwayTemplates } from '../staff/-PathwayTemplates'
import { ServicePoints } from '../staff/-ServicePoints'
import { ReferenceJourney, ReferenceJourneyDone } from './ReferenceJourney'
import { ScreenFrame, Section, SubHead } from './CatalogueParts'

const MOBILE = { width: 390, height: 844, scale: 0.82 }
const DESKTOP = { width: 1440, height: 900, scale: 0.62 }

/**
 * The gallery's Thai specimen of a resolved destination plan. The screen
 * takes this as a prop now — the component itself stays language-neutral
 * (ADR-0012), so this literal lives in the staff/demo gallery, not in the
 * patient component.
 */
const REFERENCE_PLAN = {
  title: 'เส้นทางไปรับยา',
  name: 'Pharmacy',
  subtitle: 'ชั้น 1 · Pharmacy · PHARMACY-01',
  floorId: 'I-1301',
  floorLabel: 'ชั้น 1',
  placeId: 'PHARMACY-01',
  servicePointCode: 'PHARMACY',
  x: 885,
  y: 190,
}

/**
 * The gallery specimen of the navigate screen. It fetches the plan the same
 * way the real route does (ADR-0015) rather than holding a copy: with the
 * API up this is the live drawing, and with it down the frame shows the
 * screen's honest map-unavailable state — both worth seeing in a gallery.
 */
function ReferenceNavigate() {
  const { data: floors } = useFloors()
  const planUrl = planUrlFor(REFERENCE_PLAN.floorId, floors ?? [])
  const planQuery = useFloorPlan(planUrl)
  return (
    <NavigateScreen
      plan={{ state: 'plan', ...REFERENCE_PLAN }}
      planSvg={planQuery.data}
      planUnavailable={planQuery.isError || (floors !== undefined && planUrl === undefined)}
      floorLabel={(floorId) => floorLabelFor(floorId, 'th', floors ?? [])}
    />
  )
}

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
          <code>docs/deliverables/03-prototype-wireframe.md</code> §3.3 (เข้าสู่ระบบ, แผนการดูแล,
          เรียกคิว) — ออกแบบไว้ก่อนใน canvas เดียวกัน แล้วนำมาสร้างเป็นคอมโพเนนต์จริงที่นี่ ปรับตาม
          ความกว้างของกรอบด้วย container query เหมือนหน้าเจ้าหน้าที่ชุดเดิม จึงมีทั้งเวอร์ชันมือถือและ
          เดสก์ท็อป หน้า "แผนการดูแล" เปลี่ยนบทบาทไปจากดีไซน์เดิมใน canvas — ดูเหตุผลใน
          §3.3
        </>
      }
    >
      <SubHead>ผู้ป่วย · มือถือ (LINE LIFF)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="หน้าแรกเส้นทาง" {...MOBILE}>
          <ReferenceJourney />
        </ScreenFrame>
        <ScreenFrame title="หน้าแรกเส้นทาง · เสร็จสิ้น" {...MOBILE}>
          <ReferenceJourneyDone />
        </ScreenFrame>
        <ScreenFrame title="นำทางไปจุดบริการ" {...MOBILE}>
          <ReferenceNavigate />
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · มือถือ</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...MOBILE}>
          <StaffSession><Overview /></StaffSession>
        </ScreenFrame>
        <ScreenFrame title="ผังจุดบริการ" {...MOBILE}>
          <StaffSession><ServicePoints /></StaffSession>
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · เดสก์ท็อป (คอนโซลหน้าเคาน์เตอร์)</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="ภาพรวม" {...DESKTOP}>
          <StaffSession><Overview /></StaffSession>
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="ผังจุดบริการ" {...DESKTOP}>
          <StaffSession><ServicePoints /></StaffSession>
        </ScreenFrame>
      </div>

      <SubHead>
        เจ้าหน้าที่ · เข้าสู่ระบบ, แผนการดูแล, เรียกคิว · มือถือ (FR-13, FR-14, FR-15, FR-16, FR-18)
      </SubHead>
      <div className="ds-frames">
        <ScreenFrame title="เข้าสู่ระบบ · เลือกบทบาท" {...MOBILE}>
          <StaffSession><LoginScreen /></StaffSession>
        </ScreenFrame>
        <ScreenFrame title="แผนการดูแล (Pathway Template)" {...MOBILE}>
          <StaffSession><PathwayTemplates /></StaffSession>
        </ScreenFrame>
        <ScreenFrame title="เรียกคิว" {...MOBILE}>
          <StaffSession>
            <Queue servicePointId={undefined} onServicePointChange={() => {}} />
          </StaffSession>
        </ScreenFrame>
      </div>

      <SubHead>เจ้าหน้าที่ · เข้าสู่ระบบ, แผนการดูแล, เรียกคิว · เดสก์ท็อป</SubHead>
      <div className="ds-frames">
        <ScreenFrame title="เข้าสู่ระบบ · เลือกบทบาท" {...DESKTOP}>
          <StaffSession><LoginScreen /></StaffSession>
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="แผนการดูแล (Pathway Template)" {...DESKTOP}>
          <StaffSession><PathwayTemplates /></StaffSession>
        </ScreenFrame>
      </div>
      <div className="ds-frames mt-6">
        <ScreenFrame title="เรียกคิว" {...DESKTOP}>
          <StaffSession>
            <Queue servicePointId={undefined} onServicePointChange={() => {}} />
          </StaffSession>
        </ScreenFrame>
      </div>
    </Section>
  )
}

/**
 * The gallery renders the real login form, which reads the staff auth
 * context; the provider is stubbed in so the frame shows the signed-out
 * state (submitting would hit the live API and land on its error line).
 */
function StaffSession({ children }: { children: ReactNode }) {
  return <StaffAuthProvider>{children}</StaffAuthProvider>
}
