import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'

import { SettingsSection } from '../components/settings-section'
import {
  deleteUserRateLimit,
  getRateLimitGroups,
  getRateLimitModels,
  getRateLimitUsers,
  getUserRateLimits,
  saveUserRateLimit,
  type UserRateLimit,
  type UserRateLimitPayload,
  type UserRateLimitScope,
} from './user-rate-limit-api'

const emptyForm: UserRateLimitPayload = {
  scope: 'global',
  user_id: 0,
  group_name: '',
  model_name: '',
  rpm: 0,
  tpm: 0,
}

function scopeLabel(t: (key: string) => string, scope: UserRateLimitScope) {
  if (scope === 'group_model') {
    return t('User group and model')
  }
  return t(
    {
      global: 'Global default',
      group: 'User group',
      user: 'User',
      model: 'Model',
      user_model: 'User and model',
    }[scope]
  )
}

function RateLimitTarget({
  form,
  groups,
  users,
  models,
  onChange,
}: {
  form: UserRateLimitPayload
  groups: string[]
  users: { label: string; value: string }[]
  models: string[]
  onChange: (next: UserRateLimitPayload) => void
}) {
  const { t } = useTranslation()
  if (form.scope === 'global') {
    return (
      <div className='bg-muted flex h-9 items-center rounded-md px-3 text-sm'>
        {t('Every user')}
      </div>
    )
  }
  if (form.scope === 'model') {
    return (
      <Combobox
        className='w-full'
        emptyText={t('No option found.')}
        onValueChange={(modelName) =>
          onChange({ ...form, model_name: modelName ?? '' })
        }
        options={models.map((model) => ({ label: model, value: model }))}
        placeholder={t('Select a model')}
        value={form.model_name}
      />
    )
  }
  if (form.scope === 'group' || form.scope === 'group_model') {
    return (
      <Combobox
        className='w-full'
        emptyText={t('No option found.')}
        onValueChange={(groupName) =>
          onChange({ ...form, group_name: groupName ?? '' })
        }
        options={groups.map((group) => ({ label: group, value: group }))}
        placeholder={t('Select a user group')}
        value={form.group_name}
      />
    )
  }
  return (
    <Combobox
      className='w-full'
      emptyText={t('No option found.')}
      onValueChange={(userId) => onChange({ ...form, user_id: Number(userId) })}
      options={users}
      placeholder={t('Select a user')}
      value={form.user_id ? String(form.user_id) : ''}
    />
  )
}

function getLimitTargetLabel(t: (key: string) => string, limit: UserRateLimit) {
  if (limit.scope === 'group' || limit.scope === 'group_model') {
    return limit.group_name
  }
  if (limit.user_id) return `${limit.username} (${limit.user_id})`
  return t('Every user')
}

