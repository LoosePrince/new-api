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
import { afterEach, describe, expect, it, vi } from 'vitest'

import { sendPublicChatMessage } from '../api'

describe('public chat streaming API', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('parses split SSE events and sends an unauthenticated streaming request', async () => {
    const chunks = [
      'data: {"model":"routed-model","choices":[{"delta":{"content":"Hello"}}]}\n\n',
      'data: {"choices":[{"delta":{"content":" world"}}]}\n\n',
      'data: [DONE]\n\n',
    ]
    const encoded = new TextEncoder().encode(chunks.join(''))
    const splitAt = chunks[0].length - 4
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        new ReadableStream({
          start(controller) {
            controller.enqueue(encoded.slice(0, splitAt))
            controller.enqueue(encoded.slice(splitAt))
            controller.close()
          },
        }),
        { status: 200 }
      )
    )
    vi.stubGlobal('fetch', fetchMock)
    const onChunk = vi.fn()

    await sendPublicChatMessage(
      '/',
      'public',
      'model',
      'configured-model',
      [{ role: 'user', content: 'Hi' }],
      onChunk
    )

    expect(fetchMock).toHaveBeenCalledWith(
      '/v1/chat/completions',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        headers: expect.objectContaining({
          Accept: 'text/event-stream',
          'X-Group': 'public',
          'X-Public-Chat-Model': 'configured-model',
        }),
        body: JSON.stringify({
          messages: [{ role: 'user', content: 'Hi' }],
          stream: true,
        }),
      })
    )
    expect(fetchMock.mock.calls[0][1]?.headers).not.toHaveProperty(
      'Authorization'
    )
    expect(onChunk).toHaveBeenNthCalledWith(1, {
      content: 'Hello',
      model: 'routed-model',
    })
    expect(onChunk).toHaveBeenNthCalledWith(2, {
      content: ' world',
      model: undefined,
    })
  })

  it('throws the server message when the streaming request fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({ error: { message: 'Route unavailable' } }),
          {
            status: 503,
            headers: { 'Content-Type': 'application/json' },
          }
        )
      )
    )

    await expect(
      sendPublicChatMessage('/', 'public', 'group', '', [], vi.fn())
    ).rejects.toThrow('Route unavailable')
  })
})
