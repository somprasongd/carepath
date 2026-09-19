import { colors, radius, spacing, typography, zones, type Zone } from '@/design-system'
import { Section, Specimen, SubHead, Swatch } from './CatalogueParts'

const colorRoles: Record<string, string> = {
  primary: 'สีเส้นทางจากผังอาคาร แปลว่า "นี่คือเส้นทางของคุณ ทำสิ่งนี้ตอนนี้" เท่านั้น',
  'primary-tint': 'พื้นหลังของคิว, KPI ที่ต้องดูแล และแถบเตือนแบบไม่เร่งเร้า',
  secondary: 'สีโหนดนำทาง = สีหลักของฝั่งเจ้าหน้าที่ (ลิงก์, ปุ่ม, เมนูที่เลือกอยู่)',
  'secondary-tint': 'พื้นหลังเมนูที่เลือกอยู่ และแถบบอกตำแหน่งปัจจุบัน',
  ink: 'ตัวอักษรหลัก และตัวอักษรบนปุ่มสีส้ม',
  'ink-muted': 'คำอธิบาย, meta, ป้ายกำกับบนผัง — ค่าเดียวกับ --muted ใน SVG',
  neutral: 'พื้นหลังหน้า = สีพื้นทางเดินจากผังอาคาร',
  surface: 'การ์ดและแผงที่วางอยู่บนพื้นหลัง',
  line: 'เส้นคั่นและขอบการ์ด 1px (แทนการใช้เงา)',
  success: 'สถานะ "เดินทางได้" — ไม่ใช่สีเขียวแจ้งเตือน',
  'success-tint': 'พื้นหลังของป้าย routable',
  warning: 'สถานะ "ยังไม่รองรับเส้นทาง" / จุดบริการหนาแน่น',
  'warning-tint': 'พื้นหลังของป้าย not-routable',
}

const typeSamples: Record<keyof typeof typography, string> = {
  'display-stat': '24',
  h1: 'การมาโรงพยาบาลของคุณวันนี้',
  h2: 'จุดบริการวันนี้',
  'body-md': 'ติดตามขั้นตอนของคุณ แล้วไปยังจุดบริการถัดไปได้จากปุ่มด้านล่าง',
  'body-sm': 'ห้องเจาะเลือด · ชั้น 2 · LAB-01',
  'label-code': 'PHARMACY-01',
  caption: 'อัปเดตล่าสุด 09:42 น.',
  button: 'นำทางไปห้องยา',
}

export function FoundationsSection() {
  return (
    <Section
      id="foundations"
      index="01"
      title="Foundations"
      description={
        <>
          ทุกค่าในหน้านี้มาจาก frontmatter ของ <code>DESIGN.md</code> ซึ่งคัดลอกมาจาก{' '}
          <code>:root</code> ของไฟล์ <code>packages/floorplans/floors/*.svg</code> อีกทอดหนึ่ง —
          นี่คือข้อบังคับ ไม่ใช่ความชอบ เพื่อให้ UI ไม่ตีกับผังอาคารที่มันเรนเดอร์อยู่
        </>
      }
    >
      <SubHead>สีหลัก</SubHead>
      <div className="ds-grid">
        {Object.entries(colors).map(([name, value]) => (
          <Swatch key={name} name={name} value={value} use={colorRoles[name] ?? ''} />
        ))}
      </div>

      <SubHead>สีโซน — คัดลอกตรงจากผังอาคาร</SubHead>
      <div className="ds-grid">
        {(Object.keys(zones) as Zone[]).map((zone) => (
          <Swatch
            key={zone}
            name={`zone-${zone}`}
            value={zones[zone].fill}
            use={`${zones[zone].label} — ${zones[zone].use}`}
          />
        ))}
      </div>

      <SubHead>ตัวอักษร</SubHead>
      <div className="ds-specimen" style={{ padding: '0 var(--cp-space-xl)' }}>
        {(Object.keys(typography) as (keyof typeof typography)[]).map((token) => {
          const spec = typography[token]
          return (
            <div className="ds-type" key={token}>
              <div className="ds-type__meta">
                <code className="ds-type__token">{token}</code>
                {spec.family} · {spec.size} · {spec.weight} · line-height {spec.lineHeight}
              </div>
              <div style={{ font: `var(--cp-text-${token})` }}>{typeSamples[token]}</div>
            </div>
          )
        })}
      </div>
      <p className="ds-section__desc" style={{ marginTop: 'var(--cp-space-md)' }}>
        Noto Sans Thai รับงานข้อความทั้งหมด (เป็นฟอนต์เดียวกับป้ายในผังอาคาร) ส่วน Space Grotesk
        สงวนไว้สำหรับตัวเลขและรหัสสั้น ๆ เช่น <code>REG-01</code>, <code>V-2384</code>, 11 นาที
        ไม่ผสมสองฟอนต์ในข้อความชิ้นเดียวกัน
      </p>

      <SubHead>ระยะห่างและมุมโค้ง</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="spacing" note="สเกลฐาน 4px · gutter 20px บนมือถือ, 40px บนเดสก์ท็อป" block>
          <div className="ds-scale">
            {Object.entries(spacing).map(([name, value]) => (
              <div className="ds-scale__row" key={name}>
                <span className="ds-scale__label">{name}</span>
                <span className="ds-scale__bar" style={{ width: value }} />
                <span className="ds-type__meta">{value}</span>
              </div>
            ))}
          </div>
        </Specimen>

        <Specimen
          name="rounded"
          note="ฝั่งผู้ป่วยใช้ lg/xl และ pill — ฝั่งเจ้าหน้าที่ใช้ sm/md ให้ดูใกล้เคียงป้ายที่พิมพ์ออกมา"
        >
          {Object.entries(radius)
            .filter(([name]) => name !== 'full')
            .map(([name, value]) => (
              <div className="ds-radius-box" key={name} style={{ borderRadius: value }}>
                {name}
              </div>
            ))}
          <div className="ds-radius-box" style={{ borderRadius: radius.full, width: 110 }}>
            full
          </div>
        </Specimen>
      </div>

      <SubHead>ความลึก</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen
          name="ไม่มีเงา (ค่าเริ่มต้น)"
          note="การ์ดและแถวแยกกันด้วยเส้น 1px — ให้บอร์ดของเจ้าหน้าที่อ่านเหมือนป้ายรวมแผนก"
          paper
        >
          <div
            style={{
              background: 'var(--cp-surface)',
              border: '1px solid var(--cp-line)',
              borderRadius: 'var(--cp-radius-lg)',
              padding: 'var(--cp-space-lg)',
              width: '100%',
              font: 'var(--cp-text-body-sm)',
              color: 'var(--cp-ink-muted)',
            }}
          >
            การ์ดปกติ
          </div>
        </Specimen>

        <Specimen
          name="shadow-bar / shadow-sheet"
          note="สงวนเงาไว้ให้เฉพาะสิ่งที่ลอยอยู่เหนือเนื้อหาที่เลื่อนได้จริง: แถบปุ่มติดขอบล่าง และ bottom sheet"
          paper
        >
          <div
            style={{
              background: 'var(--cp-surface)',
              borderTop: '1px solid var(--cp-line)',
              boxShadow: 'var(--cp-shadow-bar)',
              borderRadius: 'var(--cp-radius-md)',
              padding: 'var(--cp-space-lg)',
              width: '100%',
              font: 'var(--cp-text-body-sm)',
              color: 'var(--cp-ink-muted)',
            }}
          >
            แถบที่ลอยอยู่
          </div>
        </Specimen>
      </div>
    </Section>
  )
}
