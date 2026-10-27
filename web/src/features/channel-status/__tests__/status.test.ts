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
import { expect, test } from 'vitest'

import {
  canRestoreModel,
  CHANNEL_HEALTH_STATUS_META,
  CHANNEL_HEALTH_STATUS_ORDER,
  formatAvailability,
} from '../lib/status'

test('every grade has a label and a block color, and the legend is ordered by health', () => {
  expect(CHANNEL_HEALTH_STATUS_ORDER).toEqual([
    'normal',
    'latency',
    'degraded',
    'abnormal',
    'unavailable',
    'no_traffic',
  ])
  for (const status of CHANNEL_HEALTH_STATUS_ORDER) {
    const meta = CHANNEL_HEALTH_STATUS_META[status]
    expect(meta.labelKey).not.toBe('')
    expect(meta.blockClassName).not.toBe('')
  }
})

test('availability renders a percent value and a dash when the window has no traffic', () => {
  expect(formatAvailability(99.123, 'en')).toBe('99.12%')
  expect(formatAvailability(0, 'en')).toBe('0%')
  expect(formatAvailability(null, 'en')).toBe('—')
  expect(formatAvailability(undefined, 'en')).toBe('—')
})

test('a removed model is only restorable when it has request history', () => {
  expect(canRestoreModel({ enabled: true, historical: true })).toBe(false)
  expect(canRestoreModel({ enabled: false, historical: true })).toBe(true)
  expect(canRestoreModel({ enabled: false, historical: false })).toBe(false)
})
