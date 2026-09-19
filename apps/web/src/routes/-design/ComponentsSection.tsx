import { useState } from 'react'
import {
  AttentionCard,
  BottomSheet,
  BottomTabBar,
  Button,
  Card,
  ChevronRightIcon,
  DataTable,
  InfoNote,
  JourneyRail,
  LoadBadge,
  LocationBanner,
  QueuePill,
  RefPill,
  RouteStatusBadge,
  SchematicMap,
  ServicePointMappingCard,
  ServicePointRow,
  SideRail,
  StatCard,
  StickyActionBar,
  TableCode,
  ZoneChip,
  ZoneDot,
  ZoneLegend,
  type Column,
  type Zone,
} from '@/design-system'
import {
  attentionList,
  currentLocation,
  floor1Corridor,
  floor1Rooms,
  journeySteps,
  pharmacyRoute,
  pharmacyRouteEnd,
  servicePointLoad,
  servicePointMappings,
  walkingSteps,
  type ServicePointLoad,
} from '@/mocks/demo-data'
import { staffNavItems, staffRailItems, staffRole } from '../staff/-nav'
import { Section, Specimen, SubHead } from './CatalogueParts'

const allZones: Zone[] = ['public', 'opd', 'diagnostic', 'pharmacy', 'rehab', 'ipd', 'support']

const tableColumns: Column<ServicePointLoad>[] = [
  { key: 'point', header: 'จุดบริการ', render: (sp) => <TableCode code={sp.code} sub={sp.name} /> },
  { key: 'zone', header: 'โซน', render: (sp) => <ZoneChip zone={sp.zone} /> },
  {
    key: 'waiting',
    header: 'กำลังรอ',
    render: (sp) => (
      <span className={`cp-sp-row__count ${sp.load === 'busy' ? 'cp-sp-row__count--busy' : ''}`}>
        {sp.waiting}
      </span>
    ),
  },
  { key: 'load', header: 'สถานะ', render: (sp) => <LoadBadge load={sp.load} /> },
]

