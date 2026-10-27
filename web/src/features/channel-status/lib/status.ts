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
import type { StatusVariant } from '@/components/status-badge'
import { formatNumber } from '@/lib/format'

import type { ChannelHealthStatus } from '../types'

export interface ChannelHealthStatusMeta {
  /** i18n key of the status label. */
  labelKey: string
  /** Success-rate band the status stands for, shown in the legend. */
  band: string
  variant: StatusVariant
  textClassName: string
  blockClassName: string
}

/**
 * Grades ordered from the healthiest to the worst. The thresholds mirror the
 * backend `channelHealthStatus` classification so the legend never disagrees
 * with the statuses rendered next to it.
 */
export const CHANNEL_HEALTH_STATUS_ORDER: ChannelHealthStatus[] = [
  'normal',
  'latency',
  'degraded',
  'abnormal',
  'unavailable',
  'no_traffic',
]

export const CHANNEL_HEALTH_STATUS_META: Record<
  ChannelHealthStatus,
  ChannelHealthStatusMeta
> = {
  normal: {
    labelKey: 'Normal',
    band: '≥ 95%',
    variant: 'success',
    textClassName: 'text-emerald-600 dark:text-emerald-400',
    blockClassName: 'bg-emerald-500',
  },
  latency: {
    labelKey: 'Latency',
    band: '85% – 95%',
    variant: 'warning',
    textClassName: 'text-amber-600 dark:text-amber-400',
    blockClassName: 'bg-amber-400',
  },
  degraded: {
    labelKey: 'Degraded',
    band: '60% – 85%',
    variant: 'warning',
    textClassName: 'text-orange-600 dark:text-orange-400',
    blockClassName: 'bg-orange-500',
  },
  abnormal: {
    labelKey: 'Abnormal',
    band: '30% – 60%',
    variant: 'danger',
    textClassName: 'text-red-600 dark:text-red-400',
    blockClassName: 'bg-red-500',
  },
  unavailable: {
    labelKey: 'Unavailable',
    band: '< 30%',
    variant: 'danger',
    textClassName: 'text-red-700 dark:text-red-300',
    blockClassName: 'bg-red-800 dark:bg-red-900',
  },
  no_traffic: {
    labelKey: 'No traffic',
    band: '—',
    variant: 'neutral',
    textClassName: 'text-muted-foreground',
    blockClassName: 'bg-muted-foreground/25',
  },
}

/** Format an availability percentage, or a dash when the window had no traffic. */
export function formatAvailability(
  value: number | null | undefined,
  locale?: Intl.LocalesArgument
): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return `${formatNumber(value, locale)}%`
}

/** A model that is no longer configured can only be restored from its logs. */
export function canRestoreModel(model: {
  enabled: boolean
  historical: boolean
}): boolean {
  return !model.enabled && model.historical
}
