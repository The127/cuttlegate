import { useState, type ReactNode } from 'react'

export interface ColumnDef<T> {
  /** Unique key for this column — used as React key and sort state key */
  key: string
  /** Column header label */
  header: string
  /** Render the cell value for a given row */
  cell: (row: T) => ReactNode
  /** If true, this column can be sorted client-side */
  sortable?: boolean
  /** Optional: derive a primitive sort value from a row (defaults to stringified cell output) */
  sortValue?: (row: T) => string | number
}

type SortDir = 'asc' | 'desc'

interface SortState {
  key: string
  dir: SortDir
}

export interface DataTableProps<T> {
  columns: ColumnDef<T>[]
  data: T[]
  /** Rendered when data is empty */
  emptyState?: ReactNode
  /** Called when the user clicks a data row */
  onRowClick?: (row: T) => void
  /** Accessible label for the table element */
  'aria-label'?: string
}

function sortRows<T>(data: T[], columns: ColumnDef<T>[], sort: SortState): T[] {
  const col = columns.find((c) => c.key === sort.key)
  if (!col) return data

  return [...data].sort((a, b) => {
    const av = col.sortValue ? col.sortValue(a) : String(col.cell(a))
    const bv = col.sortValue ? col.sortValue(b) : String(col.cell(b))
    const cmp = av < bv ? -1 : av > bv ? 1 : 0
    return sort.dir === 'asc' ? cmp : -cmp
  })
}

/**
 * DataTable — generic typed data table with client-side sort and empty state.
 *
 * Uses role="table" with standard Tab focus order. Arrow-key grid navigation
 * is out of scope — this is a display table, not an editable grid.
 */
export function DataTable<T>({
  columns,
  data,
  emptyState,
  onRowClick,
  'aria-label': ariaLabel,
}: DataTableProps<T>) {
  const [sort, setSort] = useState<SortState | null>(null)

  function toggleSort(key: string) {
    setSort((prev) => {
      if (prev?.key === key) {
        return { key, dir: prev.dir === 'asc' ? 'desc' : 'asc' }
      }
      return { key, dir: 'asc' }
    })
  }

  const rows = sort ? sortRows(data, columns, sort) : data

  return (
    <div className="w-full overflow-x-auto">
      <table
        role="table"
        aria-label={ariaLabel}
        className="w-full text-sm text-left border-collapse"
      >
        <thead>
          <tr className="border-b border-[var(--color-border)] bg-[var(--color-surface-elevated)]">
            {columns.map((col) => (
              <th
                key={col.key}
                scope="col"
                aria-sort={
                  col.sortable
                    ? sort?.key === col.key
                      ? sort.dir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : 'none'
                    : undefined
                }
                className="px-4 py-2.5 text-xs font-medium text-[var(--color-text-secondary)] font-medium whitespace-nowrap"
              >
                {col.sortable ? (
                  <button
                    type="button"
                    onClick={() => toggleSort(col.key)}
                    className="inline-flex items-center gap-1 hover:text-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
                  >
                    {col.header}
                    <SortIcon active={sort?.key === col.key} dir={sort?.dir} />
                  </button>
                ) : col.header ? (
                  col.header
                ) : (
                  // Empty header cell — screen-reader label for accessibility
                  <span className="sr-only">Actions</span>
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-8 text-center">
                {emptyState ?? (
                  <span className="text-sm text-[var(--color-text-muted)]">No data</span>
                )}
              </td>
            </tr>
          ) : (
            rows.map((row, idx) => (
              <tr
                key={idx}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                onKeyDown={
                  onRowClick
                    ? (e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          onRowClick(row)
                        }
                      }
                    : undefined
                }
                tabIndex={onRowClick ? 0 : undefined}
                className={`border-b border-[var(--color-border)] last:border-0 ${
                  onRowClick
                    ? 'cursor-pointer hover:bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-inset focus:ring-[var(--color-accent)]'
                    : ''
                }`}
              >
                {columns.map((col) => (
                  <td key={col.key} className="px-4 py-3">
                    {col.cell(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

function SortIcon({
  active,
  dir,
}: {
  active: boolean | undefined
  dir: SortDir | undefined
}) {
  return (
    <svg
      className={`w-3 h-3 ${active ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}`}
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
    >
      {active && dir === 'asc' ? (
        // Up arrow
        <path d="M8 3l4 6H4l4-6z" />
      ) : active && dir === 'desc' ? (
        // Down arrow
        <path d="M8 13L4 7h8l-4 6z" />
      ) : (
        // Up+down stacked (unsorted)
        <>
          <path d="M8 3l3 4.5H5L8 3z" />
          <path d="M8 13l-3-4.5h6L8 13z" />
        </>
      )}
    </svg>
  )
}
