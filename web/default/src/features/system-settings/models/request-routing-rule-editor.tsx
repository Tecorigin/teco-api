import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import type { Option } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import type { RoutingInstance, RoutingRule } from './request-routing-schema'
import { newRoutingInstance } from './request-routing-schema'

function NumericField(props: {
  label: string
  value: number
  min?: number
  max?: number
  step?: number
  onChange: (value: number) => void
}) {
  return (
    <label className='flex min-w-0 flex-col gap-2 text-sm'>
      <span>{props.label}</span>
      <Input
        type='number'
        value={Number.isNaN(props.value) ? '' : props.value}
        min={props.min ?? 1}
        max={props.max}
        step={props.step ?? 1}
        onChange={(event) => props.onChange(event.target.valueAsNumber)}
      />
    </label>
  )
}

function InstancePool(props: {
  title: string
  instances: RoutingInstance[]
  channels: Option[]
  onChange: (instances: RoutingInstance[]) => void
  readOnly?: boolean
}) {
  const { t } = useTranslation()
  const update = (index: number, patch: Partial<RoutingInstance>) => {
    props.onChange(
      props.instances.map((instance, current) =>
        current === index ? { ...instance, ...patch } : instance
      )
    )
  }
  return (
    <section className='min-w-0 space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h4 className='text-sm font-semibold'>{props.title}</h4>
        {!props.readOnly && (
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={props.instances.length >= 100}
            onClick={() =>
              props.onChange([...props.instances, newRoutingInstance()])
            }
          >
            <Plus data-icon='inline-start' />
            {t('Add Instance')}
          </Button>
        )}
      </div>
      {props.instances.length === 0 && (
        <p className='text-muted-foreground py-3 text-sm'>
          {t('No instances configured')}
        </p>
      )}
      {props.instances.map((instance, index) => (
        <div key={index} className='space-y-3 rounded-md border p-3'>
          <div className='flex items-end gap-2'>
            <label className='flex min-w-0 flex-1 flex-col gap-2 text-sm'>
              <span>{t('Instance ID')}</span>
              <Input
                value={instance.instance_id}
                maxLength={100}
                onChange={(event) =>
                  update(index, { instance_id: event.target.value })
                }
              />
            </label>
            {!props.readOnly && (
              <Button
                type='button'
                variant='ghost'
                size='icon'
                title={t('Remove Instance')}
                aria-label={t('Remove Instance')}
                onClick={() =>
                  props.onChange(
                    props.instances.filter((_, current) => current !== index)
                  )
                }
              >
                <Trash2 className='size-4' />
              </Button>
            )}
          </div>
          <div className='space-y-2'>
            <label
              htmlFor={`channels-${instance.instance_id}-${index}`}
              className='text-sm'
            >
              {t('Channels')}
            </label>
            <MultiSelect
              id={`channels-${instance.instance_id}-${index}`}
              options={props.channels}
              selected={instance.channel_ids.map(String)}
              onChange={(values) =>
                update(index, { channel_ids: values.map(Number) })
              }
            />
          </div>
          {instance.channel_ids.some(
            (id) =>
              !props.channels.some((option) => option.value === String(id))
          ) && (
            <p className='text-destructive text-sm'>
              {t('Selected channels do not support any selected model')}
            </p>
          )}
          <div className='grid gap-3 sm:grid-cols-2'>
            <label className='flex min-w-0 flex-col gap-2 text-sm'>
              <span>{t('Maximum Inflight Requests')}</span>
              <Input
                type='number'
                min={0}
                max={10000}
                placeholder={t('Unlimited')}
                value={instance.max_inflight === 0 ? '' : instance.max_inflight}
                onChange={(event) =>
                  update(index, {
                    max_inflight:
                      event.target.value === ''
                        ? 0
                        : event.target.valueAsNumber,
                  })
                }
              />
            </label>
          </div>
        </div>
      ))}
    </section>
  )
}

export function RequestRoutingRuleEditor(props: {
  rule: RoutingRule
  channels: Option[]
  models: Option[]
  groups: Option[]
  onChange: (rule: RoutingRule) => void
  readOnly?: boolean
}) {
  const { t } = useTranslation()
  const update = (patch: Partial<RoutingRule>) =>
    props.onChange({ ...props.rule, ...patch })
  return (
    <section className='space-y-5 border-t py-5'>
      <div className='flex items-end gap-3'>
        <label className='flex min-w-0 flex-1 flex-col gap-2 text-sm'>
          <span>{t('Rule Name')}</span>
          <Input
            value={props.rule.id}
            onChange={(event) => update({ id: event.target.value })}
          />
        </label>
      </div>
      <div className='grid gap-4 md:grid-cols-2'>
        <div className='space-y-2'>
          <label htmlFor={`models-${props.rule.id}`} className='text-sm'>
            {t('Models')}
          </label>
          <MultiSelect
            id={`models-${props.rule.id}`}
            options={props.models}
            selected={props.rule.models}
            allowCreate
            onChange={(models) => update({ models })}
          />
        </div>
        <div className='space-y-2'>
          <label htmlFor={`groups-${props.rule.id}`} className='text-sm'>
            {t('Groups')}
          </label>
          <MultiSelect
            id={`groups-${props.rule.id}`}
            options={props.groups}
            selected={props.rule.groups}
            placeholder={t('All Groups')}
            allowCreate
            onChange={(groups) => update({ groups })}
          />
        </div>
      </div>
      <div className='space-y-6'>
        {props.rule.ranges.map((interval, index) => {
          const changeRange = (patch: Partial<typeof interval>) =>
            update({
              ranges: props.rule.ranges.map((current, position) =>
                position === index ? { ...current, ...patch } : current
              ),
            })
          return (
            <section key={index} className='space-y-4 border-t pt-4'>
              <div className='flex items-end gap-3'>
                <div className='grid flex-1 gap-3 sm:grid-cols-2'>
                  <NumericField
                    label={t('Minimum Input Tokens (inclusive)')}
                    value={interval.min_tokens}
                    min={0}
                    max={10000000}
                    onChange={(min_tokens) => changeRange({ min_tokens })}
                  />
                  <label className='flex flex-col gap-2 text-sm'>
                    <span>{t('Maximum Input Tokens (exclusive)')}</span>
                    <Input
                      type='number'
                      min={1}
                      max={10000000}
                      placeholder={t('Unlimited')}
                      value={interval.max_tokens ?? ''}
                      onChange={(event) =>
                        changeRange({
                          max_tokens:
                            event.target.value === ''
                              ? null
                              : event.target.valueAsNumber,
                        })
                      }
                    />
                  </label>
                </div>
                {!props.readOnly && (
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    title={t('Remove Range')}
                    aria-label={t('Remove Range')}
                    onClick={() =>
                      update({
                        ranges: props.rule.ranges.filter(
                          (_, position) => position !== index
                        ),
                      })
                    }
                  >
                    <Trash2 className='size-4' />
                  </Button>
                )}
              </div>
              <InstancePool
                title={t('Instances')}
                instances={interval.instances}
                channels={props.channels}
                onChange={(instances) => changeRange({ instances })}
                readOnly={props.readOnly}
              />
            </section>
          )
        })}
        {!props.readOnly && (
          <Button
            type='button'
            variant='outline'
            disabled={props.rule.ranges.length >= 100}
            onClick={() =>
              update({
                ranges: [
                  ...props.rule.ranges,
                  { min_tokens: 0, max_tokens: null, instances: [] },
                ],
              })
            }
          >
            <Plus data-icon='inline-start' />
            {t('Add Token Range')}
          </Button>
        )}
      </div>
    </section>
  )
}
