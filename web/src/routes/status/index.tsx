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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { PublicStatus } from '@/features/public-status'
import { statusQueryOptions, type StatusData } from '@/lib/status-query'

export const Route = createFileRoute('/status/')({
  beforeLoad: async ({ context }) => {
    // The page only exists while the operator publishes it. Both conditions are
    // needed: the feature switch turns the whole section on, and the home mode
    // decides whether visitors may open the dedicated page at all. Anything else
    // stays a plain redirect to the home page rather than an empty shell.
    let status: StatusData | null = null
    try {
      status = await context.queryClient.fetchQuery(statusQueryOptions)
    } catch {
      throw redirect({ to: '/' })
    }
    if (
      status?.public_channel_status_enabled !== true ||
      status.public_channel_status_page_enabled !== true
    ) {
      throw redirect({ to: '/' })
    }
  },
  component: PublicStatus,
})
