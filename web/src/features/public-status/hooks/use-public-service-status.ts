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
import { useQuery } from '@tanstack/react-query'

import { getPublicChannelStatus, PUBLIC_STATUS_QUERY_KEY } from '../api'

const REFRESH_INTERVAL_MS = 60_000

/**
 * Whole-site status summary of the public runtime data area.
 *
 * The key is shared with the status page so one visit reuses the cached
 * response; the home page only reads the summary fields, plus the published
 * layout flags that decide between the compact bar and the detailed card.
 */
export function usePublicServiceStatus() {
  const { data, isPending } = useQuery({
    queryKey: [...PUBLIC_STATUS_QUERY_KEY, 'summary'],
    queryFn: () => getPublicChannelStatus(true),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: REFRESH_INTERVAL_MS,
    meta: { errorToast: false },
  })

  return {
    enabled: data?.enabled === true,
    summary: data?.summary,
    detailed: data?.home_detailed === true,
    pageEnabled: data?.page_enabled === true,
    channelsVisible: data?.channels_visible !== false,
    modelsVisible: data?.models_enabled !== false,
    isPending,
  }
}
