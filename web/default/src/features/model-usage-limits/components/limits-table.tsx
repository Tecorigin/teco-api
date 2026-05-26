import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  type SortingState,
  type VisibilityState,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useMediaQuery } from '@/hooks'
import { DataTablePage } from '@/components/data-table'
import { getLimits } from '../api'
import { useLimitsColumns } from './limits-columns'
import { useLimits } from './limits-provider'

export function LimitsTable() {
  const { t } = useTranslation()
  const columns = useLimitsColumns()
  const { refreshTrigger } = useLimits()
  const isMobile = useMediaQuery('(max-width: 640px)')

  const [globalFilter, setGlobalFilter] = useState('')
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: isMobile ? 10 : 20,
  })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['model-usage-limits', globalFilter, pagination.pageIndex, pagination.pageSize, refreshTrigger],
    queryFn: () =>
      getLimits({
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
      emptyTitle={t('No limits configured')}
      emptyDescription={t('Create a limit to restrict usage per user and model.')}
      toolbarProps={{ searchPlaceholder: t('Search username or model...') }}
    />
  )
}
