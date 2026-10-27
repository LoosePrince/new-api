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
export type ChannelHealthStatus =
  | 'normal'
  | 'latency'
  | 'degraded'
  | 'abnormal'
  | 'unavailable'
  | 'no_traffic'

export interface ChannelStatusDay {
  day: number
  status: ChannelHealthStatus
  availability: number | null
  attempts: number
  average_latency_ms: number
}

export interface ChannelStatusModel {
  name: string
  enabled: boolean
  historical: boolean
  status: ChannelHealthStatus
  latency_ms: number
  availability_7d: number | null
  availability_30d: number | null
  attempts_7d: number
  attempts_30d: number
  last_activity: number
  history: ChannelStatusDay[]
}

export interface ChannelStatusChannel {
  id: number
  name: string
  enabled: boolean
  status: ChannelHealthStatus
  latency_ms: number
  availability_7d: number | null
  availability_30d: number | null
  attempts_7d: number
  last_activity: number
  history: ChannelStatusDay[]
  models: ChannelStatusModel[]
}

export interface ChannelStatusSummary {
  normal_channels: number
  non_normal_channels: number
  unavailable_channels: number
  no_traffic_channels: number
  channel_count: number
  model_count: number
  monitored_models: number
}

export interface ChannelStatusResponse {
  generated_at: number
  window_start: number
  summary: ChannelStatusSummary
  channels: ChannelStatusChannel[]
}

export interface UpdateChannelModelStatusResponse {
  success: boolean
  message?: string
  data?: {
    channel_id: number
    model: string
    enabled: boolean
  }
}
