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
/** Availability grades derived from the request success rate (0-100). */
export type PublicServiceStatus =
  | 'normal'
  | 'latency'
  | 'degraded'
  | 'abnormal'
  | 'unavailable'
  | 'no_traffic'

export interface PublicStatusDay {
  day: number
  status: PublicServiceStatus
  availability: number | null
}

/** One elapsed hour of the current day, reported separately from the days. */
export interface PublicStatusHour {
  hour: number
  status: PublicServiceStatus
  availability: number | null
}

export interface PublicStatusChannel {
  name: string
  status: PublicServiceStatus
  latency_ms: number
  availability_7d: number | null
}

export interface PublicStatusModel {
  name: string
  status: PublicServiceStatus
  latency_ms: number
  availability_7d: number | null
  availability_30d: number | null
  attempts: number
  history: PublicStatusDay[]
  today: PublicStatusHour[]
  channels: PublicStatusChannel[]
}

export interface PublicStatusSummary {
  status: PublicServiceStatus
  latency_ms: number
  availability_7d: number | null
  availability_30d: number | null
  attempts: number
  channel_count: number
  model_count: number
  history: PublicStatusDay[]
  today: PublicStatusHour[]
}

export interface PublicChannelStatus {
  enabled: boolean
  /** The home page renders the full overview card instead of the compact bar. */
  home_detailed?: boolean
  /** The dedicated status route stays reachable. */
  page_enabled?: boolean
  /** The per-model section is published. */
  models_enabled?: boolean
  /** Channel names are published at all; when false the count is meaningless. */
  channels_visible?: boolean
  generated_at?: number
  window_start?: number
  summary?: PublicStatusSummary
  models?: PublicStatusModel[]
}

export interface PublicChannelStatusResponse {
  success: boolean
  message?: string
  data?: PublicChannelStatus
}
