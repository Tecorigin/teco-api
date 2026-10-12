import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { getChannels } from '@/features/channels/api'

import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { RequestRoutingRuleList } from './request-routing-rule-list'
import {
  emptyRoutingConfiguration,
  routingConfigurationSchema,
} from './request-routing-schema'
import type { RoutingConfiguration } from './request-routing-schema'

function parseConfiguration(raw: string): RoutingConfiguration | null {
  try {
    const source: unknown = JSON.parse(raw)
    if (
      source &&
      typeof source === 'object' &&
      'rules' in source &&
      Array.isArray(source.rules)
    ) {
      for (const rule of source.rules) {
        if (!rule || typeof rule !== 'object') continue
        if (
          !Array.isArray(rule.ranges) &&
          typeof rule.input_token_threshold === 'number'
        ) {
          rule.ranges = [
            {
              min_tokens: 0,
              max_tokens: rule.input_token_threshold,
              instances: rule.short_pool ?? [],
            },
            {
              min_tokens: rule.input_token_threshold,
              max_tokens: null,
              instances: rule.long_pool ?? [],
            },
          ]
        }
        delete rule.paths
        delete rule.input_token_threshold
        delete rule.short_pool
        delete rule.long_pool
        delete rule.default_output_tokens
        delete rule.output_budget_cap
        delete rule.input_weight
        delete rule.output_weight
        if (Array.isArray(rule.ranges)) {
          for (const interval of rule.ranges) {
            if (!interval || !Array.isArray(interval.instances)) continue
            for (const instance of interval.instances) {
              if (instance && typeof instance === 'object')
                delete instance.work_capacity_units
            }
          }
        }
      }
    }
    const parsed = routingConfigurationSchema.safeParse(source)
    if (!parsed.success) return null
    const configuration = parsed.data
    delete configuration.version
    delete configuration.state_backend
    return configuration
  } catch {
    return null
  }
}

export function RequestRoutingSection(props: { value: string }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const source = props.value || JSON.stringify(emptyRoutingConfiguration)
  const parsedSource = parseConfiguration(source)
  const initial = parsedSource ? JSON.stringify(parsedSource, null, 2) : source
  const [raw, setRaw] = useState(initial)
  const [configuration, setConfiguration] = useState(() =>
    parseConfiguration(initial)
  )
  const [mode, setMode] = useState<'visual' | 'json'>(() =>
    parseConfiguration(initial) ? 'visual' : 'json'
  )
  const [error, setError] = useState('')
  const channels = useQuery({
    queryKey: ['routing-channel-options'],
    queryFn: async () => {
      const options: {
        id: number
        name: string
        models: string
        group: string
      }[] = []
      for (let page = 1; ; page++) {
        const response = await getChannels({
          p: page,
          page_size: 100,
          tag_mode: false,
        })
        if (!response.success || !response.data)
          throw new Error(response.message || 'Failed to load channels')
        const items = response.data.items
        options.push(
          ...items.map((channel) => ({
            id: channel.id,
            name: channel.name,
            models: channel.models,
            group: channel.group,
          }))
        )
        if (items.length === 0 || options.length >= response.data.total)
          return options
      }
    },
    staleTime: 60000,
  })
  useEffect(() => {
    setRaw(initial)
    const parsed = parseConfiguration(initial)
    setConfiguration(parsed)
    setMode(parsed ? 'visual' : 'json')
    setError('')
  }, [initial])
  const switchMode = (value: string) => {
    if (value === 'visual') {
      const parsed = parseConfiguration(raw)
      if (!parsed) {
        setError(t('Invalid routing configuration'))
        return
      }
      setConfiguration(parsed)
    }
    setMode(value === 'json' ? 'json' : 'visual')
    setError('')
  }
  const persist = async (value: RoutingConfiguration): Promise<boolean> => {
    setError('')
    try {
      const result = await updateOption.mutateAsync({
        key: 'RequestRouting',
        value: JSON.stringify(value),
      })
      if (!result.success) {
        setError(result.message || t('Failed to update setting'))
        return false
      }
      setConfiguration(value)
      setRaw(JSON.stringify(value, null, 2))
      return true
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : t('Failed to update setting')
      )
      return false
    }
  }
  const save = async () => {
    const value = parseConfiguration(raw)
    if (!value) {
      setError(t('Invalid routing configuration'))
      return
    }
    await persist(value)
  }
  const channelOptions = (channels.data ?? []).map((channel) => ({
    value: String(channel.id),
    label: `#${channel.id} ${channel.name}`,
    models: channel.models.split(',').map((model) => model.trim()),
  }))
  const modelOptions = [
    ...new Set(
      (channels.data ?? [])
        .flatMap((channel) => channel.models.split(','))
        .filter(Boolean)
    ),
  ].map((value) => ({ value, label: value }))
  const groupOptions = [
    ...new Set(
      [
        'default',
        'auto',
        ...(channels.data ?? []).flatMap((channel) => channel.group.split(',')),
      ].filter(Boolean)
    ),
  ].map((value) => ({ value, label: value }))
  return (
    <SettingsSection title={t('Request Routing')}>
      <Tabs value={mode} onValueChange={switchMode}>
        <TabsList>
          <TabsTrigger value='visual'>{t('Visual')}</TabsTrigger>
          <TabsTrigger value='json'>JSON</TabsTrigger>
        </TabsList>
      </Tabs>
      {error && (
        <p role='alert' className='text-destructive text-sm'>
          {error}
        </p>
      )}
      <fieldset
        disabled={updateOption.isPending}
        className='min-w-0 space-y-5 disabled:opacity-60'
      >
        {mode === 'json' ? (
          <label className='flex flex-col gap-2 text-sm'>
            <span>{t('JSON Configuration')}</span>
            <Textarea
              value={raw}
              onChange={(event) => {
                setRaw(event.target.value)
                setError('')
              }}
              className='min-h-96 font-mono'
              spellCheck={false}
            />
          </label>
        ) : (
          configuration && (
            <>
              <div className='flex items-center'>
                <label className='flex h-8 items-center gap-3 text-sm'>
                  <Switch
                    checked={configuration.enabled}
                    onCheckedChange={(enabled) =>
                      persist({ ...configuration, enabled })
                    }
                  />
                  {t('Enabled')}
                </label>
              </div>
              {channels.isError && (
                <div className='text-destructive flex items-center gap-3 text-sm'>
                  <span>{t('Failed to load channels')}</span>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => channels.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}
              {channels.isPending && (
                <p role='status' className='text-muted-foreground text-sm'>
                  {t('Loading...')}
                </p>
              )}
              <RequestRoutingRuleList
                rules={configuration.rules}
                channels={channelOptions}
                models={modelOptions}
                groups={groupOptions}
                isSaving={updateOption.isPending}
                saveError={error}
                onChange={(rules) => persist({ ...configuration, rules })}
              />
            </>
          )
        )}
      </fieldset>
      {mode === 'json' && (
        <SettingsPageFormActions
          onSave={save}
          onReset={() => {
            setRaw(initial)
            setConfiguration(parseConfiguration(initial))
            setMode(parseConfiguration(initial) ? 'visual' : 'json')
            setError('')
          }}
          isSaving={updateOption.isPending}
          isSaveDisabled={raw === initial}
        />
      )}
    </SettingsSection>
  )
}
