import { Card, SectionTitle, Lead } from '@/design-system'
import './design-page.css'
import { ComponentsSection } from './ComponentsSection'
import { FoundationsSection } from './FoundationsSection'
import { RulesSection } from './RulesSection'
import { ScreensSection } from './ScreensSection'

const sections = [
  { id: 'foundations', label: '01 Foundations' },
  { id: 'components', label: '02 Components' },
  { id: 'screens', label: '03 Screens' },
  { id: 'rules', label: "04 Do's & Don'ts" },
]

/**
 * /design — the living version of DESIGN.md. Everything below renders from the
 * same components the product uses, so a drift between spec and code shows up
 * here first.
 */
export function DesignSystemPage() {
  return (
    <div className="ds-page font-sans text-ink antialiased">
      <div className="ds-shell">
        <header className="ds-hero">
          <div className="ds-hero__eyebrow">CAREPATH DESIGN SYSTEM · ALPHA</div>
          <h1 className="ds-hero__title">ภาษาภาพเดียว สำหรับผู้ป่วยและเจ้าหน้าที่</h1>
          <p className="ds-hero__lead">
            CarePath พาผู้ป่วยจากขั้นตอนหนึ่งไปอีกขั้นตอนหนึ่ง และให้เจ้าหน้าที่เห็นการเดินทางเดียวกันจากฝั่งปฏิบัติการ
            เรื่องจริงของผลิตภัณฑ์นี้คือป้ายบอกทางในโรงพยาบาล — แถบสีโซน เส้นผนังสีเข้ม
            และเส้นสีส้มบนพื้นที่บอกว่า "เดินตามนี้ไปห้องยา" ระบบดีไซน์นี้จึงไม่ได้คิดสกินขึ้นใหม่
            แต่ถอดออกมาจากไฟล์ผังอาคารของโปรเจกต์เองโดยตรง
          </p>

          <div className="ds-registers">
            <Card>
              <SectionTitle className="mb-0">ผู้ป่วย</SectionTitle>
              <Lead className="mt-2">
                มือถืออย่างเดียว เปิดจาก LINE LIFF มักเป็นคนที่กังวล สูงอายุ
                หรือไม่เคยเห็นอาคารนี้มาก่อน — หนึ่งการกระทำหลักต่อหนึ่งหน้าจอ ตัวอักษรใหญ่
                ไม่มีศัพท์ทางคลินิก
              </Lead>
            </Card>
            <Card>
              <SectionTitle className="mb-0">เจ้าหน้าที่</SectionTitle>
              <Lead className="mt-2">
                เริ่มที่มือถือ แต่ต้องขยายไปเป็นคอนโซลบนเดสก์ท็อปของเคาน์เตอร์หรือเคาน์เตอร์พยาบาลได้ —
                ออกแบบให้กวาดสายตาหลายแถวได้เร็ว และจับคู่แถวกับพื้นที่สีบนผังได้ด้วยสี
              </Lead>
            </Card>
          </div>
        </header>

        <nav className="ds-nav" aria-label="สารบัญระบบดีไซน์">
          {sections.map((section) => (
            <a className="ds-nav__link" key={section.id} href={`#${section.id}`}>
              {section.label}
            </a>
          ))}
        </nav>

        <FoundationsSection />
        <ComponentsSection />
        <ScreensSection />
        <RulesSection />

        <p className="ds-footnote">
          แหล่งอ้างอิง: <code>DESIGN.md</code> (สเปก) · <code>docs/designs/patient-staff-ui.html</code>{' '}
          (ภาพอ้างอิง) · <code>packages/floorplans/floors/*.svg</code> (ที่มาของสี) — โค้ดอยู่ที่{' '}
          <code>apps/web/src/design-system/</code> และ <code>apps/web/src/routes/</code>
        </p>
      </div>
    </div>
  )
}
