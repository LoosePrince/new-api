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
import { formatNumber } from '@/lib/format'

import type { PublicServiceStatus } from '../types'

export interface PublicStatusMeta {
  /** i18n key of the grade label. */
  labelKey: string
  /** Success-rate band the grade stands for. */
  band: string
  /** Filled block colour of the daily strip. */
  blockClassName: string
  /** Text colour used by the grade label. */
  textClassName: string
  /** Soft background used by the grade pill. */
  pillClassName: string
}

/**
 * Grades ordered from the healthiest to the worst. The thresholds mirror the
 * backend classification, so the legend, the strip and the model rows can never
 * disagree about what a grade means.
 */
export const PUBLIC_STATUS_ORDER: PublicServiceStatus[] = [
  'normal',
  'latency',
  'degraded',
  'abnormal',
  'unavailable',
  'no_traffic',
]

export const PUBLIC_STATUS_META: Record<PublicServiceStatus, PublicStatusMeta> =
  {
    normal: {
      labelKey: 'Operating normally',
      band: '≥ 95%',
      blockClassName: 'bg-emerald-500',
      textClassName: 'text-emerald-600 dark:text-emerald-400',
      pillClassName: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    },
    latency: {
      labelKey: 'Elevated latency',
      band: '85% – 95%',
      blockClassName: 'bg-amber-400',
      textClassName: 'text-amber-600 dark:text-amber-400',
      pillClassName: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    },
    degraded: {
      labelKey: 'Partially degraded',
      band: '60% – 85%',
      blockClassName: 'bg-orange-500',
      textClassName: 'text-orange-600 dark:text-orange-400',
      pillClassName: 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
    },
    abnormal: {
      labelKey: 'Service abnormal',
      band: '30% – 60%',
      blockClassName: 'bg-red-500',
      textClassName: 'text-red-600 dark:text-red-400',
      pillClassName: 'bg-red-500/10 text-red-600 dark:text-red-400',
    },
    unavailable: {
      labelKey: 'Service unavailable',
      band: '< 30%',
      blockClassName: 'bg-red-800 dark:bg-red-900',
      textClassName: 'text-red-700 dark:text-red-300',
      pillClassName: 'bg-red-700/10 text-red-700 dark:text-red-300',
    },
    no_traffic: {
      labelKey: 'No traffic',
      band: '—',
      blockClassName: 'bg-muted-foreground/25',
      textClassName: 'text-muted-foreground',
      pillClassName: 'bg-muted text-muted-foreground',
    },
  }

/** Soft badge tone of a grade, used by the summary icon. */
export function summaryTone(
  status: PublicServiceStatus | undefined
): 'success' | 'warning' | 'destructive' | 'neutral' {
  switch (status) {
    case 'normal':
      return 'success'
    case 'latency':
    case 'degraded':
      return 'warning'
    case 'abnormal':
    case 'unavailable':
      return 'destructive'
    default:
      return 'neutral'
  }
}

/** Format an availability percentage, or a dash when the window had no traffic. */ export function formatPublicAvailability(
  value: number | null | undefined,
  locale?: Intl.LocalesArgument
): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return `${formatNumber(value, locale)}%`
}

/** Format an average latency in milliseconds, or a dash when unmeasured. */
export function formatPublicLatency(
  value: number | null | undefined,
  locale?: Intl.LocalesArgument
): string {
  if (value == null || !Number.isFinite(value) || value <= 0) return '—'
  return `${formatNumber(value, locale)} ms`
}

/**
 * Share of healthy days in the strip, used as the strip's text alternative so
 * the status is never conveyed by colour alone.
 */
export function healthyDayShare(days: Array<{ status: PublicServiceStatus }>): {
  healthy: number
  total: number
} {
  return {
    healthy: days.filter((day) => day.status === 'normal').length,
    total: days.length,
  }
}
