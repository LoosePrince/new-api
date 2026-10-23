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

import type {
  HomePageContentResponse,
  PublicChatResponse,
  PublicRuntimeStatsResponse,
} from './types'

// ============================================================================
// Home Page APIs
// ============================================================================

/**
 * Get custom home page content
 * Returns Markdown/HTML content or iframe URL
 */
export async function getHomePageContent(): Promise<HomePageContentResponse> {
  // See getNotice in @/lib/api: the global `Cache-Control: no-store` is dropped
  // so the browser can hold an ETag and revalidate, letting the server answer
  // 304. Server-side `no-cache` keeps admin edits immediate.
  const res = await api.get('/api/home_page_content', {
    headers: { 'Cache-Control': null },
  })
  return res.data
}

export async function getPublicRuntimeStats(): Promise<PublicRuntimeStatsResponse> {
  const res = await api.get('/api/public/stats')
  return res.data
}

export async function sendPublicChatMessage(
  endpoint: string,
  group: string,
  routingMode: 'group' | 'model',
  model: string,
  messages: Array<{ role: 'user' | 'assistant'; content: string }>
): Promise<PublicChatResponse> {
  const normalizedEndpoint = endpoint.trim().replace(/\/+$/, '')
  const path = `${normalizedEndpoint || ''}/v1/chat/completions`
  const headers =
    routingMode === 'group'
      ? { 'X-Group': group }
      : { 'X-Group': group, 'X-Public-Chat-Model': model }
  const res = await api.post(
    path,
    { messages, stream: false },
    {
      headers,
      skipAuthorization: true,
      skipErrorHandler: true,
    }
  )
  return res.data
}
