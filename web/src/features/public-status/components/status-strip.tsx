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
  healthyDayShare,
  PUBLIC_STATUS_META,
  PUBLIC_STATUS_ORDER,
} from '../lib/status'
import type { PublicStatusDay, PublicStatusHour } from '../types'

type StatusStripProps = {
  /** One cell per completed day of the window. */
  history: PublicStatusDay[]
  /** One cell per elapsed hour of the current day. */
  today: PublicStatusHour[]
  size?: 'sm' | 'md'
  className?: string
}

type Translate = (key: string, options?: Record<string, unknown>) => string

function dayTitle(t: Translate, day: PublicStatusDay): string {
  const date = new Date(day.day * 1000).toISOString().slice(0, 10)
  if (day.availability == null) {
    return t('{{date}}: no requests', { date })
  }
  return t('{{date}}: {{status}}, {{availability}} availability', {
    date,
    status: t(PUBLIC_STATUS_META[day.status].labelKey),
    availability: formatPublicAvailability(day.availability),
  })
}

function hourTitle(t: Translate, hour: PublicStatusHour): string {
  const start = new Date(hour.hour * 1000)
  const end = new Date((hour.hour + 3600) * 1000)
  const startLabel = start.toISOString().slice(11, 16)
  const endLabel = end.toISOString().slice(11, 16)
  if (hour.availability == null) {
    return t('{{start}} – {{end}}: no requests', {
      start: startLabel,
      end: endLabel,
    })
  }
  return t('{{start}} – {{end}}: {{status}}, {{availability}} availability', {
    start: startLabel,
    end: endLabel,
    status: t(PUBLIC_STATUS_META[hour.status].labelKey),
    availability: formatPublicAvailability(hour.availability),
  })
}

/**
 * Availability strip of the monitored window.
 *
 * The strip is split into two halves: the first carries one cell per completed
 * day (29 of the 30-day window), the second carries one cell per elapsed hour of
 * today, so the last few hours stay readable instead of collapsing into a single
 * daily cell. The daily cells are therefore wider than the hourly ones, and the
 * two groups keep their own cell count, gap and rounding so the split is visible
 * without relying on colour.
 */
export function StatusStrip(props: StatusStripProps) {
  const { t } = useTranslation()
  const size = props.size ?? 'md'
  const days = props.history ?? []
  const hours = props.today ?? []
  const share = healthyDayShare(days)
  const cellHeight = size === 'sm' ? 'h-2' : 'h-6 sm:h-7'

  return (
    <div
      role='group'
      aria-label={t(
        '{{healthy}} of {{total}} days normal, {{hours}} hours counted today',
        { healthy: share.healthy, total: share.total, hours: hours.length }
      )}
      className={cn('flex min-w-0 items-stretch gap-1', props.className)}
    >
      {/* Completed days: 2/3 of the strip, one wider cell per day. */}
      <div className='flex min-w-0 flex-[2] items-center gap-0.5'>
        {days.map((day) => (
          <span
            key={day.day}
            title={dayTitle(t, day)}
            className={cn(
              'min-w-0 flex-1 rounded-[2px]',
              cellHeight,
              PUBLIC_STATUS_META[day.status].blockClassName
            )}
          />
        ))}
      </div>

      {/* Today: 1/3 of the strip, one narrower cell per elapsed hour. */}
      <div className='flex min-w-0 flex-1 items-center gap-px'>
        {hours.map((hour) => (
          <span
            key={hour.hour}
            title={hourTitle(t, hour)}
            className={cn(
              'min-w-0 flex-1 rounded-[1px]',
              cellHeight,
              PUBLIC_STATUS_META[hour.status].blockClassName
            )}
          />
        ))}
      </div>
    </div>
  )
}

/**
 * Grade legend of the strip, ordered from the healthiest grade to the worst.
 */
export function StatusStripLegend() {
  const { t } = useTranslation()

  return (
    <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs'>
      {PUBLIC_STATUS_ORDER.map((status) => {
        const meta = PUBLIC_STATUS_META[status]
        return (
          <span key={status} className='inline-flex items-center gap-1.5'>
            <span
              className={cn('size-2 rounded-full', meta.blockClassName)}
              aria-hidden='true'
            />
            {t(meta.labelKey)}
            <span className='text-muted-foreground/70 tabular-nums'>
              {meta.band}
            </span>
          </span>
        )
      })}
    </div>
  )
}
