import type { ReactNode } from 'react'

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
}

/**
 * Staff desktop only — the responsive counterpart of the mobile card list, not
 * a separate design. Tabular data deserves a real table once there is room.
 */
export function DataTable<Row>({ columns, rows, rowKey, caption }: DataTableProps<Row>) {
  return (
    <table className="cp-table">
      {caption && <caption className="cp-meta">{caption}</caption>}
      <thead>
        <tr>
          {columns.map((col) => (
            <th key={col.key} style={col.width ? { width: col.width } : undefined} scope="col">
              {col.header}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={rowKey(row)}>
            {columns.map((col) => (
              <td key={col.key}>{col.render(row)}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

/** Code + secondary name, the standard first cell of a staff table. */
export function TableCode({ code, sub }: { code: string; sub?: string }) {
  return (
    <>
      <div className="cp-table__code">{code}</div>
      {sub && <div className="cp-table__sub">{sub}</div>}
    </>
  )
}