export function UserRateLimitSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<UserRateLimitPayload>(emptyForm)
  const [editingId, setEditingId] = useState<number | undefined>()
  const needsUser = form.scope === 'user' || form.scope === 'user_model'
  const needsGroup = form.scope === 'group' || form.scope === 'group_model'
  const needsModel =
    form.scope === 'model' ||
    form.scope === 'group_model' ||
    form.scope === 'user_model'
  const showModelSelector =
    form.scope === 'group_model' || form.scope === 'user_model'
  const limitsQuery = useQuery({
    queryKey: ['user-rate-limits'],
    queryFn: getUserRateLimits,
  })
  const usersQuery = useQuery({
    queryKey: ['rate-limit-users'],
    queryFn: getRateLimitUsers,
    staleTime: 300000,
  })
  const groupsQuery = useQuery({
    queryKey: ['rate-limit-groups'],
    queryFn: getRateLimitGroups,
    staleTime: 300000,
  })
  const modelsQuery = useQuery({
    queryKey: ['rate-limit-models'],
    queryFn: getRateLimitModels,
    staleTime: 300000,
  })

  const saveMutation = useMutation({
    mutationFn: () => saveUserRateLimit(form, editingId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['user-rate-limits'] })
      setForm(emptyForm)
      setEditingId(undefined)
      toast.success(t('Rate limit saved'))
    },
    onError: (error: Error) =>
      toast.error(error.message || t('Operation failed')),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteUserRateLimit,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['user-rate-limits'] }),
    onError: (error: Error) =>
      toast.error(error.message || t('Operation failed')),
  })

  useEffect(() => {
    setForm((current) => ({
      ...current,
      user_id:
        current.scope === 'user' || current.scope === 'user_model'
          ? current.user_id
          : 0,
      group_name:
        current.scope === 'group' || current.scope === 'group_model'
          ? current.group_name
          : '',
      model_name: ['model', 'group_model', 'user_model'].includes(current.scope)
        ? current.model_name
        : '',
    }))
  }, [form.scope])

  const beginEdit = (limit: UserRateLimit) => {
    setEditingId(limit.id)
    setForm({
      scope: limit.scope,
      user_id: limit.user_id,
      group_name: limit.group_name,
      model_name: limit.model_name,
      rpm: limit.rpm,
      tpm: limit.tpm,
    })
  }

  const save = () => {
    if (needsUser && form.user_id <= 0) {
      toast.error(t('Select a user'))
      return
    }
    if (needsGroup && !form.group_name) {
      toast.error(t('Select a user group'))
      return
    }
    if (needsModel && !form.model_name) {
      toast.error(t('Select a model'))
      return
    }
    if (
      ![form.rpm, form.tpm].every(
        (value) => Number.isInteger(value) && value >= 0 && value <= 2147483647
      )
    ) {
      toast.error(t('Please enter a valid number'))
      return
    }
    saveMutation.mutate()
  }

  const submitLabel = editingId ? t('Save Changes') : t('Add')
  let submitAriaLabel = t('Add rate limit')
  if (editingId) submitAriaLabel = t('Save Changes')

  return (
    <SettingsSection title={t('RPM/TPM Limits')}>
      <div className='space-y-5'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Set per-minute request and token limits. More specific rules override less specific rules; 0 means unlimited.'
          )}
        </p>
        <div
          className={
            showModelSelector
              ? 'grid gap-4 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_100px_100px_auto]'
              : 'grid gap-4 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_120px_120px_auto]'
          }
        >
          <div className='space-y-2'>
            <Label>{t('Apply to')}</Label>
            <Select<UserRateLimitScope>
              value={form.scope}
              onValueChange={(scope) => {
                if (scope !== null) {
                  setForm({ ...form, scope })
                }
              }}
            >
              <SelectTrigger>
                <span className='truncate'>{scopeLabel(t, form.scope)}</span>
              </SelectTrigger>
              <SelectContent>
                {(
                  [
                    'global',
                    'group',
                    'user',
                    'model',
                    'group_model',
                    'user_model',
                  ] as const
                ).map((scope) => (
                  <SelectItem key={scope} value={scope}>
                    {scopeLabel(t, scope)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className='space-y-2'>
            <Label>{t('Target')}</Label>
            <RateLimitTarget
              form={form}
              groups={groupsQuery.data ?? []}
              users={usersQuery.data ?? []}
              models={modelsQuery.data ?? []}
              onChange={setForm}
            />
          </div>
          {showModelSelector && (
            <div className='space-y-2'>
              <Label>{t('Model')}</Label>
              <Combobox
                className='w-full'
                emptyText={t('No option found.')}
                onValueChange={(modelName) =>
                  setForm({ ...form, model_name: modelName ?? '' })
                }
                options={(modelsQuery.data ?? []).map((model) => ({
                  label: model,
                  value: model,
                }))}
                placeholder={t('Select a model')}
                value={form.model_name}
              />
            </div>
          )}
          <div className='space-y-2'>
            <Label>RPM</Label>
            <Input
              type='number'
              min={0}
              max={2147483647}
              step={1}
              value={form.rpm}
              onChange={(event) =>
                setForm({ ...form, rpm: Number(event.target.value) || 0 })
              }
            />
          </div>
          <div className='space-y-2'>
            <Label>TPM</Label>
            <Input
              type='number'
              min={0}
              max={2147483647}
              step={1}
              value={form.tpm}
              onChange={(event) =>
                setForm({ ...form, tpm: Number(event.target.value) || 0 })
              }
            />
          </div>
          <div className='flex items-end gap-2'>
            <Button
              type='button'
              onClick={save}
              disabled={saveMutation.isPending}
              aria-label={submitAriaLabel}
            >
              <Plus className='size-4' />
              <span className='ml-1'>{submitLabel}</span>
            </Button>
            {editingId && (
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  setForm(emptyForm)
                  setEditingId(undefined)
                }}
              >
                {t('Cancel')}
              </Button>
            )}
          </div>
        </div>

        <div className='overflow-x-auto border-t pt-4'>
          <table className='w-full min-w-[680px] text-sm'>
            <thead className='text-muted-foreground border-b text-left'>
              <tr>
                <th className='px-2 py-2'>{t('Scope')}</th>
                <th className='px-2 py-2'>{t('Target')}</th>
                <th className='px-2 py-2'>{t('Model')}</th>
                <th className='px-2 py-2'>RPM</th>
                <th className='px-2 py-2'>TPM</th>
                <th className='px-2 py-2 text-right'>{t('Actions')}</th>
              </tr>
            </thead>
            <tbody>
              {limitsQuery.data?.map((limit) => (
                <tr key={limit.id} className='border-b last:border-0'>
                  <td className='px-2 py-3'>{scopeLabel(t, limit.scope)}</td>
                  <td className='px-2 py-3'>{getLimitTargetLabel(t, limit)}</td>
                  <td className='px-2 py-3 font-mono'>
                    {limit.model_name || '—'}
                  </td>
                  <td className='px-2 py-3 tabular-nums'>{limit.rpm || '—'}</td>
                  <td className='px-2 py-3 tabular-nums'>{limit.tpm || '—'}</td>
                  <td className='px-2 py-3'>
                    <div className='flex justify-end gap-1'>
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon-sm'
                        aria-label={t('Edit')}
                        onClick={() => beginEdit(limit)}
                      >
                        <Pencil className='size-4' />
                      </Button>
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon-sm'
                        className='text-destructive hover:text-destructive'
                        aria-label={t('Delete')}
                        onClick={() => deleteMutation.mutate(limit.id)}
                      >
                        <Trash2 className='size-4' />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {!limitsQuery.isLoading && !limitsQuery.data?.length && (
                <tr>
                  <td
                    colSpan={6}
                    className='text-muted-foreground px-2 py-8 text-center'
                  >
                    {t('No rate limits configured')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </SettingsSection>
  )
}
