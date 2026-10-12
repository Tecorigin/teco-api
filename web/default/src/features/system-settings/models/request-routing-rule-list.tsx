import { Eye, Pencil, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import type { Option } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { RequestRoutingRuleEditor } from './request-routing-rule-editor'
import {
  newRoutingRule,
  routingConfigurationSchema,
} from './request-routing-schema'
import type { RoutingRule } from './request-routing-schema'

type RoutingChannelOption = Option & { models: string[] }

export function RequestRoutingRuleList(props: {
  rules: RoutingRule[]
  channels: RoutingChannelOption[]
  models: Option[]
  groups: Option[]
  onChange: (rules: RoutingRule[]) => Promise<boolean>
  isSaving: boolean
  saveError: string
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [editor, setEditor] = useState<{
    index: number
    rule: RoutingRule
    readOnly: boolean
  } | null>(null)
  const [deleting, setDeleting] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const busy = saving || props.isSaving
  const visibleRules = props.rules
    .map((rule, index) => ({ rule, index }))
    .filter(({ rule }) => {
      const text = [rule.id, ...rule.models, ...rule.groups]
        .join(' ')
        .toLocaleLowerCase()
      return text.includes(search.trim().toLocaleLowerCase())
    })
  const openRule = (index: number, readOnly: boolean) => {
    const rule =
      index < 0 ? newRoutingRule() : structuredClone(props.rules[index])
    setError('')
    setEditor({ index, rule, readOnly })
  }
  const commitRule = async () => {
    if (!editor || busy) return
    const rule = editor.rule
    const valid = routingConfigurationSchema.safeParse({
      enabled: false,
      rules: [rule],
    }).success
    if (
      !valid ||
      !rule.id.trim() ||
      rule.models.length === 0 ||
      rule.ranges.length === 0 ||
      rule.ranges.some(
        (range) =>
          range.instances.length === 0 ||
          (range.max_tokens != null && range.max_tokens <= range.min_tokens) ||
          range.instances.some(
            (instance) =>
              !instance.instance_id.trim() || instance.channel_ids.length === 0
          )
      )
    ) {
      setError(t('Invalid routing configuration'))
      return
    }
    if (
      props.rules.some(
        (current, index) => index !== editor.index && current.id === rule.id
      )
    ) {
      setError(t('Invalid routing configuration'))
      return
    }
    const next =
      editor.index < 0
        ? [...props.rules, rule]
        : props.rules.map((current, index) =>
            index === editor.index ? rule : current
          )
    setSaving(true)
    try {
      if (await props.onChange(next)) setEditor(null)
    } finally {
      setSaving(false)
    }
  }
  const deleteRule = async () => {
    if (deleting === null || busy) return
    setSaving(true)
    try {
      if (
        await props.onChange(
          props.rules.filter((_, index) => index !== deleting)
        )
      )
        setDeleting(null)
    } finally {
      setSaving(false)
    }
  }

  let dialogTitle = t('Edit Rule')
  if (editor?.readOnly) dialogTitle = t('Details')
  else if (editor?.index === -1) dialogTitle = t('Add Rule')

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex min-w-0 flex-wrap items-center gap-3'>
          <span className='text-sm font-medium'>
            {t('Rules')} ({props.rules.length})
          </span>
          <Input
            className='w-64 max-w-full'
            aria-label={t('Search rules...')}
            placeholder={t('Search rules...')}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
        <Button
          type='button'
          size='sm'
          disabled={busy || props.rules.length >= 100}
          onClick={() => openRule(-1, false)}
        >
          <Plus data-icon='inline-start' />
          {t('Add Rule')}
        </Button>
      </div>
      <div className='overflow-x-auto rounded-md border'>
        <table className='w-full text-sm'>
          <thead className='bg-muted/50 text-muted-foreground'>
            <tr>
              {[
                t('Rule Name'),
                t('Models'),
                t('Groups'),
                t('Input Tokens'),
                t('Channels'),
                t('Actions'),
              ].map((label) => (
                <th
                  key={label}
                  className='px-3 py-3 text-left font-medium whitespace-nowrap'
                >
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visibleRules.flatMap(({ rule, index }) => {
              const ranges = rule.ranges.length > 0 ? rule.ranges : [null]
              return ranges.map((range, position) => {
                const channelIDs = [
                  ...new Set(
                    range?.instances.flatMap(
                      (instance) => instance.channel_ids
                    ) ?? []
                  ),
                ]
                const channelLabels = channelIDs.map(
                  (id) =>
                    props.channels.find(
                      (channel) => channel.value === String(id)
                    )?.label ?? `#${id}`
                )
                return (
                  <tr
                    key={`${index}-${position}`}
                    className='hover:bg-muted/30 border-t align-top'
                  >
                    {position === 0 && (
                      <td
                        rowSpan={ranges.length}
                        className='max-w-60 min-w-36 border-r px-3 py-3 align-middle break-words'
                      >
                        <button
                          type='button'
                          className='text-left font-medium hover:underline'
                          onClick={() => openRule(index, true)}
                        >
                          {rule.id}
                        </button>
                      </td>
                    )}
                    {position === 0 && (
                      <td
                        rowSpan={ranges.length}
                        className='max-w-64 border-r px-3 py-3 align-middle break-words'
                      >
                        {rule.models.join(', ')}
                      </td>
                    )}
                    {position === 0 && (
                      <td
                        rowSpan={ranges.length}
                        className='max-w-48 border-r px-3 py-3 align-middle break-words'
                      >
                        {rule.groups.length
                          ? rule.groups.join(', ')
                          : t('All Groups')}
                      </td>
                    )}
                    <td className='min-w-40 border-r px-3 py-3 align-middle font-mono text-xs'>
                      {range ? (
                        <div>
                          [{range.min_tokens.toLocaleString()},{' '}
                          {range.max_tokens == null
                            ? t('Unlimited')
                            : range.max_tokens.toLocaleString()}
                          )
                        </div>
                      ) : (
                        t('Not set')
                      )}
                    </td>
                    <td className='max-w-64 border-r px-3 py-3 align-middle'>
                      <div className='space-y-1 break-words'>
                        {channelLabels.length > 0
                          ? channelLabels.map((label, channelIndex) => (
                              <div key={channelIDs[channelIndex]}>{label}</div>
                            ))
                          : t('No channels selected')}
                      </div>
                    </td>
                    {position === 0 && (
                      <td
                        rowSpan={ranges.length}
                        className='px-3 py-2 align-middle'
                      >
                        <div className='flex gap-1'>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon'
                            title={t('View details')}
                            aria-label={t('View details')}
                            onClick={() => openRule(index, true)}
                          >
                            <Eye className='size-4' />
                          </Button>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon'
                            title={t('Edit Rule')}
                            aria-label={t('Edit Rule')}
                            onClick={() => openRule(index, false)}
                          >
                            <Pencil className='size-4' />
                          </Button>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon'
                            title={t('Remove Rule')}
                            aria-label={t('Remove Rule')}
                            onClick={() => setDeleting(index)}
                          >
                            <Trash2 className='size-4' />
                          </Button>
                        </div>
                      </td>
                    )}
                  </tr>
                )
              })
            })}
            {visibleRules.length === 0 && (
              <tr>
                <td
                  colSpan={6}
                  className='text-muted-foreground py-10 text-center'
                >
                  {t(props.rules.length ? 'No matching rules' : 'No rules yet')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <Sheet
        open={editor !== null}
        onOpenChange={(open) => {
          if (!open && !busy) setEditor(null)
        }}
      >
        <SheetContent
          side='right'
          className={sideDrawerContentClassName('sm:max-w-2xl')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{dialogTitle}</SheetTitle>
          </SheetHeader>
          <div className={sideDrawerFormClassName()}>
            {(error || props.saveError) && (
              <p role='alert' className='text-destructive text-sm'>
                {error || props.saveError}
              </p>
            )}
            {editor && (
              <fieldset disabled={editor.readOnly || busy} className='min-w-0'>
                <RequestRoutingRuleEditor
                  rule={editor.rule}
                  channels={props.channels.filter(
                    (channel) =>
                      editor.rule.models.length > 0 &&
                      editor.rule.models.some((model) =>
                        channel.models.includes(model)
                      )
                  )}
                  models={props.models}
                  groups={props.groups}
                  onChange={(rule) => {
                    setEditor({ ...editor, rule })
                    setError('')
                  }}
                  readOnly={editor.readOnly}
                />
              </fieldset>
            )}
          </div>
          <SheetFooter className={sideDrawerFooterClassName()}>
            <Button
              type='button'
              variant='outline'
              disabled={busy}
              onClick={() => setEditor(null)}
            >
              {t(editor?.readOnly ? 'Close' : 'Cancel')}
            </Button>
            {editor?.readOnly ? (
              <Button
                type='button'
                onClick={() => setEditor({ ...editor, readOnly: false })}
              >
                <Pencil data-icon='inline-start' />
                {t('Edit')}
              </Button>
            ) : (
              <Button type='button' disabled={busy} onClick={commitRule}>
                {t(busy ? 'Saving...' : 'Save Changes')}
              </Button>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <Dialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open && !busy) setDeleting(null)
        }}
        title={t('Confirm delete')}
        contentClassName='sm:max-w-md'
        footer={
          <>
            <Button
              type='button'
              variant='outline'
              disabled={busy}
              onClick={() => setDeleting(null)}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              disabled={busy}
              onClick={deleteRule}
            >
              {t(busy ? 'Saving...' : 'Delete')}
            </Button>
          </>
        }
      >
        {props.saveError && (
          <p role='alert' className='text-destructive text-sm'>
            {props.saveError}
          </p>
        )}
        <p className='text-sm break-words'>
          {deleting !== null ? props.rules[deleting]?.id : ''}
        </p>
      </Dialog>
    </div>
  )
}
