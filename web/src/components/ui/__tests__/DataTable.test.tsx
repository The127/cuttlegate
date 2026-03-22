import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithAxe } from '../../../test/renderWithAxe'
import { DataTable, type ColumnDef } from '../DataTable'

interface Row {
  id: number
  name: string
  value: number
}

const columns: ColumnDef<Row>[] = [
  { key: 'name', header: 'Name', cell: (r) => r.name, sortable: true, sortValue: (r) => r.name },
  { key: 'value', header: 'Value', cell: (r) => String(r.value), sortable: true, sortValue: (r) => r.value },
  { key: 'actions', header: '', cell: () => <button type="button">Edit</button> },
]

const rows: Row[] = [
  { id: 1, name: 'Charlie', value: 30 },
  { id: 2, name: 'Alice', value: 10 },
  { id: 3, name: 'Bob', value: 20 },
]

describe('DataTable', () => {
  it('renders column headers', () => {
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('Value')).toBeInTheDocument()
  })

  it('renders all row data', () => {
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('Charlie')).toBeInTheDocument()
  })

  it('renders empty state when data is empty', () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        emptyState={<span>Nothing here</span>}
        aria-label="Test table"
      />,
    )
    expect(screen.getByText('Nothing here')).toBeInTheDocument()
  })

  it('renders default empty state when no emptyState prop', () => {
    render(<DataTable columns={columns} data={[]} aria-label="Test table" />)
    expect(screen.getByText('No data')).toBeInTheDocument()
  })

  it('uses role="table"', () => {
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    expect(screen.getByRole('table')).toBeInTheDocument()
  })

  it('empty header cell renders sr-only Actions label', () => {
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    expect(screen.getByText('Actions')).toHaveClass('sr-only')
  })

  it('calls onRowClick when a row is clicked', async () => {
    const user = userEvent.setup()
    const onRowClick = vi.fn()
    render(<DataTable columns={columns} data={rows} onRowClick={onRowClick} aria-label="Test table" />)
    await user.click(screen.getByText('Alice').closest('tr')!)
    expect(onRowClick).toHaveBeenCalledWith(rows[1])
  })

  it('calls onRowClick on Enter key', async () => {
    const user = userEvent.setup()
    const onRowClick = vi.fn()
    render(<DataTable columns={columns} data={rows} onRowClick={onRowClick} aria-label="Test table" />)
    const row = screen.getByText('Alice').closest('tr')!
    row.focus()
    await user.keyboard('{Enter}')
    expect(onRowClick).toHaveBeenCalledWith(rows[1])
  })

  it('calls onRowClick on Space key', async () => {
    const user = userEvent.setup()
    const onRowClick = vi.fn()
    render(<DataTable columns={columns} data={rows} onRowClick={onRowClick} aria-label="Test table" />)
    const row = screen.getByText('Bob').closest('tr')!
    row.focus()
    await user.keyboard(' ')
    expect(onRowClick).toHaveBeenCalledWith(rows[2])
  })

  it('sorts ascending on first click of a sortable column header', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    await user.click(screen.getByRole('button', { name: /Name/i }))
    const cells = screen.getAllByRole('cell', { name: /^(Alice|Bob|Charlie)$/ })
    expect(cells[0]).toHaveTextContent('Alice')
    expect(cells[1]).toHaveTextContent('Bob')
    expect(cells[2]).toHaveTextContent('Charlie')
  })

  it('sorts descending on second click of same column header', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    await user.click(screen.getByRole('button', { name: /Name/i }))
    await user.click(screen.getByRole('button', { name: /Name/i }))
    const cells = screen.getAllByRole('cell', { name: /^(Alice|Bob|Charlie)$/ })
    expect(cells[0]).toHaveTextContent('Charlie')
    expect(cells[1]).toHaveTextContent('Bob')
    expect(cells[2]).toHaveTextContent('Alice')
  })

  it('th has aria-sort="none" for sortable but unsorted column', () => {
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    const nameTh = screen.getByRole('columnheader', { name: /Name/i })
    expect(nameTh).toHaveAttribute('aria-sort', 'none')
  })

  it('th has aria-sort="ascending" after first sort click', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    await user.click(screen.getByRole('button', { name: /Name/i }))
    const nameTh = screen.getByRole('columnheader', { name: /Name/i })
    expect(nameTh).toHaveAttribute('aria-sort', 'ascending')
  })

  it('th has aria-sort="descending" after second sort click', async () => {
    const user = userEvent.setup()
    render(<DataTable columns={columns} data={rows} aria-label="Test table" />)
    await user.click(screen.getByRole('button', { name: /Name/i }))
    await user.click(screen.getByRole('button', { name: /Name/i }))
    const nameTh = screen.getByRole('columnheader', { name: /Name/i })
    expect(nameTh).toHaveAttribute('aria-sort', 'descending')
  })

  // @happy — passes axe
  it('passes axe — table with data', async () => {
    const { axeResults } = await renderWithAxe(
      <DataTable columns={columns} data={rows} aria-label="Test table" />,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe — empty table', async () => {
    const { axeResults } = await renderWithAxe(
      <DataTable columns={columns} data={[]} emptyState={<span>Nothing</span>} aria-label="Test table" />,
    )
    expect(axeResults).toHaveNoViolations()
  })

  it('passes axe — clickable rows', async () => {
    const { axeResults } = await renderWithAxe(
      <DataTable columns={columns} data={rows} onRowClick={() => {}} aria-label="Test table" />,
    )
    expect(axeResults).toHaveNoViolations()
  })
})
