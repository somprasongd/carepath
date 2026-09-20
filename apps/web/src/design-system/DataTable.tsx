import type { KeyboardEvent, ReactNode } from 'react'
import { cn } from 'cn'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

export type Column<Row> = {
  key: string
  header: ReactNode
  width?: string
  render: (row: Row) => ReactNode
}

export type DataTableProps<Row> = {
  columns: Column<Row>[]
  rows: Row[]
  rowKey: (row: Row) => string
  caption?: string
  /** Rows become clickable/keyboard-actionable (Enter/Space) — the desktop counterpart of a selectable card list. */
  onRowClick?: (row: Row) => void
  isRowSelected?: (row: Row) => boolean
}

/**
 * Staff desktop only — the responsive counterpart of the mobile card list, not
 * a separate design. Tabular data deserves a real table once there is room.
 *
 * Built on shadcn's Table, minus its row hover: DESIGN.md wants this reading
 * like a printed directory board, so nothing here lights up under the cursor
 * — unless `onRowClick` makes the row a selectable target, in which case it
 * gets the same hover/selected treatment as the mobile card it stands in for.
 */
export function DataTable<Row>({
  columns,
  rows,
  rowKey,
  caption,
  onRowClick,
  isRowSelected,
}: DataTableProps<Row>) {
  return (
    <Table className="border-collapse">
      {caption && (
        <caption className="caption-bottom font-sans text-caption text-ink-muted">
          {caption}
        </caption>
      )}
      <TableHeader>
        <TableRow className="border-line hover:bg-transparent">
          {columns.map((col) => (
            <TableHead
              key={col.key}
              scope="col"
              style={col.width ? { width: col.width } : undefined}
              className="h-auto px-0 py-3 text-left font-sans text-caption font-semibold text-ink-muted"
            >
              {col.header}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => {
          const selected = isRowSelected?.(row) ?? false
          const activate = () => onRowClick?.(row)
          const onKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
            if (event.key !== 'Enter' && event.key !== ' ') return
            event.preventDefault()
            activate()
          }
          return (
            <TableRow
              key={rowKey(row)}
              className={cn(
                'border-line',
                onRowClick ? 'cursor-pointer hover:bg-primary-tint' : 'hover:bg-transparent',
                selected && 'bg-primary-tint',
              )}
              onClick={onRowClick ? activate : undefined}
              onKeyDown={onRowClick ? onKeyDown : undefined}
              tabIndex={onRowClick ? 0 : undefined}
              role={onRowClick ? 'button' : undefined}
              aria-pressed={onRowClick ? selected : undefined}
            >
              {columns.map((col) => (
                <TableCell
                  key={col.key}
                  className="px-0 py-4 align-middle font-sans text-body-sm text-ink"
                >
                  {col.render(row)}
                </TableCell>
              ))}
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

/** Code + secondary name, the standard first cell of a staff table. */
export function TableCode({ code, sub }: { code: string; sub?: string }) {
  return (
    <>
      <div className="font-code text-[14px] font-bold text-ink">{code}</div>
      {sub && <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">{sub}</div>}
    </>
  )
}
