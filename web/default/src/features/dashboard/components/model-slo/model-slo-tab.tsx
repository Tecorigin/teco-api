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
import { Activity, ArrowDown, ArrowUp, Loader2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getModelSLOMetrics } from '@/features/performance-metrics/api'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import { getRollingDateRange } from '@/lib/time'

const SLO_THRESHOLD_OPTIONS = [1000, 2000, 3000, 5000, 8000]
const MODEL_SLO_THRESHOLD_STORAGE_KEY = 'dashboard.model-slo.threshold-ms'

type SortKey = 'model_name' | 'slo' | 'p99_latency_ms' | 'p99_tpot_ms' | 'rpm' | 'tpm'
type SortDirection = 'asc' | 'desc'

function formatDecimal(value: number, digits = 2) {
  return Number.isFinite(value) ? value.toFixed(digits) : '0.00'
}

export function ModelSLOTab() {
  const { t } = useTranslation()
  const [selectedRange, setSelectedRange] = useState(1)
  const [startDate, setStartDate] = useState<Date | undefined>()
  const [endDate, setEndDate] = useState<Date | undefined>()
  const [modelName, setModelName] = useState('')
  const [sloThresholdMs, setSloThresholdMs] = useState(() => {
    if (typeof window === 'undefined') return 5000
    const raw = window.localStorage.getItem(MODEL_SLO_THRESHOLD_STORAGE_KEY)
    const parsed = Number(raw)
    return SLO_THRESHOLD_OPTIONS.includes(parsed) ? parsed : 5000
  })
  const [sortKey, setSortKey] = useState<SortKey>('slo')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

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

  useEffect(() => {
    if (typeof window === 'undefined') return
    window.localStorage.setItem(
      MODEL_SLO_THRESHOLD_STORAGE_KEY,
      String(sloThresholdMs)
    )
  }, [sloThresholdMs])

  const metricsQuery = useQuery({
    queryKey: ['model-slo-metrics', timeRange, modelName],
    queryFn: () =>
      getModelSLOMetrics({
        ...timeRange,
        model_name: modelName || undefined,
        slo_threshold_ms: sloThresholdMs,
      }),
    staleTime: 60_000,
  })

  const rows = useMemo(() => {
    const models = [...(metricsQuery.data?.data.models ?? [])]
    models.sort((left, right) => {
      const leftValue = left[sortKey]
      const rightValue = right[sortKey]
      if (sortKey === 'model_name') {
        const compared = String(leftValue).localeCompare(String(rightValue))
        return sortDirection === 'asc' ? compared : -compared
      }
      const compared = Number(leftValue) - Number(rightValue)
      return sortDirection === 'asc' ? compared : -compared
    })
    return models
  }, [metricsQuery.data?.data.models, sortDirection, sortKey])

  const toggleSort = useCallback((key: SortKey) => {
    setSortKey((current) => {
      if (current === key) {
        setSortDirection((direction) => (direction === 'asc' ? 'desc' : 'asc'))
        return current
      }
      setSortDirection(key === 'model_name' ? 'asc' : 'desc')
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
          value={modelName}
          onChange={(e) => setModelName(e.target.value)}
          placeholder={t('Filter by model name')}
          className='h-8 w-56 text-xs'
        />

        <Select
          value={String(sloThresholdMs)}
          onValueChange={(value) => setSloThresholdMs(Number(value ?? 5000))}
        >
          <SelectTrigger className='h-8 min-w-[150px] text-xs'>
            <SelectValue placeholder={t('SLO threshold')} />
          </SelectTrigger>
          <SelectContent>
            {SLO_THRESHOLD_OPTIONS.map((value) => (
              <SelectItem key={value} value={String(value)}>
                {t('SLO ≤ {{ms}} ms', { ms: value })}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {metricsQuery.isLoading && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <Activity className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>{t('Model SLO Analytics')}</div>
        </div>

        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{renderSortableHead(t('Model'), 'model_name')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('SLO'), 'slo', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('P99 Latency'), 'p99_latency_ms', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('P99 TPOT'), 'p99_tpot_ms', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('RPM'), 'rpm', 'right')}</TableHead>
                <TableHead className='text-right'>{renderSortableHead(t('TPM'), 'tpm', 'right')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {metricsQuery.isLoading ? (
                Array.from({ length: 8 }).map((_, idx) => (
                  <TableRow key={idx}>
                    <TableCell colSpan={6}>
                      <Skeleton className='h-8 w-full' />
                    </TableCell>
                  </TableRow>
                ))
              ) : rows.length > 0 ? (
                rows.map((row) => (
                  <TableRow key={row.model_name}>
                    <TableCell className='font-medium'>{row.model_name}</TableCell>
                    <TableCell className='text-right'>{formatDecimal(row.slo)}%</TableCell>
                    <TableCell className='text-right'>{row.p99_latency_ms} ms</TableCell>
                    <TableCell className='text-right'>{row.p99_tpot_ms} ms</TableCell>
                    <TableCell className='text-right'>{formatDecimal(row.rpm)}</TableCell>
                    <TableCell className='text-right'>{formatDecimal(row.tpm)}</TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className='text-muted-foreground py-8 text-center'>
                    {t('No model SLO data available for the selected range.')}
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