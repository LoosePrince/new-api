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
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import {
  formatPublicAvailability,
  formatPublicLatency,
  PUBLIC_STATUS_META,
} from '../lib/status'
import type { PublicStatusModel } from '../types'
import { StatusStrip } from './status-strip'

interface PublicStatusModelCardProps {
  model: PublicStatusModel
  locale?: Intl.LocalesArgument
}

/**
 * Read-only presentation of one model's availability. Nothing here is
 * interactive on purpose: the public page must not reveal channel identifiers,
 * model configuration, or offer any edits.
 */
export function PublicStatusModelCard(props: PublicStatusModelCardProps) {
  const { t } = useTranslation()
  const meta = PUBLIC_STATUS_META[props.model.status]

  return (
    <div className='bg-card rounded-xl border p-4 sm:p-5'>
      <div className='flex flex-wrap items-start justify-between gap-x-3 gap-y-2'>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='truncate font-mono text-sm font-semibold'>
              {props.model.name}
            </span>
            <span
              className={cn(
                'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
                meta.pillClassName
              )}
            >
              <span
                className={cn('size-1.5 rounded-full', meta.blockClassName)}
                aria-hidden='true'
              />
              {t(meta.labelKey)}
            </span>
          </div>
          <div className='text-muted-foreground mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
            <span>
              {t('7-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatPublicAvailability(
                  props.model.availability_7d,
                  props.locale
                )}
              </span>
            </span>
            <span>
              {t('30-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatPublicAvailability(
                  props.model.availability_30d,
                  props.locale
                )}
              </span>
            </span>
            <span>
              {t('Average latency')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatPublicLatency(props.model.latency_ms, props.locale)}
              </span>
            </span>
          </div>
        </div>
      </div>

      <StatusStrip
        history={props.model.history}
        today={props.model.today ?? []}
        size='sm'
        className='mt-3'
      />

      {props.model.channels.length > 0 && (
        <div className='mt-3 space-y-1.5'>
          <div className='text-muted-foreground text-xs font-medium'>
            {t('Served by')}
          </div>
          <div className='flex flex-wrap gap-1.5'>
            {props.model.channels.map((channel) => {
              const channelMeta = PUBLIC_STATUS_META[channel.status]
              return (
                <span
                  key={channel.name}
                  title={t(
                    '{{availability}} availability, {{latency}} latency',
                    {
                      availability: formatPublicAvailability(
                        channel.availability_7d,
                        props.locale
                      ),
                      latency: formatPublicLatency(
                        channel.latency_ms,
                        props.locale
                      ),
                    }
                  )}
                  className='bg-muted/50 inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs'
                >
                  <span
                    className={cn(
                      'size-1.5 rounded-full',
                      channelMeta.blockClassName
                    )}
                    aria-hidden='true'
                  />
                  <span className='max-w-40 truncate'>{channel.name}</span>
                  <span className='text-muted-foreground tabular-nums'>
                    {formatPublicAvailability(
                      channel.availability_7d,
                      props.locale
                    )}
                  </span>
                </span>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
