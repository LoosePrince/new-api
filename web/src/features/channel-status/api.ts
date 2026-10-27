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
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import type {
  ChannelStatusResponse,
  UpdateChannelModelStatusResponse,
} from './types'

export const CHANNEL_STATUS_QUERY_KEY = ['channel-status'] as const

export async function getChannelStatus(): Promise<ChannelStatusResponse> {
  const response = await api.get<{
    success: boolean
    message?: string
    data: ChannelStatusResponse
  }>('/api/channel/status', { disableDuplicate: true })
  return requireServerSuccess(response.data).data
}

/**
 * Remove a model from the channel's model list, or restore a model that the
 * channel served before. Removal changes routing immediately.
 */
export async function updateChannelModelStatus(
  channelId: number,
  model: string,
  enable: boolean
): Promise<UpdateChannelModelStatusResponse> {
  const response = await api.patch<UpdateChannelModelStatusResponse>(
    `/api/channel/${channelId}/models/status`,
    { model, enable }
  )
  return requireServerSuccess(response.data)
}
