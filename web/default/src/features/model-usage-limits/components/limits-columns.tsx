import { type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { DataTableColumnHeader } from '@/components/data-table'
import { Pencil, Trash2 } from 'lucide-react'
import type { LimitItem } from '../types'
import { useLimits } from './limits-provider'

function formatNum(v: number) {
  return v > 0 ? v.toLocaleString() : '—'
}

function formatUsage(used: number, limit: number) {
  if (limit > 0) {
    return `${(used || 0).toLocaleString()} / ${limit.toLocaleString()}`
  }
  return '—'
}

export function useLimitsColumns(): ColumnDef<LimitItem, unknown>[] {
  const { t } = useTranslation()
  const { setOpen, setCurrentRow, setDeleteOpen, setDeleteRow } = useLimits()

  return [
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
      accessorKey: 'daily_token_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Daily Token')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.daily_token_used, row.original.daily_token_limit)}
        </span>
      ),
      size: 130,
    },
    {
      accessorKey: 'daily_request_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Daily Req')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.daily_request_used, row.original.daily_request_limit)}
        </span>
      ),
      size: 130,
    },
    {
      accessorKey: 'weekly_token_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Weekly Token')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.weekly_token_used, row.original.weekly_token_limit)}
        </span>
      ),
      size: 140,
    },
    {
      accessorKey: 'weekly_request_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Weekly Req')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.weekly_request_used, row.original.weekly_request_limit)}
        </span>
      ),
      size: 130,
    },
    {
      accessorKey: 'monthly_token_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Monthly Token')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.monthly_token_used, row.original.monthly_token_limit)}
        </span>
      ),
      size: 150,
    },
    {
      accessorKey: 'monthly_request_limit',
      header: ({ column }) => <DataTableColumnHeader column={column} title={t('Monthly Req')} />,
      cell: ({ row }) => (
        <span className='tabular-nums'>
          {formatUsage(row.original.monthly_request_used, row.original.monthly_request_limit)}
        </span>
      ),
      size: 140,
    },
    {
      id: 'actions',
      header: t('Actions'),
      cell: ({ row }) => (
        <div className='flex items-center gap-1'>
          <Button
            variant='ghost'
            size='icon'
            className='size-8'
            onClick={() => {
              setCurrentRow(row.original as unknown as Record<string, unknown>)
              setOpen('update')
            }}
          >
            <Pencil className='size-4' />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            className='size-8 text-destructive hover:text-destructive'
            onClick={() => {
              setDeleteRow(row.original as unknown as Record<string, unknown>)
              setDeleteOpen(true)
            }}
          >
            <Trash2 className='size-4' />
          </Button>
        </div>
      ),
      size: 80,
      enableHiding: false,
    },
  ]
}
