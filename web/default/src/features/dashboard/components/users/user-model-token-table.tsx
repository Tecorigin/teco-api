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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatNumber } from '@/lib/format'

import type { UserModelTokenStatsItem } from '../../types'

interface UserModelTokenTableProps {
  data: UserModelTokenStatsItem[]
  loading: boolean
}

interface UserTokenStatsRow {
  username: string
  requestCount: number
  totalTokens: number
  promptTokens: number
  completionTokens: number
  cacheReadTokens: number
}

interface UserModelTokenRow extends UserTokenStatsRow {
  modelName: string
  userRowSpan?: number
}

const SKELETON_ROWS = ['one', 'two', 'three', 'four', 'five']

export function UserModelTokenTable(props: UserModelTokenTableProps) {
  const { t } = useTranslation()
  const [usernameFilter, setUsernameFilter] = useState('')
  const rows = useMemo<UserModelTokenRow[]>(() => {
    const input = usernameFilter ? usernameFilter.trim().toLowerCase() : ''
    const source = input ? props.data.filter((it) => (it.username || '').toLowerCase().includes(input)) : props.data
    const grouped = new Map<string, UserModelTokenRow>()
    for (const item of source) {
      const username = item.username || t('Unknown')
      const modelName = item.model_name || '-'
      const key = `${username}\u0000${modelName}`
      const row = grouped.get(key) ?? {
        username,
        modelName,
        requestCount: 0,
        totalTokens: 0,
        promptTokens: 0,
        completionTokens: 0,
        cacheReadTokens: 0,
      }
      row.requestCount += Number(item.request_count) || 0
      row.totalTokens += Number(item.total_tokens) || 0
      row.promptTokens += Number(item.prompt_tokens) || 0
      row.completionTokens += Number(item.completion_tokens) || 0
      row.cacheReadTokens += Number(item.cache_read_tokens) || 0
      grouped.set(key, row)
    }
    const sortedRows = [...grouped.values()].sort(
      (left, right) =>
        left.username.localeCompare(right.username) ||
        right.totalTokens - left.totalTokens
    )
    let index = 0
    while (index < sortedRows.length) {
      let end = index + 1
      while (
        end < sortedRows.length &&
        sortedRows[end].username === sortedRows[index].username
      ) {
        end++
      }
      sortedRows[index].userRowSpan = end - index
      index = end
    }
    return sortedRows
  }, [props.data, t, usernameFilter])

  const userRows = useMemo<UserTokenStatsRow[]>(() => {
    const grouped = new Map<string, UserTokenStatsRow>()
    for (const row of rows) {
      const userRow = grouped.get(row.username) ?? {
        username: row.username,
        requestCount: 0,
        totalTokens: 0,
        promptTokens: 0,
        completionTokens: 0,
        cacheReadTokens: 0,
      }
      userRow.requestCount += row.requestCount
      userRow.totalTokens += row.totalTokens
      userRow.promptTokens += row.promptTokens
      userRow.completionTokens += row.completionTokens
      userRow.cacheReadTokens += row.cacheReadTokens
      grouped.set(row.username, userRow)
    }
    return [...grouped.values()].sort(
      (left, right) => right.totalTokens - left.totalTokens
    )
  }, [rows])

  let content: React.ReactNode
  if (props.loading) {
    content = (
      <div className='space-y-2 p-4'>
        {SKELETON_ROWS.map((row) => (
          <Skeleton key={row} className='h-10 w-full' />
        ))}
      </div>
    )
  } else if (rows.length === 0) {
    content = (
      <div className='text-muted-foreground p-6 text-sm'>
        {t('No data available')}
      </div>
    )
  } else {
    content = (
      <div className='grid gap-3'>
        <div className='overflow-hidden rounded-lg border'>
          <div className='border-b px-3 py-2 sm:px-5 sm:py-3 flex items-center justify-between gap-2'>
            <div className='text-sm font-semibold'>
              {t('User Token Usage Summary')}
            </div>
            <div className='ml-2'>
              <input
                type='text'
                value={usernameFilter}
                onChange={(e) => setUsernameFilter(e.target.value)}
                placeholder={t('Filter by username')}
                className='h-8 rounded-md border px-2 text-xs'
              />
            </div>
          </div>
          <div className='p-2 sm:p-3'>
            <TokenStatsTable rows={userRows} />
          </div>
        </div>
        <div className='overflow-hidden rounded-lg border'>
          <div className='border-b px-3 py-2 sm:px-5 sm:py-3'>
            <div className='text-sm font-semibold'>
              {t('User Model Token Usage Details')}
            </div>
          </div>
          <div className='p-2 sm:p-3'>
            <TokenStatsTable rows={rows} showModel />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className='space-y-3'>{content}</div>
  )
}

function TokenStatsTable(props: {
  rows: UserTokenStatsRow[] | UserModelTokenRow[]
  showModel?: boolean
}) {
  const { t } = useTranslation()

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('User')}</TableHead>
          {props.showModel && <TableHead>{t('Model')}</TableHead>}
          <TableHead className='text-right'>{t('Requests')}</TableHead>
          <TableHead className='text-right'>{t('Total Tokens')}</TableHead>
          <TableHead className='text-right'>{t('Input Tokens')}</TableHead>
          <TableHead className='text-right'>{t('Output Tokens')}</TableHead>
          <TableHead className='text-right'>{t('Cache Hit Tokens')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {props.rows.map((row) => {
          const modelRow = props.showModel ? (row as UserModelTokenRow) : null
          return (
            <TableRow
              key={
                modelRow
                  ? `${row.username}\u0000${modelRow.modelName}`
                  : row.username
              }
            >
              {(!modelRow || modelRow.userRowSpan) && (
                <TableCell
                  rowSpan={modelRow?.userRowSpan}
                  className={modelRow ? 'align-top font-medium' : 'font-medium'}
                >
                  {row.username}
                </TableCell>
              )}
              {modelRow && <TableCell>{modelRow.modelName}</TableCell>}
              <TableCell className='text-right'>
                {formatNumber(row.requestCount)}
              </TableCell>
              <TableCell className='text-right font-medium'>
                {formatNumber(row.totalTokens)}
              </TableCell>
              <TableCell className='text-right'>
                {formatNumber(row.promptTokens)}
              </TableCell>
              <TableCell className='text-right'>
                {formatNumber(row.completionTokens)}
              </TableCell>
              <TableCell className='text-right'>
                {formatNumber(row.cacheReadTokens)}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}