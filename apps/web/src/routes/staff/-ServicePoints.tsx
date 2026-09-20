import { useState } from 'react'
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
import { anchorQrPayload, floorLabelFor, wayfindingAnchors } from '@/features/floorplan'
import { lookup, messagesFor } from '@/i18n'
import {
  ServicePointMappingCard,
  locationQrPayload,
  qrModules,
  servicePointsQueryOptions,
  toServicePointView,
  type ServicePoint,
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
  const [qrOpen, setQrOpen] = useState(false)

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
        <div className="flex gap-2">
          <Button variant="ghost" onClick={() => setQrOpen(true)}>
            QR จุดบริการ
          </Button>
          <Button
            variant="ghost"
            onClick={() => void points.refetch()}
            disabled={points.isFetching}
          >
            {points.isFetching ? 'กำลังรีเฟรช…' : 'รีเฟรช'}
          </Button>
        </div>
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

      {qrOpen && <QrSheet points={points.data ?? []} onClose={() => setQrOpen(false)} />}
    </StaffShell>
  )
}

/**
 * The printable sticker sheet (FR: staff open each point's location QR):
 * one card per scannable point — the QR patients scan at that point, plus
 * the place name/floor/payload a sticker must carry to be identifiable.
 * Two groups: service places (1 sticker per place — clinics sharing a nurse
 * station share it) and wayfinding anchors (lifts, stairs, entrances —
 * points patients stand at that are not places; their QR carries a bare
 * node reference). Both list only points with an entry node / graph node,
 * the same rule as the patient's pick list.
 */
function QrSheet({ points, onClose }: { points: ServicePoint[]; onClose: () => void }) {
  // One QR blown up full-screen — the demo path: hold a phone up to scan and
  // the combined sheet's neighbours must not compete for the camera.
  const [focused, setFocused] = useState<{ payload: string; title: string; subtitle: string } | null>(
    null,
  )

  const places = new Map<string, { placeId: string; name: string; floorCode: string }>()
  for (const sp of points) {
    const place = sp.place
    if (!place?.entryNodeId || places.has(place.id)) continue
    places.set(place.id, { placeId: place.id, name: place.name, floorCode: place.floor.code })
  }
  const anchors = wayfindingAnchors()
  // Staff console pins Thai (ADR-0012 §2): anchor kind names read straight
  // from the Thai catalog, falling back to the raw kind for unmapped nodes.
  const anchorLabel = (kind: string) => lookup(messagesFor('th'), `anchor.kind.${kind}`) ?? kind

  return (
    <div className="qr-sheet fixed inset-0 z-50 overflow-y-auto bg-ink/40 backdrop-blur-[2px]">
      <div className="mx-auto flex min-h-full max-w-5xl flex-col gap-4 bg-background px-gutter py-6">
        {focused ? (
          <>
            <div className="flex items-start justify-between gap-3 print:hidden">
              <div>
                <SectionTitle>{focused.title}</SectionTitle>
                <Meta className="mt-1">{focused.subtitle} · เอามือถือสแกน QR นี้ได้เลย</Meta>
              </div>
              <div className="flex gap-2">
                <Button variant="ghost" onClick={() => setFocused(null)}>
                  กลับหน้ารวม
                </Button>
                <Button variant="ghost" onClick={onClose}>
                  ปิด
                </Button>
              </div>
            </div>
            <div className="flex flex-1 items-center justify-center py-6">
              <QrCard payload={focused.payload} title={focused.title} subtitle={focused.subtitle} size="xl" />
            </div>
          </>
        ) : (
          <>
            <div className="flex items-start justify-between gap-3 print:hidden">
              <div>
                <SectionTitle>QR จุดบริการ — พิมพ์ติดหน้าจุดบริการ</SectionTitle>
                <Meta className="mt-1">
                  ผู้ป่วยสแกนแล้วระบบจะถือว่าอยู่ที่จุดนั้น · 1 สติกเกอร์ต่อ 1 สถานที่ · กดที่การ์ดเพื่อขยายทีละอัน
                </Meta>
              </div>
              <div className="flex gap-2">
                <Button variant="ghost" onClick={() => window.print()}>
                  พิมพ์
                </Button>
                <Button variant="ghost" onClick={onClose}>
                  ปิด
                </Button>
              </div>
            </div>

            <div className="grid grid-cols-[repeat(auto-fill,minmax(210px,1fr))] gap-4">
              {Array.from(places.values()).map((place) => (
                <QrCard
                  key={place.placeId}
                  payload={locationQrPayload(place.placeId)}
                  title={place.name}
                  subtitle={`ชั้น ${place.floorCode} · ${place.placeId}`}
                  onOpen={() =>
                    setFocused({
                      payload: locationQrPayload(place.placeId),
                      title: place.name,
                      subtitle: `ชั้น ${place.floorCode} · ${place.placeId}`,
                    })
                  }
                />
              ))}
            </div>

            <SectionTitle className="text-[14px]/[1.3] font-bold">
              จุดอ้างอิงบนแผนที่ — ลิฟต์ บันได ทางเข้า
            </SectionTitle>
            <div className="grid grid-cols-[repeat(auto-fill,minmax(210px,1fr))] gap-4">
              {anchors.map((anchor) => (
                <QrCard
                  key={anchor.nodeId}
                  payload={anchorQrPayload(anchor)}
                  title={anchorLabel(anchor.kind)}
                  subtitle={`${floorLabelFor(anchor.floorId, 'th')} · ${anchor.nodeId}`}
                  onOpen={() =>
                    setFocused({
                      payload: anchorQrPayload(anchor),
                      title: anchorLabel(anchor.kind),
                      subtitle: `${floorLabelFor(anchor.floorId, 'th')} · ${anchor.nodeId}`,
                    })
                  }
                />
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  )
}

function QrCard({
  payload,
  title,
  subtitle,
  size = 'md',
  onOpen,
}: {
  payload: string
  title: string
  subtitle: string
  /** `xl` is the focused demo view — one QR filling most of the screen. */
  size?: 'md' | 'xl'
  onOpen?: () => void
}) {
  const { path, moduleCount } = qrModules(payload)

  return (
    <Card
      radius="md"
      padding={size === 'xl' ? 'xl' : 'md'}
      className={`flex flex-col items-center gap-2.5 ${onOpen ? 'cursor-pointer hover:border-primary' : ''}`}
      onClick={onOpen}
      onKeyDown={
        onOpen
          ? (event) => {
              if (event.key === 'Enter' || event.key === ' ') onOpen()
            }
          : undefined
      }
      role={onOpen ? 'button' : undefined}
      tabIndex={onOpen ? 0 : undefined}
    >
      <svg
        viewBox={`-4 -4 ${moduleCount + 8} ${moduleCount + 8}`}
        className={size === 'xl' ? 'h-[min(68vh,68vw)] w-[min(68vh,68vw)]' : 'h-44 w-44'}
        shapeRendering="crispEdges"
        role="img"
        aria-label={`QR ${title}`}
      >
        <rect x={-4} y={-4} width={moduleCount + 8} height={moduleCount + 8} fill="var(--surface)" />
        <path d={path} fill="var(--ink)" />
      </svg>
      <div className="text-center">
        <div className="font-sans text-body-sm font-bold text-ink">{title}</div>
        <div className="font-sans text-caption text-ink-muted">{subtitle}</div>
        <div className="font-code text-caption text-ink-muted">{payload}</div>
      </div>
    </Card>
  )
}
