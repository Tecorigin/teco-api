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
*/
import { useQuery } from '@tanstack/react-query'
import { Clock, Gauge, ShieldCheck, Sparkles } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { getPerfMetrics, getPerfMetricsSummary } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
} from '@/features/performance-metrics/lib/format'
import { computeTimeRange } from '@/lib/time'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { DashboardFilters } from '@/features/dashboard/types'

interface ModelPerformanceStatisticsProps {
  filters?: DashboardFilters
}

export function ModelPerformanceStatistics({ filters }: ModelPerformanceStatisticsProps) {
  const { t } = useTranslation()
  const [selectedModel, setSelectedModel] = useState<string>('')

  const defaultDays =
    filters?.time_granularity === 'hour'
      ? 1
      : filters?.time_granularity === 'day'
      ? 7
      : 30

  const timeRange = useMemo(
    () =>
      computeTimeRange(
        defaultDays,
        filters?.start_timestamp,
        filters?.end_timestamp
      ),
    [defaultDays, filters?.end_timestamp, filters?.start_timestamp]
  )

  const summaryQuery = useQuery({
    queryKey: ['perf-metrics-summary', timeRange],
    queryFn: () => getPerfMetricsSummary(timeRange),
    staleTime: 60_000,
    retry: false,
  })

  const models = useMemo(
    () => summaryQuery.data?.data.models ?? [],
    [summaryQuery.data]
  )

  const selectedModelSummary = useMemo(
    () => models.find((model) => model.model_name === selectedModel),
    [models, selectedModel]
  )

  const aggregateSummary = useMemo(() => {
    if (models.length === 0) {
      return null
    }

    const totals = models.reduce(
      (acc, model) => {
        const requestCount = model.request_count ?? 0
        acc.requests += requestCount
        acc.weightedLatency += model.avg_latency_ms * requestCount
        acc.weightedSuccess += model.success_rate * requestCount
        acc.weightedTps += model.avg_tps * requestCount
        return acc
      },
      {
        requests: 0,
        weightedLatency: 0,
        weightedSuccess: 0,
        weightedTps: 0,
      }
    )

    if (totals.requests === 0) {
      return {
        avg_latency_ms: 0,
        success_rate: 0,
        avg_tps: 0,
      }
    }

    return {
      avg_latency_ms: totals.weightedLatency / totals.requests,
      success_rate: totals.weightedSuccess / totals.requests,
      avg_tps: totals.weightedTps / totals.requests,
    }
  }, [models])

  useEffect(() => {
    if (!selectedModel && models.length > 0) {
      setSelectedModel(models[0].model_name)
    }
  }, [models, selectedModel])

  const selectedModelQuery = useQuery({
    queryKey: ['perf-metrics', selectedModel, timeRange],
    queryFn: () => getPerfMetrics(selectedModel, timeRange),
    enabled: Boolean(selectedModel),
    staleTime: 60_000,
    retry: false,
  })

  const modelGroups = selectedModelQuery.data?.data.groups ?? []
  const loading = summaryQuery.isLoading || selectedModelQuery.isLoading
  const hasModels = models.length > 0

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex flex-col gap-3 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Model Performance Statistics')}</div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {t('Compare model-level latency, throughput, success rate and request volume over the selected time range.')}
          </div>
        </div>

        <div className='flex flex-col gap-2 sm:flex-row sm:items-center'>
          <Select
            value={selectedModel || null}
            onValueChange={(value) => setSelectedModel(value ?? '')}
          >
            <SelectTrigger className='min-w-[220px]'>
              <SelectValue placeholder={t('Select model')} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {models.map((model) => (
                  <SelectItem key={model.model_name} value={model.model_name}>
                    {model.model_name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className='grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4'>
        <StatisticCard
          icon={Sparkles}
          label={t('Models')}
          value={String(models.length)}
          loading={summaryQuery.isLoading}
        />
        <StatisticCard
          icon={Clock}
          label={t('Overall average latency')}
          value={aggregateSummary ? formatLatency(aggregateSummary.avg_latency_ms) : '—'}
          loading={summaryQuery.isLoading}
        />
        <StatisticCard
          icon={ShieldCheck}
          label={t('Overall average success rate')}
          value={aggregateSummary ? formatUptimePct(aggregateSummary.success_rate) : '—'}
          loading={summaryQuery.isLoading}
        />
        <StatisticCard
          icon={Gauge}
          label={t('Selected model throughput')}
          value={selectedModelSummary ? formatThroughput(selectedModelSummary.avg_tps) : '—'}
          loading={summaryQuery.isLoading}
        />
      </div>

      <div className='overflow-hidden border-t'>
        <div className='px-4 py-3 text-xs font-semibold uppercase tracking-wide text-muted-foreground'>
          {t('Top models by request volume')}
        </div>
        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Model')}</TableHead>
                <TableHead className='text-right'>{t('Requests')}</TableHead>
                <TableHead className='text-right'>{t('Avg latency')}</TableHead>
                <TableHead className='text-right'>{t('Success')}</TableHead>
                <TableHead className='text-right'>{t('Throughput')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 5 }).map((_, idx) => (
                  <TableRow key={idx}>
                    <TableCell colSpan={5} className='py-3'>
                      <div className='h-5 rounded bg-muted/30' />
                    </TableCell>
                  </TableRow>
                ))
              ) : hasModels ? (
                models.slice(0, 10).map((model) => (
                  <TableRow key={model.model_name}>
                    <TableCell className='font-medium'>{model.model_name}</TableCell>
                    <TableCell className='text-right'>{model.request_count}</TableCell>
                    <TableCell className='text-right'>
                      {formatLatency(model.avg_latency_ms)}
                    </TableCell>
                    <TableCell className='text-right'>
                      {formatUptimePct(model.success_rate)}
                    </TableCell>
                    <TableCell className='text-right'>
                      {formatThroughput(model.avg_tps)}
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={5} className='text-muted-foreground py-6 text-center'>
                    {t('No performance data available for the selected range.')}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {selectedModel && (
        <div className='border-t p-4'>
          <div className='mb-3 text-sm font-semibold'>{t('Group-level metrics for {{model}}', { model: selectedModel })}</div>
          <div className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Group')}</TableHead>
                  <TableHead className='text-right'>{t('Requests')}</TableHead>
                  <TableHead className='text-right'>{t('Avg latency')}</TableHead>
                  <TableHead className='text-right'>{t('Avg TTFT')}</TableHead>
                  <TableHead className='text-right'>{t('Success')}</TableHead>
                  <TableHead className='text-right'>{t('Throughput')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {selectedModelQuery.isLoading ? (
                  Array.from({ length: 4 }).map((_, idx) => (
                    <TableRow key={idx}>
                      <TableCell colSpan={6} className='py-3'>
                        <div className='h-5 rounded bg-muted/30' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : modelGroups.length > 0 ? (
                  modelGroups.map((group) => (
                    <TableRow key={group.group}> 
                      <TableCell className='font-medium'>{group.group}</TableCell>
                      <TableCell className='text-right'>{group.request_count}</TableCell>
                      <TableCell className='text-right'>
                        {formatLatency(group.avg_latency_ms)}
                      </TableCell>
                      <TableCell className='text-right'>
                        {formatLatency(group.avg_ttft_ms)}
                      </TableCell>
                      <TableCell className='text-right'>
                        {formatUptimePct(group.success_rate)}
                      </TableCell>
                      <TableCell className='text-right'>
                        {formatThroughput(group.avg_tps)}
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className='text-muted-foreground py-6 text-center'>
                      {t('No group performance data available for the selected model.')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </div>
  )
}

function StatisticCard({
  icon: Icon,
  label,
  value,
  loading,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  loading?: boolean
}) {
  return (
    <div className='rounded-lg border bg-muted/40 p-4'>
      <div className='flex items-center gap-2 text-muted-foreground text-xs font-semibold uppercase tracking-wider'>
        <Icon className='size-4' />
        {label}
      </div>
      <div className='mt-3 text-2xl font-semibold'>
        {loading ? <span className='inline-block h-8 w-24 rounded bg-muted/60' /> : value}
      </div>
    </div>
  )
}