export function ComponentsSection() {
  const [tab, setTab] = useState('overview')
  const [railItem, setRailItem] = useState('service-points')

  return (
    <Section
      id="components"
      index="02"
      title="Components"
      description={
        <>
          คอมโพเนนต์จริงจาก <code>src/design-system/</code> — หน้าจอตัวอย่างด้านล่างประกอบขึ้นจากชิ้นเหล่านี้
          ทั้งหมด ไม่มีการเขียนสไตล์ซ้ำ
        </>
      }
    >
      <SubHead>ปุ่ม</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen
          name="Button · primary"
          note="CTA ของผู้ป่วยหนึ่งปุ่มต่อหนึ่งหน้าจอ · ตัวอักษรสี ink บนพื้นส้ม (ขาวได้ contrast แค่ 2.6:1)"
        >
          <Button variant="primary">
            <span>นำทางไปห้องยา</span>
            <ChevronRightIcon />
          </Button>
        </Specimen>
        <Specimen name="Button · secondary" note="การกระทำหลักของเจ้าหน้าที่ — สีน้ำเงินโหนดนำทาง">
          <Button variant="secondary">+ เพิ่มการเชื่อมโยง</Button>
        </Specimen>
        <Specimen name="Button · ghost" note="การกระทำรองของเจ้าหน้าที่ · สูงอย่างน้อย 44px">
          <Button variant="ghost">ช่วยนำทาง</Button>
          <Button variant="ghost">ตรวจสอบคิว</Button>
        </Specimen>
      </div>

      <SubHead>ชิปโซนและป้ายสถานะ</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen
          name="ZoneChip"
          note="หนึ่งแบบต่อหนึ่งโซนในผังอาคาร · ตัวอักษรสี ink เสมอเพราะทุกสีโซนอ่อน"
        >
          {allZones.map((zone) => (
            <ZoneChip key={zone} zone={zone} />
          ))}
        </Specimen>
        <Specimen name="ZoneDot" note="ชิปที่เหลือแต่สี — ใช้ในแถวที่รหัสทำหน้าที่บอกชื่อแล้ว">
          {allZones.map((zone) => (
            <ZoneDot key={zone} zone={zone} />
          ))}
        </Specimen>
        <Specimen
          name="RouteStatusBadge"
          note="บอกว่า Place นี้มีโหนดในผังนำทางแล้วหรือยัง — ห้ามซ่อนห้องที่ยังไม่มีโหนด"
        >
          <RouteStatusBadge status="routable" />
          <RouteStatusBadge status="not-routable" />
        </Specimen>
        <Specimen name="LoadBadge · QueuePill · RefPill" note="สถานะเชิงปฏิบัติการ ไม่ใช่การแจ้งเตือน">
          <LoadBadge load="normal" />
          <LoadBadge load="busy" />
          <QueuePill>คิวที่ 12</QueuePill>
          <RefPill>V-2384</RefPill>
        </Specimen>
      </div>

      <SubHead>พื้นผิวและการ์ด</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="Card · lg" note="การ์ดมาตรฐาน: ขอบ 1px ไม่มีเงา" paper>
          <Card style={{ width: '100%' }}>
            <div className="cp-h2">ห้องยา</div>
            <div className="cp-rail__meta">ชั้น 1 · PHARMACY-01</div>
          </Card>
        </Specimen>
        <Specimen
          name="Card · ticket"
          note="มุมซ้ายบนคม มุมอื่นโค้ง — สัญญะ 'ใบคิวที่คุณถืออยู่' สงวนไว้ให้ขั้นตอนปัจจุบันเท่านั้น"
          paper
        >
          <Card radius="ticket" padding="md" style={{ width: '100%' }}>
            <div className="cp-rail__eyebrow">ขั้นตอนปัจจุบัน</div>
            <div className="cp-ticket__name">เจาะเลือด</div>
            <div className="cp-ticket__place" style={{ marginBottom: 0 }}>
              ห้องเจาะเลือด · ชั้น 2 · LAB-01
            </div>
          </Card>
        </Specimen>
        <Specimen name="StatCard" note="KPI สีส้มได้เพียงตัวเดียวต่อหนึ่งหน้าจอ" paper block>
          <div className="cp-grid-2">
            <StatCard compact label="กำลังใช้บริการ" value="24" unit="ราย" />
            <StatCard compact tone="attention" label="ไม่ทราบตำแหน่ง" value="3" unit="ราย" />
          </div>
        </Specimen>
      </div>

      <SubHead>เส้นทางการรับบริการ</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen
          name="JourneyRail"
          note="เส้นเชื่อมเปลี่ยนเป็นสีส้มหนึ่งขั้นก่อนขั้นตอนปัจจุบัน สะท้อนเส้นทางบนผังอาคาร"
          paper
          block
        >
          <JourneyRail steps={journeySteps} />
        </Specimen>

        <div>
          <Specimen
            name="SchematicMap"
            note="รูปทรงของโหนดและเส้นทางเหมือนใน SVG ทุกประการ — ผังในแอปกับไฟล์ผังจริงสลับกันได้"
            paper
            block
          >
            <SchematicMap
              rooms={floor1Rooms}
              corridor={floor1Corridor}
              route={pharmacyRoute}
              routeEnd={pharmacyRouteEnd}
              you={currentLocation}
            />
          </Specimen>
          <div style={{ height: 'var(--cp-space-lg)' }} />
          <Specimen
            name="LocationBanner"
            note="สถานะจาก location provider ที่ใช้อยู่ (QR / Zigbee / เลือกเอง) ตาม ADR-0004"
            paper
            block
          >
            <LocationBanner actionLabel="สแกนใหม่">
              ตำแหน่งล่าสุดจากการสแกน QR ที่ทางลงชั้น 1 · 2 นาทีที่แล้ว
            </LocationBanner>
          </Specimen>
        </div>
      </div>

      <SubHead>รายการของเจ้าหน้าที่</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="ServicePointRow" note="แถวหนาแน่น: สี, รหัส, ชื่อ, จำนวนคิว" block>
          {servicePointLoad.map((sp) => (
            <ServicePointRow
              key={sp.code}
              zone={sp.zone}
              code={sp.code}
              name={sp.name}
              waiting={sp.waiting}
              busy={sp.load === 'busy'}
            />
          ))}
        </Specimen>
        <Specimen name="ServicePointMappingCard" note="การเชื่อมขั้นตอนการดูแล → สถานที่จริง หนึ่งใบ" paper block>
          <div className="cp-stack">
            <ServicePointMappingCard {...servicePointMappings[3]} />
            <ServicePointMappingCard {...servicePointMappings[6]} />
          </div>
        </Specimen>
        <Specimen name="AttentionCard" note="ผู้ป่วยที่ประชาสัมพันธ์ต้องลงมือทำอะไรสักอย่าง" paper block>
          <div className="cp-stack">
            {attentionList.map((item) => (
              <AttentionCard key={item.visitRef} {...item} />
            ))}
          </div>
        </Specimen>
        <Specimen name="InfoNote · ZoneLegend" note="อธิบายกฎของโดเมน ไม่ใช่การเตือน" paper block>
          <div className="cp-stack">
            <InfoNote>ห้องที่ยังไม่มีโหนดนำทางจะเลือกได้ แต่จะยังไม่แสดงเส้นทางให้ผู้ป่วย</InfoNote>
            <Card>
              <ZoneLegend only={['public', 'opd', 'diagnostic', 'pharmacy']} />
            </Card>
          </div>
        </Specimen>
      </div>

      <SubHead>ตาราง (เดสก์ท็อปของเจ้าหน้าที่เท่านั้น)</SubHead>
      <div className="ds-specimen">
        <div className="ds-specimen__stage ds-specimen__stage--block">
          <DataTable columns={tableColumns} rows={servicePointLoad} rowKey={(sp) => sp.code} />
        </div>
        <div className="ds-specimen__foot">
          <div className="ds-specimen__name">DataTable</div>
          <p className="ds-specimen__note">
            คู่ตรงข้ามแบบ responsive ของรายการการ์ดบนมือถือ ไม่ใช่ดีไซน์คนละชุด
          </p>
        </div>
      </div>

      <SubHead>โครงหน้าจอที่ลอยอยู่</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="StickyActionBar" note="อยู่ในระยะนิ้วโป้ง · หนึ่งปุ่มหลักเท่านั้น" paper block>
          <div style={{ maxWidth: 390 }}>
            <StickyActionBar label="ขั้นตอนถัดไป" value="รับยา · ห้องยา ชั้น 1">
              <Button variant="primary" block>
                <span>นำทางไปห้องยา</span>
                <ChevronRightIcon />
              </Button>
            </StickyActionBar>
          </div>
        </Specimen>
        <Specimen name="BottomSheet" note="สรุปเป็นตัวเลขใหญ่ แล้วตามด้วยคำบอกทางแบบข้อความ" paper block>
          <div style={{ maxWidth: 390 }}>
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
          </div>
        </Specimen>
      </div>

      <SubHead>เมนู</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="BottomTabBar" note="เจ้าหน้าที่บนมือถือ · 4 ปลายทางเดียวกับบนเดสก์ท็อป" paper block>
          <div style={{ maxWidth: 390, border: '1px solid var(--cp-line)' }}>
            <BottomTabBar items={staffNavItems} activeId={tab} onSelect={setTab} />
          </div>
        </Specimen>
        <Specimen name="SideRail" note="เจ้าหน้าที่บนเดสก์ท็อป · rail กว้าง 220px ตายตัว" paper block>
          <div style={{ height: 420, display: 'flex' }}>
            <SideRail
              items={staffRailItems}
              activeId={railItem}
              onSelect={setRailItem}
              role={staffRole}
            />
          </div>
        </Specimen>
      </div>
    </Section>
  )
}
