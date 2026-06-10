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
/* eslint-disable react-refresh/only-export-components */
import { useState, useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { AlertCircle, Music, Play } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { DataTableColumnHeader } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { TASK_ACTIONS, TASK_STATUS } from '../../constants'
import { taskActionMapper, taskStatusMapper } from '../../lib/mappers'
import type { TaskLog } from '../../types'
import {
  AudioPreviewDialog,
  type AudioClip,
} from '../dialogs/audio-preview-dialog'
import { FailReasonDialog } from '../dialogs/fail-reason-dialog'
import { useUsageLogsContext } from '../usage-logs-provider'
import {
  createDurationColumn,
  createChannelColumn,
  createProgressColumn,
} from './column-helpers'

function parseTaskData(data: unknown): unknown[] {
  if (Array.isArray(data)) return data
  if (typeof data === 'string') {
    try {
      const parsed = JSON.parse(data)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

function getVideoUrlFromValue(value: unknown): string {
  if (!value || typeof value !== 'object') return ''
  const record = value as Record<string, unknown>
  const direct = [record.result_url, record.url, record.download_url].find(
    (item) => typeof item === 'string' && item.trim()
  )
  if (typeof direct === 'string') return direct.trim()

  const metadata = record.metadata
  if (metadata && typeof metadata === 'object' && !Array.isArray(metadata)) {
    const metaRecord = metadata as Record<string, unknown>
    const metaUrl = [metaRecord.url, metaRecord.download_url].find(
      (item) => typeof item === 'string' && item.trim()
    )
    if (typeof metaUrl === 'string') return metaUrl.trim()
  }

  return ''
}

function getTaskVideoUrl(log: TaskLog): string {
  if (typeof log.result_url === 'string' && log.result_url.trim()) {
    return log.result_url.trim()
  }

  const data = parseTaskData(log.data)
  for (const item of data) {
    const url = getVideoUrlFromValue(item)
    if (url) return url
  }

  return getVideoUrlFromValue(log.data)
}

function parseTaskProperties(properties: unknown): Record<string, unknown> | null {
  if (!properties) return null
  if (typeof properties === 'object' && !Array.isArray(properties)) {
    return properties as Record<string, unknown>
  }
  if (typeof properties === 'string') {
    try {
      const parsed = JSON.parse(properties)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>
      }
    } catch {
      return null
    }
  }
  return null
}

function getOriginModelName(log: TaskLog): string {
  const properties = parseTaskProperties(log.properties)
  const modelName = properties?.origin_model_name
  return typeof modelName === 'string' ? modelName.trim() : ''
}

function AudioPreviewCell({ log }: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const clips = useMemo(() => {
    const data = parseTaskData(log.data)
    return data.filter(
      (c) =>
        c && typeof c === 'object' && (c as Record<string, unknown>).audio_url
    )
  }, [log.data])

  if (clips.length === 0) return null

  return (
    <>
      <button
        type='button'
        className='group flex items-center gap-1 text-left text-xs'
        onClick={() => setOpen(true)}
      >
        <Music className='text-muted-foreground size-3' />
        <span className='text-foreground leading-snug group-hover:underline'>
          {t('Click to preview audio')}
        </span>
      </button>
      <AudioPreviewDialog
        open={open}
        onOpenChange={setOpen}
        clips={clips as AudioClip[]}
      />
    </>
  )
}

function ExpiredVideoPlaceholder({ label, description }: { label: string; description: string }) {
  return (
    <div className='text-muted-foreground flex min-h-[320px] flex-col items-center justify-center gap-3 bg-muted/30 px-6 text-center'>
      <div className='flex h-12 w-12 items-center justify-center rounded-full bg-background/80 shadow-sm'>
        <AlertCircle className='h-6 w-6 text-amber-500' />
      </div>
      <div className='space-y-1'>
        <div className='text-foreground text-sm font-medium'>{label}</div>
        <div className='text-xs'>{description}</div>
      </div>
    </div>
  )
}

function VideoPreviewCell({ log }: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [isExpired, setIsExpired] = useState(false)
  const videoUrl = getTaskVideoUrl(log)

  if (!videoUrl) return null

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setIsExpired(false)
    }
    setOpen(nextOpen)
  }

  return (
    <>
      <button
        type='button'
        className='group flex items-center gap-1 text-left text-xs'
        onClick={() => handleOpenChange(true)}
      >
        <Play className='text-muted-foreground size-3' />
        <span className='text-foreground leading-snug group-hover:underline'>
          {t('Click to preview video')}
        </span>
      </button>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className='w-[96vw] max-w-5xl'>
          <DialogHeader>
            <DialogTitle>{t('Video Preview')}</DialogTitle>
            <DialogDescription>
              {t('Task ID:')} {log.task_id}
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-3'>
            <p className='text-sm text-amber-600'>
              {t('Generated videos use temporary links. If they expire, they will no longer be displayed, so please save them as soon as possible.')}
            </p>
            <div className='max-h-[80vh] overflow-auto rounded-md border bg-muted/10'>
              {isExpired ? (
                <ExpiredVideoPlaceholder
                  label={t('Video expired')}
                  description={t('This temporary video link has expired.')}
                />
              ) : (
                <video
                  src={videoUrl}
                  className='mx-auto block max-h-[80vh] w-full object-contain'
                  controls
                  autoPlay
                  playsInline
                  onError={() => setIsExpired(true)}
                />
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

export function useTaskLogsColumns(isAdmin: boolean): ColumnDef<TaskLog>[] {
  const { t } = useTranslation()
  const columns: ColumnDef<TaskLog>[] = [
    {
      accessorKey: 'submit_time',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Submit Time')} />
      ),
      cell: ({ row }) => {
        const log = row.original
        const submitTime = row.getValue('submit_time') as number

        return (
          <div className='flex flex-col gap-0.5'>
            <span className='font-mono text-xs tabular-nums'>
              {formatTimestampToDate(submitTime, 'seconds')}
            </span>
            {log.finish_time ? (
              <span className='text-muted-foreground/60 font-mono text-[11px] tabular-nums'>
                {formatTimestampToDate(log.finish_time, 'seconds')}
              </span>
            ) : (
              <span className='text-muted-foreground/50 text-[11px]'>-</span>
            )}
          </div>
        )
      },
      meta: { label: t('Submit Time') },
    },
  ]

  if (isAdmin) {
    columns.push(createChannelColumn<TaskLog>({ headerLabel: t('Channel') }), {
      id: 'user',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('User')} />
      ),
      cell: function UserCell({ row }) {
        const { sensitiveVisible, setSelectedUserId, setUserInfoDialogOpen } =
          useUsageLogsContext()
        const log = row.original
        const displayName = log.username || String(log.user_id || '?')

        return (
          <button
            type='button'
            className='flex items-center gap-1.5 text-left'
            onClick={(e) => {
              e.stopPropagation()
              setSelectedUserId(log.user_id)
              setUserInfoDialogOpen(true)
            }}
          >
            <Avatar className='ring-border/60 size-6 ring-1'>
              <AvatarFallback
                className={cn(
                  'text-[11px] font-semibold',
                  !sensitiveVisible && 'bg-muted text-muted-foreground'
                )}
                style={
                  sensitiveVisible ? getUserAvatarStyle(displayName) : undefined
                }
              >
                {sensitiveVisible ? getUserAvatarFallback(displayName) : '•'}
              </AvatarFallback>
            </Avatar>
            <span className='text-muted-foreground truncate text-sm hover:underline'>
              {sensitiveVisible ? displayName : '••••'}
            </span>
          </button>
        )
      },
      meta: { label: t('User'), mobileHidden: true },
    })
  }

  columns.push(
    {
      accessorKey: 'task_id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Task ID')} />
      ),
      cell: ({ row }) => {
        const log = row.original
        const taskId = row.getValue('task_id') as string
        if (!taskId) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }
        return (
          <div className='flex max-w-[170px] flex-col gap-0.5'>
            <StatusBadge
              label={taskId}
              autoColor={taskId}
              size='sm'
              showDot={false}
              className='border-border/60 bg-muted/30 max-w-full truncate rounded-md border px-1.5 py-0.5 font-mono'
            />
            <span className='text-muted-foreground/60 truncate text-[11px]'>
              {t(log.platform)} · {t(taskActionMapper.getLabel(log.action))}
            </span>
          </div>
        )
      },
      meta: { label: t('Task ID'), mobileTitle: true },
    },
    {
      id: 'origin_model_name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Model Name')} />
      ),
      accessorFn: (row) => getOriginModelName(row),
      cell: ({ row }) => {
        const log = row.original
        const modelName = getOriginModelName(log)

        if (!modelName) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }

        return (
          <div className='flex max-w-[220px] flex-col gap-0.5'>
            <StatusBadge
              label={modelName}
              copyText={modelName}
              size='sm'
              showDot={false}
              className='border-border/60 bg-muted/30 text-foreground max-w-full truncate rounded-md border px-1.5 py-0.5'
            />
          </div>
        )
      },
      meta: { label: t('Model Name') },
    },
    createDurationColumn<TaskLog>({
      submitTimeKey: 'submit_time',
      finishTimeKey: 'finish_time',
      unit: 'seconds',
      headerLabel: t('Duration'),
      warningThresholdSec: 300,
    }),
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Status')} />
      ),
      cell: ({ row }) => {
        const status = row.getValue('status') as string
        return (
          <StatusBadge
            label={t(taskStatusMapper.getLabel(status, status || 'Submitting'))}
            variant={taskStatusMapper.getVariant(status)}
            size='sm'
            copyable={false}
            showDot
          />
        )
      },
      meta: { label: t('Status') },
    },
    createProgressColumn<TaskLog>({ headerLabel: t('Progress') }),
    {
      accessorKey: 'fail_reason',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('Details')} />
      ),
      cell: function DetailsCell({ row }) {
        const log = row.original
        const failReason = row.getValue('fail_reason') as string
        const status = log.status
        const [dialogOpen, setDialogOpen] = useState(false)

        const isSunoSuccess =
          log.platform === 'suno' && status === TASK_STATUS.SUCCESS
        if (isSunoSuccess) {
          const data = parseTaskData(log.data)
          if (
            data.some(
              (c) =>
                c &&
                typeof c === 'object' &&
                (c as Record<string, unknown>).audio_url
            )
          ) {
            return <AudioPreviewCell log={log} />
          }
        }

        const isVideoTask =
          log.action === TASK_ACTIONS.GENERATE ||
          log.action === TASK_ACTIONS.TEXT_GENERATE ||
          log.action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
          log.action === TASK_ACTIONS.REFERENCE_GENERATE ||
          log.action === TASK_ACTIONS.REMIX_GENERATE
        const isSuccess = status === TASK_STATUS.SUCCESS

        if (isSuccess && isVideoTask && getTaskVideoUrl(log)) {
          return <VideoPreviewCell log={log} />
        }

        if (!failReason) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }

        return (
          <>
            <button
              type='button'
              className='group flex max-w-[200px] items-center gap-1 text-left text-xs'
              onClick={() => setDialogOpen(true)}
              title={t('Click to view full error message')}
            >
              <span className='truncate leading-snug text-red-600 group-hover:underline dark:text-red-400'>
                {failReason}
              </span>
            </button>
            <FailReasonDialog
              failReason={failReason}
              open={dialogOpen}
              onOpenChange={setDialogOpen}
            />
          </>
        )
      },
      meta: { label: t('Details') },
      size: 200,
      maxSize: 220,
    }
  )

  return columns
}
