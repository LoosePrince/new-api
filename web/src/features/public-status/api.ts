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

import type { PublicChannelStatus, PublicChannelStatusResponse } from './types'

export const PUBLIC_STATUS_QUERY_KEY = ['public-channel-status'] as const

/**
 * Read the anonymous site status. `summary` asks for the whole-site figures
 * only, which is all the home page bar renders.
 */
export async function getPublicChannelStatus(
  summary = false
): Promise<PublicChannelStatus> {
  const response = await api.get<PublicChannelStatusResponse>(
    '/api/public/channel-status',
    {
      params: summary ? { summary: '1' } : undefined,
      disableDuplicate: true,
    }
  )
  return response.data?.data ?? { enabled: false }
}
