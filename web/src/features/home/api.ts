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
  PublicChatStreamChunk,
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
  messages: Array<{ role: 'user' | 'assistant'; content: string }>,
  onChunk: (chunk: PublicChatStreamChunk) => void
): Promise<void> {
  const normalizedEndpoint = endpoint.trim().replace(/\/+$/, '')
  const path = `${normalizedEndpoint || ''}/v1/chat/completions`
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Content-Type': 'application/json',
    'X-Group': group,
  }
  if (routingMode === 'model') {
    headers['X-Public-Chat-Model'] = model
  }
  const response = await fetch(path, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: JSON.stringify({ messages, stream: true }),
  })

  if (!response.ok) {
    let message = `Public chat request failed (${response.status})`
    try {
      const payload = (await response.json()) as {
        error?: { message?: unknown }
        message?: unknown
      }
      const serverMessage = payload.error?.message ?? payload.message
      if (typeof serverMessage === 'string' && serverMessage) {
        message = serverMessage
      }
    } catch {
      // Keep the status-based error when the response is not JSON.
    }
    throw new Error(message)
  }

  if (!response.body) {
    throw new Error('The public chat did not return a readable stream.')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let readerDone = false
  let doneReceived = false

  const processEvent = (event: string): boolean => {
    const data = event
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
      .join('\n')
      .trim()

    if (!data) return false
    if (data === '[DONE]') return true

    const payload = JSON.parse(data) as {
      model?: unknown
      choices?: Array<{
        delta?: { content?: unknown }
      }>
      error?: { message?: unknown }
    }
    const errorMessage = payload.error?.message
    if (typeof errorMessage === 'string' && errorMessage) {
      throw new Error(errorMessage)
    }

    const content = payload.choices?.[0]?.delta?.content
    onChunk({
      content: typeof content === 'string' ? content : undefined,
      model: typeof payload.model === 'string' ? payload.model : undefined,
    })
    return false
  }

  while (!readerDone && !doneReceived) {
    const result = await reader.read()
    if (result.done) {
      buffer += decoder.decode()
      readerDone = true
    } else {
      buffer += decoder.decode(result.value, { stream: true })
    }

    const events = buffer.split(/\r?\n\r?\n/)
    buffer = events.pop() || ''
    for (const event of events) {
      if (processEvent(event)) {
        doneReceived = true
        break
      }
    }
    if (doneReceived && !result.done) await reader.cancel()
  }

  if (!doneReceived && buffer.trim()) processEvent(buffer)
}
