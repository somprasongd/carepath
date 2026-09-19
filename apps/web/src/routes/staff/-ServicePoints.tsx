import { useQuery } from '@tanstack/react-query'
import {
  Button,
  Card,
  DataTable,
  InfoNote,
  Meta,
  PageTitle,
  RouteStatusBadge,
  SectionTitle,
  Stack,
  ZoneChip,
  staffTitle,
  type Column,
} from '@/design-system'
import {
  ServicePointMappingCard,
  servicePointsQueryOptions,
  toServicePointView,
  type ServicePointView,
} from '@/features/servicepoint'
import { StaffShell } from './-StaffShell'

const columns: Column<ServicePointView>[] = [
  {
    key: 'service',
    header: 'จุดบริการ',
    render: (m) => <ZoneChip zone={m.zone}>{m.service}</ZoneChip>,
  },
  { key: 'name', header: 'ชื่อ', render: (m) => m.serviceName },
  { key: 'target', header: '→ ตำแหน่ง', render: (m) => m.target },
  {
    key: 'status',
    header: 'สถานะเส้นทาง',
    render: (m) => <RouteStatusBadge status={m.status} />,
  },
]

/**
 * เจ้าหน้าที่ · ผังจุดบริการ — the care-step → place link, read from the
 * service point API (#24/#42) so staff see the same mapping patients
 * navigate by. One mapping card per row on a phone, a real table from the
 * console breakpoint on. Read-only for the MVP (FR-11): mapping changes go
 * through seed migrations, so no edit/add controls yet. Care Graph and
 * Navigation Graph stay separate models (ADR-0002); this is where a human
 * joins them.
 */
export function ServicePoints() {
  const points = useQuery(servicePointsQueryOptions())
  const views = (points.data ?? []).map(toServicePointView)

  return (
    <StaffShell>
      <div className="mb-4 flex items-start justify-between @7xl:mb-5">
        <div>
          <PageTitle className={staffTitle}>ผังจุดบริการ</PageTitle>
          <Meta className="mt-1.5">
            {points.isLoading
              ? 'กำลังโหลดรายการ…'
              : points.isError
                ? 'โหลดรายการไม่สำเร็จ'
                : `${views.length} การเชื่อมโยงในระบบ`}
          </Meta>
        </div>
        <Button
          variant="ghost"
          onClick={() => void points.refetch()}
          disabled={points.isFetching}
        >
          {points.isFetching ? 'กำลังรีเฟรช…' : 'รีเฟรช'}
        </Button>
      </div>

      <div className="mb-4 max-w-[900px] @7xl:mb-5">
        {points.isError ? (
          <InfoNote>
            โหลดจุดบริการไม่สำเร็จ (สถานะ {points.error.status}) — ตรวจว่า API
            เปิดอยู่แล้วกดรีเฟรชอีกครั้ง
          </InfoNote>
        ) : views.length === 0 && !points.isLoading ? (
          <InfoNote>
            ยังไม่มีการเชื่อมโยงบริการกับตำแหน่งในระบบ — MVP นี้เพิ่มการเชื่อมโยงผ่าน
            seed data ของ service point
          </InfoNote>
        ) : (
          <InfoNote>
            ห้องที่ยังไม่มีโหนดนำทางในผังอาคารจะยังแสดงในหน้านี้
            แต่จะไม่แสดงเส้นทางให้ผู้ป่วยจนกว่าจะเพิ่มโหนด
          </InfoNote>
        )}
      </div>

      {/* Card list under the console breakpoint, a real table from it on. */}
      <div className="@7xl:hidden">
        <SectionTitle className="text-[14px]/[1.3] font-bold">รายการเชื่อมโยง</SectionTitle>
        <Stack>
          {views.map((view) => (
            <ServicePointMappingCard key={view.service} {...view} />
          ))}
        </Stack>
      </div>
      <Card padding="xl" className="hidden @7xl:block">
        <DataTable columns={columns} rows={views} rowKey={(m) => m.service} />
      </Card>
    </StaffShell>
  )
}
