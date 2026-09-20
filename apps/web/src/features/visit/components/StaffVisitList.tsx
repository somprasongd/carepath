import { Card, DataTable, Meta, RefPill, TableCode, type Column } from '@/design-system'
import { cn } from 'cn'
import type { Journey } from '../queries'
import { StaffStatusBadge } from './StaffStatusBadge'
import { stepProgress, syncedAtLabel, visitPosition } from '../staff'

const columns: Column<Journey>[] = [
  {
    key: 'visit',
    header: 'Visit',
    render: (journey) => <TableCode code={journey.visitId} sub={journey.patientRef} />,
  },
  { key: 'position', header: 'ตำแหน่งปัจจุบัน', render: (journey) => visitPosition(journey) },
  { key: 'progress', header: 'ขั้นที่เสร็จ', render: (journey) => stepProgress(journey) },
  {
    key: 'synced',
    header: 'ซิงก์ล่าสุด',
    render: (journey) => (
      <span className="font-code">{syncedAtLabel(journey.syncedAt, 'th')}</span>
    ),
  },
  {
    key: 'status',
    header: 'สถานะ',
    render: (journey) => <StaffStatusBadge status={journey.status} kind="visit" />,
  },
]

/**
 * The visit monitor's selectable list (#37 AC1): one row per projected
 * journey with the patient's position at a glance. The selected row carries
 * the primary border — the one accent a staff screen may use. A phone gets
 * the card stack; the console breakpoint gets a real table (DESIGN.md's
 * "tabular data deserves a table once there is room"), same selection.
 */
export function StaffVisitList({
  journeys,
  selectedId,
  onSelect,
}: {
  journeys: Journey[]
  selectedId: string | null
  onSelect: (visitId: string) => void
}) {
  return (
    <>
      <Card radius="md" padding="none" className="divide-y divide-line overflow-hidden @7xl:hidden">
        {journeys.map((journey) => {
          const selected = journey.visitId === selectedId
          return (
            <button
              key={journey.visitId}
              type="button"
              onClick={() => onSelect(journey.visitId)}
              aria-current={selected}
              className={cn(
                'block w-full cursor-pointer border-0 border-l-2 bg-none p-3 text-left',
                selected ? 'border-l-primary bg-primary-tint' : 'border-l-transparent bg-surface',
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <RefPill>{journey.visitId}</RefPill>
                <StaffStatusBadge status={journey.status} kind="visit" />
              </div>
              <div className="mt-2 font-sans text-body-sm font-semibold text-ink">
                {visitPosition(journey)}
              </div>
              <div className="mt-1 flex items-center justify-between gap-2">
                <Meta className="m-0">
                  {journey.patientRef} · ขั้นที่เสร็จ {stepProgress(journey)}
                </Meta>
                <Meta className="m-0 font-code">{syncedAtLabel(journey.syncedAt, 'th')}</Meta>
              </div>
            </button>
          )
        })}
      </Card>

      <Card radius="md" padding="lg" className="hidden @7xl:block">
        <DataTable
          columns={columns}
          rows={journeys}
          rowKey={(journey) => journey.visitId}
          onRowClick={(journey) => onSelect(journey.visitId)}
          isRowSelected={(journey) => journey.visitId === selectedId}
        />
      </Card>
    </>
  )
}
