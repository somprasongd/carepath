import {
  AttentionCard,
  Card,
  DataTable,
  LoadBadge,
  Meta,
  PageTitle,
  RefPill,
  Screen,
  ScreenMain,
  SectionTitle,
  SideRail,
  StatCard,
  TableCode,
  ZoneChip,
  ZoneLegend,
  type Column,
} from '@/design-system'
import {
  attentionList,
  consoleDate,
  lastUpdated,
  servicePointLoad,
  type ServicePointLoad,
} from '@/mocks/demo-data'
import { staffRailItems, staffRole, useStaffNav } from './-nav'

const columns: Column<ServicePointLoad>[] = [
  {
    key: 'point',
    header: 'จุดบริการ',
    render: (sp) => <TableCode code={sp.code} sub={sp.name} />,
  },
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
  { key: 'average', header: 'เวลาเฉลี่ย', render: (sp) => sp.averageWait },
  { key: 'load', header: 'สถานะ', render: (sp) => <LoadBadge load={sp.load} /> },
]

/**
 * เจ้าหน้าที่ · ภาพรวม (เดสก์ท็อป) — the same content as the mobile dashboard,
 * reorganised: fixed rail, four-up KPIs, and the list as a real table.
 */
export function OverviewDesktop() {
  const nav = useStaffNav()
  const busiest = servicePointLoad.find((sp) => sp.load === 'busy')

  return (
    <Screen variant="desktop">
      <SideRail items={staffRailItems} {...nav} role={staffRole} />

      <ScreenMain>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            marginBottom: 28,
          }}
        >
          <div>
            <PageTitle>ภาพรวมการไหลของผู้ป่วยวันนี้</PageTitle>
            <div style={{ marginTop: 6 }}>
              <Meta>อัปเดตล่าสุด {lastUpdated} · ข้อมูลจาก Mock HIS</Meta>
            </div>
          </div>
          <RefPill>{consoleDate}</RefPill>
        </div>

        <div className="cp-grid-4" style={{ marginBottom: 'var(--cp-space-3xl)' }}>
          <StatCard label="กำลังใช้บริการ" value="24" note="+4 จากเมื่อวานเวลานี้" />
          <StatCard
            textValue
            label="จุดที่รอมากที่สุด"
            value={busiest?.name ?? '—'}
            note={busiest ? `${busiest.code} · ${busiest.waiting} คนในคิว` : undefined}
          />
          <StatCard label="เวลารอเฉลี่ย" value="11" unit="นาที" note="ใกล้เคียงค่าเฉลี่ยปกติ" />
          <StatCard tone="attention" label="ไม่ทราบตำแหน่ง" value="3" unit="ราย" note="ต้องตรวจสอบ" />
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 360px',
            gap: 'var(--cp-space-2xl)',
            alignItems: 'start',
          }}
        >
          <Card padding="xl">
            <SectionTitle>จุดบริการวันนี้</SectionTitle>
            <DataTable columns={columns} rows={servicePointLoad} rowKey={(sp) => sp.code} />
          </Card>

          <div className="cp-stack cp-stack--lg">
            <Card padding="xl">
              <SectionTitle>ผู้ป่วยที่ต้องช่วยเหลือ</SectionTitle>
              <div className="cp-stack">
                {attentionList.map((item) => (
                  <AttentionCard key={item.visitRef} {...item} />
                ))}
              </div>
            </Card>

            <Card padding="xl">
              <SectionTitle>ผังสีของโซน</SectionTitle>
              <ZoneLegend only={['public', 'opd', 'diagnostic', 'pharmacy']} />
              <div
                style={{
                  marginTop: 'var(--cp-space-md)',
                  paddingTop: 'var(--cp-space-md)',
                  borderTop: '1px solid var(--cp-line)',
                }}
              >
                <Meta>สีเดียวกับผังอาคารที่ผู้ป่วยเห็นบนหน้าจอนำทาง</Meta>
              </div>
            </Card>
          </div>
        </div>
      </ScreenMain>
    </Screen>
  )
}
