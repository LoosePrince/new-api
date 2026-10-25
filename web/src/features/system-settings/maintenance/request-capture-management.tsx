import { Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { handleServerError } from '@/lib/handle-server-error'

export function RequestCaptureManagement() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const clearCaptures = async () => {
    setIsDeleting(true)
    try {
      const response = await api.delete('/api/request-captures/')
      if (!response.data?.success) {
        throw new Error(response.data?.message || t('Failed to clear request records'))
      }
      toast.success(t('Request records cleared.'))
      setOpen(false)
    } catch (error) {
      handleServerError(error, t('Failed to clear request records'))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <div className='flex min-w-0 flex-col gap-4'>
      <div>
        <h4 className='font-medium'>{t('Request record management')}</h4>
        <p className='text-muted-foreground mt-1 text-xs'>
          {t('Request records are stored separately from usage logs and can be cleared without deleting usage statistics.')}
        </p>
      </div>
      <Button
        type='button'
        variant='destructive'
        className='w-fit'
        onClick={() => setOpen(true)}
        disabled={isDeleting}
      >
        <Trash2 aria-hidden='true' />
        {t('Clear all request records')}
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={t('Clear all request records')}
        desc={t('This permanently removes every captured request, upstream request, and upstream response. This action cannot be undone.')}
        destructive
        isLoading={isDeleting}
        confirmText={t('Delete records')}
        handleConfirm={clearCaptures}
      />
    </div>
  )
}
