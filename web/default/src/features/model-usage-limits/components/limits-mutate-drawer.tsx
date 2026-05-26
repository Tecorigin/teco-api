import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { createLimit, getLimitModelOptions, getLimitUserOptions, updateLimit } from '../api'
import type { LimitItem, LimitFormValues } from '../types'
import { LIMIT_FORM_DEFAULTS } from '../types'
import { useLimits } from './limits-provider'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: LimitItem
}

export function LimitsMutateDrawer({ open, onOpenChange, currentRow }: Props) {
  const { t } = useTranslation()
  const { triggerRefresh } = useLimits()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [form, setForm] = useState<LimitFormValues>(LIMIT_FORM_DEFAULTS)

  const isUpdate = !!currentRow

  const { data: userOptions = [] } = useQuery({
    queryKey: ['limit-user-options'],
    queryFn: getLimitUserOptions,
    staleTime: 5 * 60 * 1000,
  })

  const { data: modelOptions = [] } = useQuery({
    queryKey: ['limit-model-options'],
    queryFn: getLimitModelOptions,
    staleTime: 5 * 60 * 1000,
  })

  useEffect(() => {
    if (open) {
      if (currentRow) {
        setForm({
          user_id: String(currentRow.user_id),
          model_name: currentRow.model_name,
          daily_token_limit: String(currentRow.daily_token_limit),
          daily_request_limit: String(currentRow.daily_request_limit),
          weekly_token_limit: String(currentRow.weekly_token_limit),
          weekly_request_limit: String(currentRow.weekly_request_limit),
          monthly_token_limit: String(currentRow.monthly_token_limit),
          monthly_request_limit: String(currentRow.monthly_request_limit),
        })
      } else {
        setForm(LIMIT_FORM_DEFAULTS)
      }
    }
  }, [open, currentRow])

  const handleSubmit = async () => {
    if (!form.user_id || !form.model_name) {
      toast.error(t('User ID and Model Name are required'))
      return
    }
    setIsSubmitting(true)
    try {
      const payload = {
        user_id: Number(form.user_id),
        model_name: form.model_name,
        daily_token_limit: Number(form.daily_token_limit) || 0,
        daily_request_limit: Number(form.daily_request_limit) || 0,
        weekly_token_limit: Number(form.weekly_token_limit) || 0,
        weekly_request_limit: Number(form.weekly_request_limit) || 0,
        monthly_token_limit: Number(form.monthly_token_limit) || 0,
        monthly_request_limit: Number(form.monthly_request_limit) || 0,
      }
      if (isUpdate) {
        await updateLimit(currentRow!.id, payload)
        toast.success(t('Limit updated'))
      } else {
        await createLimit(payload)
        toast.success(t('Limit created'))
      }
      triggerRefresh()
      onOpenChange(false)
    } catch (e: any) {
      toast.error(e?.response?.data?.message || e?.message || t('Operation failed'))
    }
    setIsSubmitting(false)
  }

  const setField = (key: keyof LimitFormValues, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isUpdate ? t('Edit Limit') : t('Create Limit')}</SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the usage limit for this user and model.')
              : t('Set a usage limit for a specific user and model.')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-y-auto px-4 space-y-6'>
          {/* Basic info */}
          <div className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>{t('User ID')} *</Label>
                <Select
                  value={form.user_id || undefined}
                  onValueChange={(value) => setField('user_id', value)}
                  disabled={isUpdate}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('Select existing user')} />
                  </SelectTrigger>
                  <SelectContent>
                    {userOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-2'>
                <Label>{t('Model Name')} *</Label>
                <Select
                  value={form.model_name || undefined}
                  onValueChange={(value) => setField('model_name', value)}
                  disabled={isUpdate}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('Select existing model')} />
                  </SelectTrigger>
                  <SelectContent>
                    {modelOptions.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          {/* Daily limits */}
          <div className='space-y-3'>
            <h4 className='text-sm font-medium text-muted-foreground'>{t('Daily Limits')}</h4>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Tokens</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.daily_token_limit}
                  onChange={(e) => setField('daily_token_limit', e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Request')}</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.daily_request_limit}
                  onChange={(e) => setField('daily_request_limit', e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Weekly limits */}
          <div className='space-y-3'>
            <h4 className='text-sm font-medium text-muted-foreground'>{t('Weekly Limits')}</h4>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Tokens</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.weekly_token_limit}
                  onChange={(e) => setField('weekly_token_limit', e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Request')}</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.weekly_request_limit}
                  onChange={(e) => setField('weekly_request_limit', e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Monthly limits */}
          <div className='space-y-3'>
            <h4 className='text-sm font-medium text-muted-foreground'>{t('Monthly Limits')}</h4>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Tokens</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.monthly_token_limit}
                  onChange={(e) => setField('monthly_token_limit', e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Request')}</Label>
                <Input
                  type='number'
                  min={0}
                  value={form.monthly_request_limit}
                  onChange={(e) => setField('monthly_request_limit', e.target.value)}
                />
              </div>
            </div>
          </div>
        </div>

        <SheetFooter className='px-4'>
          <SheetClose asChild>
            <Button variant='outline'>{t('Cancel')}</Button>
          </SheetClose>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? t('Saving...') : isUpdate ? t('Save Changes') : t('Create')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
