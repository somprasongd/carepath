import {
  Button,
  Card,
  DataTable,
  LoadBadge,
  Meta,
  PageTitle,
  RefPill,
  SectionTitle,
  StatCard,
  ZoneChip,
  ZoneLegend,
  staffSection,
  staffTitle,
  type Column,
} from '@/design-system'
import { ServicePointRow } from '@/features/servicepoint'
import {
  emptyOverviewCards,
  toOverviewView,
  useAnalyticsOverview,
  type ServicePointLoadView,
} from '@/features/analytics'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import { StaffShell } from './-StaffShell'

const columns: Column<ServicePointLoadView>[] = [
  { key: 'point', header: 'จุดบริการ', render: (sp) => <ZoneChip zone={sp.zone}>{sp.code}</ZoneChip> },
  { key: 'name', header: 'ชื่อ', render: (sp) => sp.name },
  {
    key: 'waiting',
    header: 'กำลังรอ',
    render: (sp) => (
      <span className={`font-code text-label-code ${sp.busy ? 'text-primary' : 'text-ink'}`}>
        {sp.waitingNow}
      </span>
    ),
  },
  {
    key: 'longest',
    header: 'รอนานสุด',
    render: (sp) => (
      <span className={`font-code text-label-code ${sp.busy ? 'text-primary' : 'text-ink'}`}>
        {sp.longestWaiting}
      </span>
    ),
  },
  { key: 'average', header: 'เวลารอเฉลี่ย', render: (sp) => sp.avgWait },
  { key: 'load', header: 'สถานะ', render: (sp) => <LoadBadge load={sp.busy ? 'busy' : 'normal'} /> },
]

/**
 * ภาพรวม · ผู้บริหาร/เจ้าหน้าที่ — the live aggregate from #86's API, not a
 * mock. One column of cards on a phone, a four-up KPI row plus a real table
 * from the ~1280px console breakpoint on (DESIGN.md). Pending renders the
 * full frame with em dashes (never a blank flash or a fake zero); a failure
 * renders one honest Thai sentence and a retry — no status codes (NFR-10).
 */
export function Overview() {
  const { identity } = useStaffAuth()
  const { data, isPending, isError, refetch } = useAnalyticsOverview()
  const view = data ? toOverviewView(data, 'th') : null
  const cards = view?.cards ?? emptyOverviewCards

  return (
    <StaffShell trailing={<RefPill>{identity?.displayName ?? 'เจ้าหน้าที่'}</RefPill>}>
      <div className="mb-0.5 flex items-start justify-between @7xl:mb-7">
        <div>
          <PageTitle className={staffTitle}>ภาพรวมการไหลของผู้ป่วยวันนี้</PageTitle>
          <Meta className="mt-1.5 @7xl:mt-1.5">
            {isPending ? 'กำลังโหลดข้อมูล…' : `อัปเดตล่าสุด ${view?.updatedAt ?? '—'}`}
            <span className="hidden @7xl:inline"> · รวมจาก timeline การรักษาจริง</span>
          </Meta>
        </div>
        <RefPill className="hidden @7xl:inline-flex">ข้อมูลวันนี้</RefPill>
      </div>

      {isError ? (
        <Card padding="xl" className="mt-5 mb-7">
          <SectionTitle>โหลดภาพรวมไม่ได้</SectionTitle>
          <Meta>เชื่อมต่อกับเซิร์ฟเวอร์ไม่สำเร็จ — ตรวจการเชื่อมต่อแล้วลองอีกครั้ง</Meta>
          <Button variant="secondary" className="mt-4" onClick={() => void refetch()}>
            ลองใหม่
          </Button>
        </Card>
      ) : (
        <>
          <div className="mt-5 mb-7 grid grid-cols-2 gap-3 @7xl:mt-0 @7xl:grid-cols-4 @7xl:gap-5">
            <StatCard label="กำลังใช้บริการ" value={cards.activeVisits} unit="ราย" />
            <StatCard
              textValue
              label="จุดที่รอมากสุด"
              value={cards.bottleneckName}
              note={cards.bottleneckNote}
            />
            <StatCard
              label="เวลารอเฉลี่ย"
              value={cards.avgWait}
              unit={cards.avgWait === '—' ? undefined : 'นาที'}
              note="เฉลี่ยจากผู้ป่วยที่ได้รับบริการแล้ว"
            />
            {/* The average only counts already-served patients — this live
                number is its honest companion while people still queue. */}
            <StatCard
              tone="attention"
              label="รอนานสุดตอนนี้"
              value={cards.longestWaiting}
              unit={cards.longestWaiting === '—' ? undefined : 'นาที'}
              note={cards.longestWaitingNote}
            />
          </div>

          <div className="@7xl:grid @7xl:grid-cols-[1fr_360px] @7xl:items-start @7xl:gap-6">
            <Card padding="none" className="border-0 bg-transparent @7xl:border @7xl:bg-surface @7xl:p-5">
              <SectionTitle className={staffSection}>จุดบริการวันนี้</SectionTitle>

              {/* Card list under the console breakpoint, a real table from it on. */}
              <div className="mb-7 @7xl:hidden">
                {(view?.rows ?? []).slice(0, 4).map((sp) => (
                  <ServicePointRow
                    key={sp.servicePointId}
                    zone={sp.zone}
                    code={sp.code}
                    name={sp.name}
                    waiting={sp.waitingNow}
                    busy={sp.busy}
                  />
                ))}
              </div>
              <div className="hidden @7xl:block">
                <DataTable columns={columns} rows={view?.rows ?? []} rowKey={(sp) => sp.servicePointId} />
              </div>
            </Card>

            <Card padding="xl" className="hidden @7xl:block">
              <SectionTitle>ผังสีของโซน</SectionTitle>
              <ZoneLegend only={['public', 'opd', 'diagnostic', 'pharmacy']} />
              <div className="mt-3 border-t border-line pt-3">
                <Meta>สีเดียวกับผังอาคารที่ผู้ป่วยเห็นบนหน้าจอนำทาง</Meta>
              </div>
            </Card>
          </div>
        </>
      )}
    </StaffShell>
  )
}
