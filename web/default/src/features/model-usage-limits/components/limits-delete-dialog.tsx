import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { deleteLimit } from '../api'
import { useLimits } from './limits-provider'

export function LimitsDeleteDialog() {
  const { t } = useTranslation()
  const { deleteOpen, setDeleteOpen, deleteRow, setDeleteRow, triggerRefresh } = useLimits()
  const [isDeleting, setIsDeleting] = useState(false)

  const handleDelete = async () => {
    if (!deleteRow) return
    setIsDeleting(true)
    try {
      await deleteLimit(deleteRow.id as number)
      toast.success(t('Limit deleted'))
      triggerRefresh()
      setDeleteOpen(false)
      setDeleteRow(null)
    } catch (e: any) {
      toast.error(e?.response?.data?.message || e?.message || t('Delete failed'))
    }
    setIsDeleting(false)
  }

  return (
    <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('Are you sure?')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('This will permanently delete the usage limit for')}{' '}
            <strong>{deleteRow?.username as string}</strong> {t('on model')}{' '}
            <strong>{deleteRow?.model_name as string}</strong>.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => setDeleteRow(null)}>{t('Cancel')}</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            disabled={isDeleting}
          >
            {isDeleting ? t('Deleting...') : t('Delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
