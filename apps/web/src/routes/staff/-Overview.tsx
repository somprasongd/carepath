import {
  Card,
  DataTable,
  LoadBadge,
  Meta,
  PageTitle,
  RefPill,
  SectionTitle,
  StatCard,
  Stack,
  ZoneChip,
  ZoneLegend,
  staffSection,
  staffTitle,
  type Column,
} from '@/design-system'
import { AttentionCard } from '@/features/visit'
import { ServicePointRow } from '@/features/servicepoint'
import {
  attentionList,
  consoleDate,
  lastUpdated,
  servicePointLoad,
  type ServicePointLoad,
} from '@/mocks/demo-data'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import { StaffShell } from './-StaffShell'

const columns: Column<ServicePointLoad>[] = [
  { key: 'point', header: 'จุดบริการ', render: (sp) => <ZoneChip zone={sp.zone}>{sp.code}</ZoneChip> },
  { key: 'name', header: 'ชื่อ', render: (sp) => sp.name },
  {
    key: 'waiting',
    header: 'กำลังรอ',
    render: (sp) => (
      <span className={`font-code text-label-code ${sp.load === 'busy' ? 'text-primary' : 'text-ink'}`}>
        {sp.waiting}
      </span>
    ),
  },
  { key: 'average', header: 'เวลาเฉลี่ย', render: (sp) => sp.averageWait },
  { key: 'load', header: 'สถานะ', render: (sp) => <LoadBadge load={sp.load} /> },
]

/**
 * เจ้าหน้าที่ · ภาพรวม — one column of cards on a phone, a four-up KPI row
 * plus a real table from the ~1280px console breakpoint on (DESIGN.md).
 */
export function Overview() {
  const busiest = servicePointLoad.find((sp) => sp.load === 'busy')
  const { identity } = useStaffAuth()

  return (
    <StaffShell trailing={<RefPill>{identity?.displayName ?? 'เจ้าหน้าที่'}</RefPill>}>
      <div className="mb-0.5 flex items-start justify-between @7xl:mb-7">
        <div>
          <PageTitle className={staffTitle}>ภาพรวมการไหลของผู้ป่วยวันนี้</PageTitle>
          <Meta className="mt-1.5 @7xl:mt-1.5">
            อัปเดตล่าสุด {lastUpdated}
            <span className="hidden @7xl:inline"> · ข้อมูลจาก Mock HIS</span>
          </Meta>
        </div>
        <RefPill className="hidden @7xl:inline-flex">{consoleDate}</RefPill>
      </div>

      <div className="mt-5 mb-7 grid grid-cols-2 gap-3 @7xl:mt-0 @7xl:grid-cols-4 @7xl:gap-5">
        <StatCard label="กำลังใช้บริการ" value="24" unit="ราย" note="+4 จากเมื่อวานเวลานี้" />
        <StatCard
          textValue
          label="จุดที่รอมากสุด"
          value={busiest?.name ?? '—'}
          note={busiest ? `${busiest.waiting} คนในคิว` : undefined}
        />
        <StatCard label="เวลารอเฉลี่ย" value="11" unit="นาที" note="ใกล้เคียงค่าเฉลี่ยปกติ" />
        <StatCard tone="attention" label="ไม่ทราบตำแหน่ง" value="3" unit="ราย" note="ต้องตรวจสอบ" />
      </div>

      <div className="@7xl:grid @7xl:grid-cols-[1fr_360px] @7xl:items-start @7xl:gap-6">
        <Card padding="none" className="border-0 bg-transparent @7xl:border @7xl:bg-surface @7xl:p-5">
          <SectionTitle className={staffSection}>จุดบริการวันนี้</SectionTitle>

          {/* Card list under the console breakpoint, a real table from it on. */}
          <div className="mb-7 @7xl:hidden">
            {servicePointLoad.slice(0, 4).map((sp) => (
              <ServicePointRow
                key={sp.code}
                zone={sp.zone}
                code={sp.code}
                name={sp.name}
                waiting={sp.waiting}
                busy={sp.load === 'busy'}
              />
            ))}
          </div>
          <div className="hidden @7xl:block">
            <DataTable columns={columns} rows={servicePointLoad} rowKey={(sp) => sp.code} />
          </div>
        </Card>

        <Stack lg>
          <Card padding="none" className="border-0 bg-transparent @7xl:border @7xl:bg-surface @7xl:p-5">
            <SectionTitle className={staffSection}>ผู้ป่วยที่ต้องช่วยเหลือ</SectionTitle>
            <Stack>
              {attentionList.map((item) => (
                <AttentionCard key={item.visitRef} {...item} />
              ))}
            </Stack>
          </Card>

          <Card padding="xl" className="hidden @7xl:block">
            <SectionTitle>ผังสีของโซน</SectionTitle>
            <ZoneLegend only={['public', 'opd', 'diagnostic', 'pharmacy']} />
            <div className="mt-3 border-t border-line pt-3">
              <Meta>สีเดียวกับผังอาคารที่ผู้ป่วยเห็นบนหน้าจอนำทาง</Meta>
            </div>
          </Card>
        </Stack>
      </div>
    </StaffShell>
  )
}
