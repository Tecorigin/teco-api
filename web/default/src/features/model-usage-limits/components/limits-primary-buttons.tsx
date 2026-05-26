import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useLimits } from './limits-provider'

export function LimitsPrimaryButtons() {
  const { t } = useTranslation()
  const { setOpen } = useLimits()

  return (
    <div className='flex items-center gap-2'>
      <Button size='sm' className='h-8 gap-1' onClick={() => setOpen('create')}>
        <Plus className='size-4' />
        <span className='hidden sm:inline'>{t('Create Limit')}</span>
      </Button>
    </div>
  )
}
