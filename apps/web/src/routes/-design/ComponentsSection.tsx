import { useState } from 'react'
import {
  BottomSheet,
  BottomTabBar,
  Button,
  Card,
  ChevronRightIcon,
  DataTable,
  InfoNote,
  JourneyRail,
  LinkButton,
  LoadBadge,
  LocationBanner,
  QueuePill,
  RefPill,
  RouteStatusBadge,
  SchematicMap,
  SectionTitle,
  SideRail,
  Stack,
  StatCard,
  StickyActionBar,
  TableCode,
  ZoneChip,
  ZoneDot,
  ZoneLegend,
  type Column,
  type Zone,
} from '@/design-system'
import { AttentionCard } from '@/features/visit'
import { ServicePointMappingCard, ServicePointRow } from '@/features/servicepoint'
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
import { staffNavItems, staffRailItems, staffRoleLabel } from '../staff/-nav'
import { Section, Specimen, SubHead } from './CatalogueParts'

const allZones: Zone[] = ['public', 'opd', 'diagnostic', 'pharmacy', 'rehab', 'ipd', 'support']

const tableColumns: Column<ServicePointLoad>[] = [
  { key: 'point', header: 'จุดบริการ', render: (sp) => <TableCode code={sp.code} sub={sp.name} /> },
  { key: 'zone', header: 'โซน', render: (sp) => <ZoneChip zone={sp.zone} /> },
  {
    key: 'waiting',
    header: 'กำลังรอ',
    render: (sp) => (
      <span className={`font-code text-label-code ${sp.load === 'busy' ? 'text-primary' : 'text-ink'}`}>
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
          <Card className="w-full">
            <SectionTitle className="mb-0">ห้องยา</SectionTitle>
            <div className="mt-0.5 font-sans text-body-sm text-ink-muted">ชั้น 1 · PHARMACY-01</div>
          </Card>
        </Specimen>
        <Specimen
          name="Card · ticket"
          note="มุมซ้ายบนคม มุมอื่นโค้ง — สัญญะ 'ใบคิวที่คุณถืออยู่' สงวนไว้ให้ขั้นตอนปัจจุบันเท่านั้น"
          paper
        >
          <Card radius="ticket" padding="md" className="w-full">
            <div className="mb-1 font-sans text-caption text-ink-muted">ขั้นตอนปัจจุบัน</div>
            <div className="mb-1 text-[18px] font-bold text-ink">เจาะเลือด</div>
            <div className="font-sans text-body-sm text-ink-muted">ห้องเจาะเลือด · ชั้น 2 · LAB-01</div>
          </Card>
        </Specimen>
        <Specimen name="StatCard" note="KPI สีส้มได้เพียงตัวเดียวต่อหนึ่งหน้าจอ" paper block>
          <div className="grid grid-cols-2 gap-3">
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
          <JourneyRail
            steps={journeySteps}
            labels={{ currentStep: 'ขั้นตอนปัจจุบัน', nextStep: 'ขั้นตอนถัดไป' }}
          />
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
              ariaLabel={`ผังเส้นทางจาก${currentLocation.label}ไปยังจุดหมาย`}
            />
          </Specimen>
          <div className="h-4" />
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
          <Stack>
            <ServicePointMappingCard {...servicePointMappings[3]} />
            <ServicePointMappingCard {...servicePointMappings[6]} />
          </Stack>
        </Specimen>
        <Specimen name="AttentionCard" note="ผู้ป่วยที่ประชาสัมพันธ์ต้องลงมือทำอะไรสักอย่าง" paper block>
          <Stack>
            {attentionList.map((item) => (
              <AttentionCard key={item.visitRef} {...item} />
            ))}
          </Stack>
        </Specimen>
        <Specimen name="InfoNote · ZoneLegend" note="อธิบายกฎของโดเมน ไม่ใช่การเตือน" paper block>
          <Stack>
            <InfoNote>ห้องที่ยังไม่มีโหนดนำทางจะเลือกได้ แต่จะยังไม่แสดงเส้นทางให้ผู้ป่วย</InfoNote>
            <Card>
              <ZoneLegend only={['public', 'opd', 'diagnostic', 'pharmacy']} />
            </Card>
          </Stack>
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
          <div className="max-w-[390px]">
            <StickyActionBar label="ขั้นตอนถัดไป" value="รับยา · ห้องยา ชั้น 1">
              <Button variant="primary" block>
                <span>นำทางไปห้องยา</span>
                <ChevronRightIcon />
              </Button>
            </StickyActionBar>
          </div>
        </Specimen>
        <Specimen name="BottomSheet" note="สรุปเป็นตัวเลขใหญ่ แล้วตามด้วยคำบอกทางแบบข้อความ" paper block>
          <div className="max-w-[390px]">
            <BottomSheet
              primary="3 นาที"
              secondary="· 65 เมตร"
              steps={walkingSteps}
              footer={<LinkButton>แจ้งเจ้าหน้าที่หากหลงทาง</LinkButton>}
            />
          </div>
        </Specimen>
      </div>

      <SubHead>เมนู</SubHead>
      <div className="ds-grid ds-grid--wide">
        <Specimen name="BottomTabBar" note="เจ้าหน้าที่บนมือถือ · 4 ปลายทางเดียวกับบนเดสก์ท็อป" paper block>
          <div className="max-w-[390px] border border-line">
            <BottomTabBar items={staffNavItems} activeId={tab} onSelect={setTab} />
          </div>
        </Specimen>
        <Specimen name="SideRail" note="เจ้าหน้าที่บนเดสก์ท็อป · rail กว้าง 220px ตายตัว" paper block>
          <div className="flex h-[420px]">
            <SideRail
              items={staffRailItems}
              activeId={railItem}
              onSelect={setRailItem}
              role={staffRoleLabel(['STAFF'])}
            />
          </div>
        </Specimen>
      </div>
    </Section>
  )
}
