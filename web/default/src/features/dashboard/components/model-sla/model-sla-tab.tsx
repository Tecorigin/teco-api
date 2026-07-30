/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { ShieldCheck, ArrowDown, ArrowUp, Loader2 } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useDebounce } from '@/hooks/use-debounce'
import { getSLAMetrics } from '@/features/performance-metrics/api'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import { getRollingDateRange } from '@/lib/time'

type SortKey =
  | 'username'
  | 'model_name'
  | 'sla'
  | 'total_windows'
  | 'error_windows'
  | 'total_reqs'
  | 'failed_reqs'
type SortDirection = 'asc' | 'desc'

function formatDecimal(value: number, digits = 2) {
  return Number.isFinite(value) ? value.toFixed(digits) : '0.00'
}

function slaColorClass(sla: number) {
  if (sla >= 99.5) return 'text-emerald-600 dark:text-emerald-400'
  if (sla >= 99) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

export function ModelSLATab() {
  const { t } = useTranslation()
  const [selectedRange, setSelectedRange] = useState(1)
  const [startDate, setStartDate] = useState<Date | undefined>()
  const [endDate, setEndDate] = useState<Date | undefined>()
  const [username, setUsername] = useState('')
  const [modelName, setModelName] = useState('')
  const debouncedUsername = useDebounce(username, 400)
  const debouncedModelName = useDebounce(modelName, 400)
  const [sortKey, setSortKey] = useState<SortKey>('sla')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')

  const timeRange = useMemo(() => {
    if (startDate && endDate) {
      return {
        start_timestamp: Math.floor(startDate.getTime() / 1000),
        end_timestamp: Math.floor(endDate.getTime() / 1000),
      }
    }
    const { start, end } = getRollingDateRange(selectedRange)
    return {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  }, [endDate, selectedRange, startDate])

  const handleRangeChange = useCallback((days: number) => {
    setSelectedRange(days)
    setStartDate(undefined)
    setEndDate(undefined)
  }, [])

  const metricsQuery = useQuery({
    queryKey: ['model-sla-metrics', timeRange, debouncedUsername, debouncedModelName],
    queryFn: () =>
      getSLAMetrics({
        ...timeRange,
        username: debouncedUsername || undefined,
        model_name: debouncedModelName || undefined,
      }),
    staleTime: 60_000,
  })

  const rows = useMemo(() => {
    const items = [...(metricsQuery.data?.data.rows ?? [])]
    // First sort by username to group same users together, then sort within each group.
    items.sort((left, right) => {
      const userCompare = left.username.localeCompare(right.username)
      if (userCompare !== 0) return userCompare
      // Same user: sort by the selected key within group.
      const leftValue = left[sortKey]
      const rightValue = right[sortKey]
      if (sortKey === 'model_name') {
        const compared = String(leftValue).localeCompare(String(rightValue))
        return sortDirection === 'asc' ? compared : -compared
      }
      const compared = Number(leftValue) - Number(rightValue)
      return sortDirection === 'asc' ? compared : -compared
    })
    return items
  }, [metricsQuery.data?.data.rows, sortDirection, sortKey])

  const toggleSort = useCallback((key: SortKey) => {
    setSortKey((current) => {
      if (current === key) {
        setSortDirection((direction) => (direction === 'asc' ? 'desc' : 'asc'))
        return current
      }
      setSortDirection(key === 'username' || key === 'model_name' ? 'asc' : 'desc')
      return key
    })
  }, [])

  const renderSortableHead = useCallback(
    (label: string, key: SortKey, align: 'left' | 'right' = 'left') => (
      <button
        type='button'
        onClick={() => toggleSort(key)}
        className={`inline-flex items-center gap-1 ${align === 'right' ? 'ml-auto' : ''}`}
      >
        <span>{label}</span>
        {sortKey === key ? (
          sortDirection === 'asc' ? (
            <ArrowUp className='size-3.5' />
          ) : (
            <ArrowDown className='size-3.5' />
          )
        ) : null}
      </button>
    ),
    [sortDirection, sortKey, toggleSort]
  )

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center gap-1.5 sm:gap-2'>
        <Tabs
          value={String(selectedRange)}
          onValueChange={(value) => handleRangeChange(Number(value))}
        >
          <TabsList>
            {TIME_RANGE_PRESETS.map((preset) => (
              <TabsTrigger
                key={preset.days}
                value={String(preset.days)}
                className='px-2.5 text-xs'
              >
                {t(preset.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <CompactDateTimeRangePicker
          start={startDate}
          end={endDate}
          onChange={(range) => {
            setStartDate(range.start)
            setEndDate(range.end)
          }}
          className='h-8 w-auto max-w-72 text-xs'
        />

        <Input
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder={t('Filter by username')}
          className='h-8 w-40 text-xs'
        />

        <Input
          value={modelName}
          onChange={(e) => setModelName(e.target.value)}
          placeholder={t('Filter by model name')}
          className='h-8 w-40 text-xs'
        />

        {metricsQuery.isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <ShieldCheck className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>{t('Model SLA Analytics')}</div>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{renderSortableHead(t('Username'), 'username')}</TableHead>
                <TableHead>{renderSortableHead(t('Model'), 'model_name')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('SLA'), 'sla', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('Total Windows'), 'total_windows', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('Error Windows'), 'error_windows', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('Total Requests'), 'total_reqs', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('Failed Requests'), 'failed_reqs', 'right')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {metricsQuery.isLoading ? (
                Array.from({ length: 8 }).map((_, idx) => (
                  <TableRow key={idx}>
                    <TableCell colSpan={7}>
                      <Skeleton className='h-8 w-full' />
                    </TableCell>
                  </TableRow>
                ))
              ) : rows.length > 0 ? (
                rows.map((row, idx) => {
                  const isFirstOfUser = idx === 0 || rows[idx - 1].username !== row.username
                  let rowSpan = 1
                  if (isFirstOfUser) {
                    for (let i = idx + 1; i < rows.length && rows[i].username === row.username; i++) {
                      rowSpan++
                    }
                  }
                  return (
                    <TableRow key={`${row.username}-${row.model_name}-${idx}`}>
                      {isFirstOfUser && (
                        <TableCell className='font-medium align-top' rowSpan={rowSpan}>
                          {row.username}
                        </TableCell>
                      )}
                      <TableCell className='font-medium'>{row.model_name}</TableCell>
                      <TableCell className={`text-right font-semibold ${slaColorClass(row.sla)}`}>
                        {formatDecimal(row.sla)}%
                      </TableCell>
                      <TableCell className='text-right'>{row.total_windows}</TableCell>
                      <TableCell className='text-right'>{row.error_windows}</TableCell>
                      <TableCell className='text-right'>{row.total_reqs}</TableCell>
                      <TableCell className='text-right'>{row.failed_reqs}</TableCell>
                    </TableRow>
                  )
                })
              ) : (
                <TableRow>
                  <TableCell colSpan={7} className='text-muted-foreground py-8 text-center'>
                    {t('No model SLA data available for the selected range.')}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}
