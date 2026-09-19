import {
  Button,
  Card,
  DataTable,
  InfoNote,
  LinkButton,
  Meta,
  PageTitle,
  RouteStatusBadge,
  SectionTitle,
  Stack,
  ZoneChip,
  staffTitle,
  type Column,
} from '@/design-system'
import { ServicePointMappingCard } from '@/features/servicepoint'
import { servicePointMappings, type ServicePointMapping } from '@/mocks/demo-data'
import { StaffShell } from './-StaffShell'

const columns: Column<ServicePointMapping>[] = [
  {
    key: 'service',
    header: 'จุดบริการ',
    render: (m) => <ZoneChip zone={m.zone}>{m.service}</ZoneChip>,
  },
  { key: 'name', header: 'ชื่อ', render: (m) => m.serviceName },
  {
    key: 'target',
    header: '→ ตำแหน่ง',
    render: (m) => (
      <>
        {m.target}
        {m.note && <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">{m.note}</div>}
      </>
    ),
  },
  {
    key: 'status',
    header: 'สถานะเส้นทาง',
    render: (m) => <RouteStatusBadge status={m.status} />,
  },
  {
    key: 'edit',
    header: '',
    render: () => <LinkButton>แก้ไข</LinkButton>,
  },
]

/**
 * เจ้าหน้าที่ · ผังจุดบริการ — the care-step → place link. One mapping card
 * per row on a phone, a real table with an add-link action from the console
 * breakpoint on. Care Graph and Navigation Graph stay separate models
 * (ADR-0002); this is where a human joins them.
 */
export function ServicePoints() {
  return (
    <StaffShell>
      <div className="mb-4 flex items-start justify-between @7xl:mb-5">
        <div>
          <PageTitle className={staffTitle}>ผังจุดบริการ</PageTitle>
          <Meta className="mt-1.5">
            เชื่อมโยงบริการทางคลินิกกับตำแหน่งจริงในอาคาร
            <span className="hidden @7xl:inline"> — แยกอิสระจากผังการดูแลผู้ป่วย</span>
          </Meta>
        </div>
        <Button variant="secondary" className="hidden @7xl:inline-flex">
          + เพิ่มการเชื่อมโยง
        </Button>
      </div>

      <div className="mb-4 max-w-[900px] @7xl:mb-5">
        <InfoNote>
          ห้องที่ยังไม่มีโหนดนำทางในผังอาคารจะยังเลือกได้ในหน้านี้
          แต่จะไม่แสดงเส้นทางให้ผู้ป่วยจนกว่าจะเพิ่มโหนด
        </InfoNote>
      </div>

      {/* Card list under the console breakpoint, a real table from it on. */}
      <div className="@7xl:hidden">
        <SectionTitle className="text-[14px]/[1.3] font-bold">รายการเชื่อมโยง</SectionTitle>
        <Stack>
          {servicePointMappings.map((mapping) => (
            <ServicePointMappingCard key={mapping.service} {...mapping} />
          ))}
        </Stack>
      </div>
      <Card padding="xl" className="hidden @7xl:block">
        <DataTable columns={columns} rows={servicePointMappings} rowKey={(m) => m.service} />
      </Card>
    </StaffShell>
  )
}
