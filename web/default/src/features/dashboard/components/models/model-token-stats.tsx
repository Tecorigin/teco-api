import { useEffect, useMemo, useState, type ComponentType } from 'react'
import { Coins, DatabaseZap, Inbox, Send } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/lib/format'
import { computeTimeRange } from '@/lib/time'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getModelTokenStats } from '@/features/dashboard/api'
import { buildQueryParams, getDefaultDays } from '@/features/dashboard/lib'
import type {
  DashboardFilters,
  ModelTokenStatsItem,
} from '@/features/dashboard/types'

interface ModelTokenStatsProps {
  filters?: DashboardFilters
  isAdmin?: boolean
}

export function ModelTokenStats({ filters, isAdmin = false }: ModelTokenStatsProps) {
  const { t } = useTranslation()
  const [data, setData] = useState<ModelTokenStatsItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    const abortController = new AbortController()
    setLoading(true)
    setError(false)

    const timeRange = computeTimeRange(
      getDefaultDays(filters?.time_granularity),
      filters?.start_timestamp,
      filters?.end_timestamp
    )

    getModelTokenStats(buildQueryParams(timeRange, filters), isAdmin)
      .then((res) => {
        if (abortController.signal.aborted) return
        setData((res?.data || []).sort((a, b) => b.total_tokens - a.total_tokens))
      })
      .catch(() => {
        if (abortController.signal.aborted) return
        setData([])
        setError(true)
      })
      .finally(() => {
        if (!abortController.signal.aborted) setLoading(false)
      })

    return () => abortController.abort()
  }, [filters, isAdmin])

  const totals = useMemo(() => {
    return data.reduce(
      (acc, item) => {
        acc.requestCount += item.request_count || 0
        acc.promptTokens += item.prompt_tokens || 0
        acc.completionTokens += item.completion_tokens || 0
        acc.cacheReadTokens += item.cache_read_tokens || 0
        return acc
      },
      {
        requestCount: 0,
        promptTokens: 0,
        completionTokens: 0,
        cacheReadTokens: 0,
      }
    )
  }, [data])

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex flex-col gap-3 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5'>
        <div>
          <div className='text-sm font-semibold'>{t('Model Token Breakdown')}</div>
          <div className='text-muted-foreground mt-1 text-xs'>
            {t('View detailed input, output, and cache token usage by model')}
          </div>
        </div>
        <div className='grid grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:justify-end'>
          <SummaryChip icon={Inbox} label={t('Input')} value={formatNumber(totals.promptTokens)} />
          <SummaryChip icon={Send} label={t('Output')} value={formatNumber(totals.completionTokens)} />
          <SummaryChip icon={DatabaseZap} label={t('Cache Hit')} value={formatNumber(totals.cacheReadTokens)} />
          <SummaryChip icon={Coins} label={t('Requests')} value={formatNumber(totals.requestCount)} />
        </div>
      </div>

      {loading ? (
        <div className='space-y-2 p-4 sm:p-5'>
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className='h-10 w-full' />
          ))}
        </div>
      ) : error ? (
        <div className='text-muted-foreground p-6 text-sm'>
          {t('Failed to load model token statistics')}
        </div>
      ) : data.length === 0 ? (
        <div className='text-muted-foreground p-6 text-sm'>
          {t('No token statistics found for the selected time range')}
        </div>
      ) : (
        <div className='p-2 sm:p-3'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Model')}</TableHead>
                <TableHead className='text-right'>{t('Requests')}</TableHead>
                <TableHead className='text-right'>{t('Input Tokens')}</TableHead>
                <TableHead className='text-right'>{t('Output Tokens')}</TableHead>
                <TableHead className='text-right'>{t('Cache Hit Tokens')}</TableHead>
                <TableHead className='text-right'>{t('Cache Write Tokens')}</TableHead>
                <TableHead className='text-right'>{t('Total Tokens')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((item) => (
                <TableRow key={item.model_name}>
                  <TableCell className='max-w-[180px] truncate font-medium'>
                    {item.model_name || '-'}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatNumber(item.request_count || 0)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatNumber(item.prompt_tokens || 0)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatNumber(item.completion_tokens || 0)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatNumber(item.cache_read_tokens || 0)}
                  </TableCell>
                  <TableCell className='text-right'>
                    {formatNumber(item.cache_write_tokens || 0)}
                  </TableCell>
                  <TableCell className='text-right font-medium'>
                    {formatNumber(item.total_tokens || 0)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}

function SummaryChip({
  icon: Icon,
  label,
  value,
}: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  return (
    <div className='bg-muted/50 inline-flex items-center gap-2 rounded-lg border px-3 py-1.5'>
      <Icon className='text-muted-foreground size-3.5' />
      <span className='text-muted-foreground text-xs'>{label}</span>
      <span className='text-sm font-semibold'>{value}</span>
    </div>
  )
}