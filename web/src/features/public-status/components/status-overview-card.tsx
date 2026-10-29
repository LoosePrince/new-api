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
import { CircleCheck, ShieldCheck, Timer } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge } from '@/components/ui/icon-badge'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  formatPublicAvailability,
  formatPublicLatency,
  PUBLIC_STATUS_META,
  summaryTone,
} from '../lib/status'
import type { PublicStatusSummary } from '../types'
import { StatusStrip, StatusStripLegend } from './status-strip'

interface PublicStatusOverviewCardProps {
  summary?: PublicStatusSummary
  locale?: Intl.LocalesArgument
  /** Marks the first status request, so counts show placeholder text. */
  isPending?: boolean
  /** Channel names are published; otherwise the channel count is hidden. */
  channelsVisible?: boolean
  /** The per-model section is published; otherwise the model count is hidden. */
  modelsVisible?: boolean
  className?: string
}

/**
 * Whole-site status overview: the headline grade, request counts, the
 * availability strip and the 7-day / 30-day / latency figures.
 *
 * Both the status page and the home page render this card, because "detailed"
 * on the home page means exactly what the status page shows. Keeping one
 * implementation guarantees the two stay identical.
 */
export function PublicStatusOverviewCard(props: PublicStatusOverviewCardProps) {
  const { t } = useTranslation()
  const summary = props.summary
  const meta = PUBLIC_STATUS_META[summary?.status ?? 'no_traffic']
  const stats = [
    {
      icon: CircleCheck,
      label: t('7-day availability'),
      value: formatPublicAvailability(summary?.availability_7d, props.locale),
    },
    {
      icon: ShieldCheck,
      label: t('30-day availability'),
      value: formatPublicAvailability(summary?.availability_30d, props.locale),
    },
    {
      icon: Timer,
      label: t('Average latency'),
      value: formatPublicLatency(summary?.latency_ms, props.locale),
    },
  ]

  return (
    <section
      className={cn('bg-card rounded-xl border p-4 sm:p-6', props.className)}
    >
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex items-center gap-3'>
          <IconBadge tone={summaryTone(summary?.status)} size='lg'>
            <ShieldCheck />
          </IconBadge>
          <div>
            <div className='text-muted-foreground text-xs font-medium tracking-wide uppercase'>
              {t('Overall service status')}
            </div>
            <div className={cn('text-lg font-semibold', meta.textClassName)}>
              {props.isPending
                ? t('Checking service status...')
                : t(meta.labelKey)}
            </div>
          </div>
        </div>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
          {/* Hidden sections make their counts meaningless, so they are dropped
              rather than shown as zero. */}
          {props.channelsVisible !== false && (
            <span>
              {t('Channels')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatNumber(summary?.channel_count ?? 0, props.locale)}
              </span>
            </span>
          )}
          {props.modelsVisible !== false && (
            <span>
              {t('Models')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatNumber(summary?.model_count ?? 0, props.locale)}
              </span>
            </span>
          )}
          <span>
            {t('Attempts')}:{' '}
            <span className='text-foreground font-medium tabular-nums'>
              {formatNumber(summary?.attempts ?? 0, props.locale)}
            </span>
          </span>
        </div>
      </div>

      <StatusStrip
        history={summary?.history ?? []}
        today={summary?.today ?? []}
        className='mt-4'
      />

      <div className='mt-5 grid gap-3 sm:grid-cols-3'>
        {stats.map((item) => {
          const Icon = item.icon
          return (
            <div
              key={item.label}
              className='bg-muted/20 flex items-center gap-2.5 rounded-lg border px-3 py-2.5'
            >
              <Icon
                className='text-muted-foreground size-4 shrink-0'
                aria-hidden='true'
              />
              <div className='min-w-0'>
                <div className='text-muted-foreground text-xs'>
                  {item.label}
                </div>
                <div className='text-base font-semibold tabular-nums'>
                  {item.value}
                </div>
              </div>
            </div>
          )
        })}
      </div>

      <div className='border-t pt-4'>
        <StatusStripLegend />
      </div>
    </section>
  )
}
