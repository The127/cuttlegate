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

export type SortDir = 'asc' | 'desc'

export interface SortState {
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

  // --- Server-mode props (optional) ---
  // When these are provided, sorting and pagination are delegated to the parent.

  /** If true, shows a loading overlay on the table body */
  loading?: boolean
  /** Total number of rows across all pages (for pagination display) */
  totalRows?: number
  /** Current page number (1-based) */
  currentPage?: number
  /** Rows per page */
  perPage?: number
  /** Called when the user changes page */
  onPageChange?: (page: number) => void
  /** Called when the user clicks a sortable column header in server mode */
  onSortChange?: (sort: SortState) => void
  /** Current server-side sort state (used for header indicators in server mode) */
  serverSort?: SortState | null
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
 * DataTable — generic typed data table with client-side or server-side sort,
 * optional pagination, and empty state.
 *
 * **Client mode (default):** sorting and display are handled locally.
 * **Server mode:** when `onSortChange`, `onPageChange`, and `totalRows` are provided,
 * the table delegates sorting/pagination to the parent and renders pagination controls.
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
  loading,
  totalRows,
  currentPage,
  perPage,
  onPageChange,
  onSortChange,
  serverSort,
}: DataTableProps<T>) {
  const isServerMode = onSortChange !== undefined || onPageChange !== undefined
  const [clientSort, setClientSort] = useState<SortState | null>(null)

  const sort = isServerMode ? (serverSort ?? null) : clientSort

  function toggleSort(key: string) {
    const newSort: SortState =
      sort?.key === key
        ? { key, dir: sort.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: 'asc' }

    if (isServerMode && onSortChange) {
      onSortChange(newSort)
    } else {
      setClientSort(newSort)
    }
  }

  const rows = !isServerMode && clientSort ? sortRows(data, columns, clientSort) : data

  const showPagination = isServerMode && totalRows !== undefined && perPage !== undefined && currentPage !== undefined && onPageChange !== undefined
  const totalPages = showPagination ? Math.max(1, Math.ceil(totalRows! / perPage!)) : 1

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
                className={`px-4 py-2.5 text-xs font-medium text-[var(--color-text-secondary)] whitespace-nowrap ${
                  sort?.key === col.key ? 'bg-[rgba(79,124,255,0.08)]' : ''
                }`}
              >
                {col.sortable ? (
                  <button
                    type="button"
                    onClick={() => toggleSort(col.key)}
                    className="inline-flex items-center gap-1 hover:text-[var(--color-text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] rounded"
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
        <tbody className={loading ? 'opacity-50 pointer-events-none' : ''}>
          {rows.length === 0 && !loading ? (
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
                className={`bg-[var(--color-surface)] border-b border-[var(--color-border)] last:border-0 ${
                  onRowClick
                    ? 'cursor-pointer hover:bg-[var(--color-surface-elevated)] focus:outline-none focus:ring-2 focus:ring-inset focus:ring-[var(--color-accent)]'
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

      {showPagination && totalPages > 1 && (
        <nav
          aria-label="Pagination"
          className="flex items-center justify-between px-4 py-3 border-t border-[var(--color-border)] bg-[var(--color-surface)]"
        >
          <span className="text-xs text-[var(--color-text-muted)]">
            Page {currentPage} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={currentPage! <= 1}
              onClick={() => onPageChange!(currentPage! - 1)}
              className="px-3 py-1 text-xs rounded border border-[var(--color-border)] text-[var(--color-text-secondary)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-elevated)] disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={currentPage! >= totalPages}
              onClick={() => onPageChange!(currentPage! + 1)}
              className="px-3 py-1 text-xs rounded border border-[var(--color-border)] text-[var(--color-text-secondary)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-elevated)] disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
            >
              Next
            </button>
          </div>
        </nav>
      )}
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
