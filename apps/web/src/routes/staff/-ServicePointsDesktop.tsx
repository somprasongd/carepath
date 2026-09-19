import {
  Button,
  Card,
  DataTable,
  InfoNote,
  Meta,
  PageTitle,
  RouteStatusBadge,
  Screen,
  ScreenMain,
  SideRail,
  TableCode,
  ZoneChip,
  type Column,
} from '@/design-system'
import { servicePointMappings, type ServicePointMapping } from '@/mocks/demo-data'
import { staffRailItems, staffRole, useStaffNav } from './-nav'

const columns: Column<ServicePointMapping>[] = [
  {
    key: 'service',
    header: 'จุดบริการ',
    render: (m) => <TableCode code={m.service} sub={m.serviceName} />,
  },
  {
    key: 'target',
    header: '→ ตำแหน่ง',
    render: (m) => (
      <>
        {m.target}
        {m.note && <div className="cp-table__sub">{m.note}</div>}
      </>
    ),
  },
  { key: 'zone', header: 'โซน', render: (m) => <ZoneChip zone={m.zone} /> },
  {
    key: 'status',
    header: 'สถานะเส้นทาง',
    render: (m) => <RouteStatusBadge status={m.status} />,
  },
  {
    key: 'edit',
    header: '',
    render: () => (
      <button type="button" className="cp-link">
        แก้ไข
      </button>
    ),
  },
]

/**
 * เจ้าหน้าที่ · ผังจุดบริการ (เดสก์ท็อป) — the care-step → place mapping table.
 * Care Graph and Navigation Graph stay separate models (ADR-0002); this screen
 * is where a human joins them.
 */
export function ServicePointsDesktop() {
  const nav = useStaffNav()

  return (
    <Screen variant="desktop">
      <SideRail items={staffRailItems} {...nav} role={staffRole} />

      <ScreenMain>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            marginBottom: 'var(--cp-space-xl)',
          }}
        >
          <div>
            <PageTitle>ผังจุดบริการ</PageTitle>
            <div style={{ marginTop: 6 }}>
              <Meta>
                เชื่อมโยงบริการทางคลินิกกับตำแหน่งจริงในอาคาร — แยกอิสระจากผังการดูแลผู้ป่วย
              </Meta>
            </div>
          </div>
          <Button variant="secondary">+ เพิ่มการเชื่อมโยง</Button>
        </div>

        <div style={{ maxWidth: 900, marginBottom: 'var(--cp-space-2xl)' }}>
          <InfoNote>
            ห้องที่ยังไม่มีโหนดนำทางในผังอาคารจะยังเลือกได้ในหน้านี้
            แต่จะไม่แสดงเส้นทางให้ผู้ป่วยจนกว่าจะเพิ่มโหนด
          </InfoNote>
        </div>

        <Card padding="xl">
          <DataTable
            columns={columns}
            rows={servicePointMappings}
            rowKey={(m) => m.service}
          />
        </Card>
      </ScreenMain>
    </Screen>
  )
}
