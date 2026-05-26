import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  type ColumnDef,
  type SortingState,
  type VisibilityState,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useMediaQuery } from '@/hooks'
import { DataTableColumnHeader, DataTablePage } from '@/components/data-table'
import { Badge } from '@/components/ui/badge'
import { getHits } from '../api'
import type { HitItem } from '../types'

function formatNum(v: number) {
  return v > 0 ? v.toLocaleString() : '—'
}

const PERIOD_COLORS: Record<string, string> = {
  daily: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  weekly: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  monthly: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
}

const TYPE_COLORS: Record<string, string> = {
  token: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  request: 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-400',
}

function useHitColumns(): ColumnDef<HitItem, unknown>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'created_at',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Time')} />,
      cell: ({ row }) => (
        <span className='text-sm text-muted-foreground'>
          {new Date(row.original.created_at).toLocaleString()}
        </span>
      ),
      size: 170,
    },
    {
      accessorKey: 'user_id',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('User ID')} />,
      cell: ({ row }) => <span className='font-mono text-sm'>{row.original.user_id}</span>,
      size: 80,
    },
    {
      accessorKey: 'username',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Username')} />,
      size: 120,
    },
    {
      accessorKey: 'model_name',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Model')} />,
      size: 150,
    },
    {
      accessorKey: 'limit_type',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Limit Type')} />,
      cell: ({ row }) => (
        <Badge variant='outline' className={TYPE_COLORS[row.original.limit_type] || ''}>
          {row.original.limit_type}
        </Badge>
      ),
      size: 100,
    },
    {
      accessorKey: 'period_type',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Period')} />,
      cell: ({ row }) => (
        <Badge variant='outline' className={PERIOD_COLORS[row.original.period_type] || ''}>
          {row.original.period_type}
        </Badge>
      ),
      size: 90,
    },
    {
      accessorKey: 'used_value',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Used')} />,
      cell: ({ row }) => <span className='tabular-nums'>{formatNum(row.original.used_value)}</span>,
      size: 100,
    },
    {
      accessorKey: 'limit_value',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Limit')} />,
      cell: ({ row }) => <span className='tabular-nums'>{formatNum(row.original.limit_value)}</span>,
      size: 100,
    },
    {
      accessorKey: 'request_id',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Request ID')} />,
      cell: ({ row }) => (
        <span className='font-mono text-xs text-muted-foreground truncate max-w-[200px] block' title={row.original.request_id}>
          {row.original.request_id || '—'}
        </span>
      ),
      size: 220,
    },
  ]
}

export function HitsTable() {
  const { t } = useTranslation()
  const columns = useHitColumns()
  const isMobile = useMediaQuery('(max-width: 640px)')

  const [globalFilter, setGlobalFilter] = useState('')
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: isMobile ? 10 : 20,
  })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['model-usage-limit-hits', globalFilter, pagination.pageIndex, pagination.pageSize],
    queryFn: () =>
      getHits({
        keyword: globalFilter,
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      }),
    staleTime: 30_000,
  })

  const items = data?.data?.items ?? []
  const total = data?.data?.total ?? 0
  const pageCount = Math.ceil(total / pagination.pageSize)

  const table = useReactTable({
    data: items,
    columns,
    pageCount,
    state: { sorting, columnVisibility, globalFilter, pagination },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onGlobalFilterChange: setGlobalFilter,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No hit records')}
      emptyDescription={t('No limit violations have been recorded yet.')}
      toolbarProps={{ searchPlaceholder: t('Search username or model...') }}
    />
  )
}
