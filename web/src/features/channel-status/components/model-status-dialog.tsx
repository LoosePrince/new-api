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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Switch } from '@/components/ui/switch'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampRelative } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { CHANNEL_STATUS_QUERY_KEY, updateChannelModelStatus } from '../api'
import {
  CHANNEL_HEALTH_STATUS_META,
  canRestoreModel,
  formatAvailability,
} from '../lib/status'
import type { ChannelStatusChannel, ChannelStatusModel } from '../types'
import { StatusHistoryBlocks } from './status-history-blocks'

type ModelRowProps = {
  model: ChannelStatusModel
  locale?: Intl.LocalesArgument
  busy: boolean
  onToggle: (model: ChannelStatusModel) => void
}

function ModelRow(props: ModelRowProps) {
  const { t } = useTranslation()
  const meta = CHANNEL_HEALTH_STATUS_META[props.model.status]
  const restorable = canRestoreModel(props.model)
  const switchDisabled = props.busy || (!props.model.enabled && !restorable)

  return (
    <div className='space-y-2 border-b py-3 last:border-b-0'>
      <div className='flex flex-wrap items-start justify-between gap-x-3 gap-y-2'>
        <div className='min-w-0 space-y-1.5'>
          <div className='flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1'>
            <span className='truncate font-mono text-sm font-medium'>
              {props.model.name}
            </span>
            <StatusBadge
              label={t(meta.labelKey)}
              variant={meta.variant}
              copyable={false}
            />
            {!props.model.enabled && (
              <StatusBadge
                label={restorable ? t('Removed, restorable') : t('Removed')}
                variant='neutral'
                copyable={false}
              />
            )}
          </div>
          <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
            <span>
              {t('7-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatAvailability(props.model.availability_7d, props.locale)}
              </span>
            </span>
            <span>
              {t('30-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatAvailability(props.model.availability_30d, props.locale)}
              </span>
            </span>
            <span>
              {t('Attempts')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatNumber(props.model.attempts_7d, props.locale)}
              </span>
            </span>
            <span>
              {t('Average latency')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {props.model.latency_ms > 0
                  ? `${formatNumber(props.model.latency_ms, props.locale)} ms`
                  : '—'}
              </span>
            </span>
            <span>
              {t('Last activity')}:{' '}
              <span className='text-foreground font-medium'>
                {formatTimestampRelative(
                  props.model.last_activity,
                  'seconds',
                  props.locale
                )}
              </span>
            </span>
          </div>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          {!props.model.enabled && !restorable && (
            <span className='text-muted-foreground text-xs'>
              {t('No request history, cannot restore')}
            </span>
          )}
          <Switch
            size='sm'
            checked={props.model.enabled}
            disabled={switchDisabled}
            aria-label={t('Enable {{model}}', { model: props.model.name })}
            onCheckedChange={() => props.onToggle(props.model)}
          />
        </div>
      </div>
      <StatusHistoryBlocks days={props.model.history} size='sm' />
    </div>
  )
}

type ModelStatusDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel: ChannelStatusChannel | null
}

/**
 * Model-level status of one channel. Removing a model changes the channel's
 * model list, so it asks for confirmation; restoring is applied immediately.
 */
export function ModelStatusDialog(props: ModelStatusDialogProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const client = useQueryClient()
  const [removeTarget, setRemoveTarget] = useState<ChannelStatusModel | null>(
    null
  )

  const mutation = useMutation({
    mutationFn: (input: { model: string; enable: boolean }) => {
      if (!props.channel) throw new Error('Channel is not selected')
      return updateChannelModelStatus(
        props.channel.id,
        input.model,
        input.enable
      )
    },
    onSuccess: async (_data, input) => {
      setRemoveTarget(null)
      toast.success(
        input.enable
          ? t('{{model}} restored', { model: input.model })
          : t('{{model}} removed', { model: input.model })
      )
      await client.invalidateQueries({ queryKey: CHANNEL_STATUS_QUERY_KEY })
    },
    onError: (error) => {
      handleServerError(error, t('Failed to update model status'))
    },
  })

  const channel = props.channel
  const enabledCount =
    channel?.models.filter((item) => item.enabled).length ?? 0
  const description = channel
    ? t(
        '{{enabled}} of {{total}} models enabled. Availability is derived from request logs.',
        { enabled: enabledCount, total: channel.models.length }
      )
    : undefined

  const handleToggle = (model: ChannelStatusModel) => {
    if (model.enabled) {
      setRemoveTarget(model)
      return
    }
    mutation.mutate({ model: model.name, enable: true })
  }

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={
          channel
            ? t('Model status of {{channel}}', { channel: channel.name })
            : t('Model status')
        }
        description={description}
        contentHeight='60vh'
        contentClassName='sm:max-w-3xl'
        footer={
          <p className='text-muted-foreground text-xs'>
            {t(
              'Removing a model stops routing to it immediately. Only models with request history can be restored.'
            )}
          </p>
        }
      >
        {channel && channel.models.length > 0 ? (
          <div className='min-w-0'>
            {channel.models.map((model) => (
              <ModelRow
                key={model.name}
                model={model}
                locale={locale}
                busy={
                  mutation.isPending && mutation.variables?.model === model.name
                }
                onToggle={handleToggle}
              />
            ))}
          </div>
        ) : (
          <p className='text-muted-foreground py-6 text-center text-sm'>
            {t('No models')}
          </p>
        )}
      </Dialog>
      <ConfirmDialog
        open={removeTarget != null}
        onOpenChange={(open) => {
          if (!open) setRemoveTarget(null)
        }}
        title={t('Remove {{model}}?', { model: removeTarget?.name ?? '' })}
        desc={t(
          'This removes the model from the channel immediately and stops routing to it. It can be restored later while request history exists.'
        )}
        confirmText={t('Remove')}
        destructive
        isLoading={mutation.isPending}
        handleConfirm={() => {
          if (removeTarget) {
            mutation.mutate({ model: removeTarget.name, enable: false })
          }
        }}
      />
    </>
  )
}
