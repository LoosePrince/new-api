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
  CHANNEL_HEALTH_STATUS_META,
  CHANNEL_HEALTH_STATUS_ORDER,
  formatAvailability,
} from '../lib/status'
import type { ChannelStatusDay, ChannelHealthStatus } from '../types'

type StatusHistoryBlocksProps = {
  days: ChannelStatusDay[]
  /** Compact size is used inside the model detail rows. */
  size?: 'sm' | 'md'
  className?: string
}

function dayLabel(
  t: (key: string, options?: Record<string, unknown>) => string,
  day: ChannelStatusDay
): string {
  const date = new Date(day.day * 1000).toISOString().slice(0, 10)
  if (day.attempts === 0) {
    return t('{{date}}: no requests', { date })
  }
  return t(
    '{{date}}: {{status}}, {{availability}} availability, {{attempts}} requests',
    {
      date,
      status: t(CHANNEL_HEALTH_STATUS_META[day.status].labelKey),
      availability: formatAvailability(day.availability),
      attempts: day.attempts,
    }
  )
}

/**
 * Daily status strip of the monitored window. Each block carries its own hover
 * text, and the strip exposes a text summary so the status is never conveyed by
 * color alone.
 */
export function StatusHistoryBlocks(props: StatusHistoryBlocksProps) {
  const { t } = useTranslation()
  const size = props.size ?? 'md'
  const healthyDays = props.days.filter((day) => day.status === 'normal').length
  const summary = t('{{healthy}} of {{total}} days normal', {
    healthy: healthyDays,
    total: props.days.length,
  })

  return (
    <div className={cn('flex items-center gap-2', props.className)}>
      <div
        role='group'
        aria-label={summary}
        className={cn('flex min-w-0 flex-1 items-center gap-0.5')}
      >
        {props.days.map((day) => (
          <span
            key={day.day}
            title={dayLabel(t, day)}
            aria-hidden='true'
            className={cn(
              'flex-1 rounded-[2px]',
              size === 'md' ? 'h-6' : 'h-4',
              CHANNEL_HEALTH_STATUS_META[day.status].blockClassName
            )}
          />
        ))}
      </div>
      <span className='text-muted-foreground shrink-0 text-xs tabular-nums'>
        {summary}
      </span>
    </div>
  )
}

/** Legend that maps every grade to its band, so blocks stay readable. */
export function StatusHistoryLegend() {
  const { t } = useTranslation()

  return (
    <ul className='flex flex-wrap items-center gap-x-4 gap-y-1.5'>
      {CHANNEL_HEALTH_STATUS_ORDER.map((status: ChannelHealthStatus) => {
        const meta = CHANNEL_HEALTH_STATUS_META[status]
        return (
          <li key={status} className='flex items-center gap-1.5'>
            <span
              aria-hidden='true'
              className={cn('size-2.5 rounded-[2px]', meta.blockClassName)}
            />
            <span className='text-xs font-medium'>{t(meta.labelKey)}</span>
            {meta.band !== '—' && (
              <span className='text-muted-foreground text-xs tabular-nums'>
                {meta.band}
              </span>
            )}
          </li>
        )
      })}
    </ul>
  )
}
