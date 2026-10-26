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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { exportRequestCaptures } from '../api'

beforeEach(() => {
  useAuthStore.setState((state) => ({
    auth: {
      ...state.auth,
      user: { id: 1, username: 'root', role: 100 },
      accessToken: 'test-token',
      accessExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    },
  }))
})

afterEach(() => {
  vi.unstubAllGlobals()
  useAuthStore.getState().auth.reset()
})

describe('exportRequestCaptures', () => {
  it('downloads the export through an authenticated fetch', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('rendered records', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const blob = await exportRequestCaptures('req-1', 'rendered')

    expect(await blob.text()).toBe('rendered records')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      '/api/request-captures/export?request_id=req-1&format=rendered'
    )
    expect(init.credentials).toBe('include')
    expect(init.headers).toMatchObject({
      Authorization: 'Bearer test-token',
    })
  })

  it('keeps the ndjson format in the export query', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{"success":true}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await exportRequestCaptures('req-2', 'ndjson')

    const [url] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      '/api/request-captures/export?request_id=req-2&format=ndjson'
    )
  })

  it('surfaces the server error instead of downloading an error document', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            success: false,
            message: 'Authorization required',
          }),
          { status: 403 }
        )
      )
    )

    await expect(exportRequestCaptures('req-1', 'rendered')).rejects.toThrow(
      'Authorization required'
    )
  })

  it('rejects when the rejection body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response('<html>Forbidden</html>', { status: 403 })
        )
    )

    await expect(
      exportRequestCaptures('req-1', 'rendered')
    ).rejects.toBeInstanceOf(Error)
  })
})
