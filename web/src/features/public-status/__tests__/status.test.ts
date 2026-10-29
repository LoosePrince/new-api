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
import { describe, expect, test } from 'vitest'

import {
  formatPublicAvailability,
  formatPublicLatency,
  healthyDayShare,
  PUBLIC_STATUS_META,
  PUBLIC_STATUS_ORDER,
  summaryTone,
} from '../lib/status'
import type { PublicServiceStatus } from '../types'

describe('public status grades', () => {
  test('orders every grade from the healthiest to the worst', () => {
    expect(PUBLIC_STATUS_ORDER).toEqual([
      'normal',
      'latency',
      'degraded',
      'abnormal',
      'unavailable',
      'no_traffic',
    ])
    for (const status of PUBLIC_STATUS_ORDER) {
      expect(PUBLIC_STATUS_META[status].labelKey.length).toBeGreaterThan(0)
      expect(PUBLIC_STATUS_META[status].band.length).toBeGreaterThan(0)
      expect(PUBLIC_STATUS_META[status].blockClassName.length).toBeGreaterThan(
        0
      )
    }
  })

  test('maps every grade to a summary badge tone', () => {
    const tones: Record<PublicServiceStatus, string> = {
      normal: 'success',
      latency: 'warning',
      degraded: 'warning',
      abnormal: 'destructive',
      unavailable: 'destructive',
      no_traffic: 'neutral',
    }
    for (const [status, tone] of Object.entries(tones)) {
      expect(summaryTone(status as PublicServiceStatus)).toBe(tone)
    }
    expect(summaryTone(undefined)).toBe('neutral')
  })
})

describe('public status formatting', () => {
  test('keeps a dash for figures the window never measured', () => {
    expect(formatPublicAvailability(null)).toBe('—')
    expect(formatPublicAvailability(undefined)).toBe('—')
    expect(formatPublicAvailability(Number.NaN)).toBe('—')
    expect(formatPublicLatency(0)).toBe('—')
    expect(formatPublicLatency(null)).toBe('—')
  })

  test('formats measured figures', () => {
    expect(formatPublicAvailability(99.5, 'en-US')).toBe('99.5%')
    expect(formatPublicLatency(1234, 'en-US')).toBe('1,234 ms')
  })

  test('counts healthy days of the strip', () => {
    expect(
      healthyDayShare([
        { status: 'normal' },
        { status: 'normal' },
        { status: 'degraded' },
        { status: 'no_traffic' },
      ])
    ).toEqual({ healthy: 2, total: 4 })
    expect(healthyDayShare([])).toEqual({ healthy: 0, total: 0 })
  })
})
