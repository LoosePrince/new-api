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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { Toaster } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import type { UsageLog } from '../../data/schema'
import { DetailsDialog } from '../dialogs/details-dialog'

const api = vi.hoisted(() => ({
  getRequestCaptures: vi.fn(),
  exportRequestCaptures: vi.fn(),
  addLogIPToBlacklist: vi.fn(),
}))

vi.mock('../../api', () => api)

const CLIENT_IP = '203.0.113.9'

function makeLog(overrides: Partial<UsageLog> = {}): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 1,
    type: 2,
    content: '',
    username: 'user',
    token_name: 'token',
    model_name: 'gpt-4o-mini',
    quota: 5000,
    prompt_tokens: 10,
    completion_tokens: 20,
    use_time: 1,
    is_stream: false,
    channel: 1,
    channel_name: '',
    token_id: 1,
    group: 'default',
    ip: CLIENT_IP,
    other: '{}',
    request_id: 'req-1',
    upstream_request_id: '',
    ...overrides,
  }
}

const i18n = createInstance()
let client: QueryClient

beforeEach(async () => {
  await i18n.init({
    lng: 'en',
    resources: { en },
    interpolation: { escapeValue: false },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {}, { updatedAt: Date.now() + 60_000 })
  api.getRequestCaptures.mockResolvedValue({
    success: true,
    data: {
      items: [
        {
          id: 1,
          request_id: 'req-1',
          attempt: 1,
          channel_id: 1,
          request_body: '{"model":"gpt-4o-mini","stream":true}',
          response_body:
            '{"choices":[{"message":{"role":"assistant","content":"# Answer\\n\\n**Rendered**"}}]}',
        },
      ],
      total: 1,
    },
  })
  api.exportRequestCaptures.mockResolvedValue(
    new Blob(['readable records'], { type: 'text/plain' })
  )
  api.addLogIPToBlacklist.mockResolvedValue({ success: true })
})

afterEach(() => {
  cleanup()
  client.clear()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

function renderDialog(options: {
  isRoot: boolean
  isAdmin?: boolean
  log?: Partial<UsageLog>
}) {
  render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <DetailsDialog
          log={makeLog(options.log)}
          isAdmin={options.isAdmin ?? true}
          isRoot={options.isRoot}
          open
          onOpenChange={() => undefined}
        />
        <Toaster />
      </QueryClientProvider>
    </I18nextProvider>
  )
}

describe('details dialog IP blacklist action', () => {
  it('shows the recorded client IP and adds it to the blacklist after confirmation', async () => {
    const user = userEvent.setup()
    renderDialog({ isRoot: true })

    expect(await screen.findByText(CLIENT_IP)).toBeVisible()

    await user.click(
      await screen.findByRole('button', { name: 'Add to IP blacklist' })
    )
    expect(api.addLogIPToBlacklist).not.toHaveBeenCalled()

    await user.click(
      await screen.findByRole('button', { name: 'Add to blacklist' })
    )

    await waitFor(() => {
      expect(api.addLogIPToBlacklist).toHaveBeenCalledWith(CLIENT_IP, false)
    })
    expect(await screen.findByText('IP added to the blacklist')).toBeVisible()
  })

  it('asks for a second confirmation before blocking the current IP', async () => {
    api.addLogIPToBlacklist
      .mockResolvedValueOnce({
        success: false,
        code: 'current_ip_requires_confirmation',
      })
      .mockResolvedValueOnce({ success: true })
    const user = userEvent.setup()
    renderDialog({ isRoot: true })

    await user.click(
      await screen.findByRole('button', { name: 'Add to IP blacklist' })
    )
    await user.click(
      await screen.findByRole('button', { name: 'Add to blacklist' })
    )

    expect(
      await screen.findByText('This is your current IP address')
    ).toBeVisible()
    await user.click(
      await screen.findByRole('button', { name: 'Block my current IP' })
    )

    await waitFor(() => {
      expect(api.addLogIPToBlacklist).toHaveBeenLastCalledWith(CLIENT_IP, true)
    })
    expect(await screen.findByText('IP added to the blacklist')).toBeVisible()
  })

  it('shows the blacklist action for administrators who are not root', async () => {
    renderDialog({ isRoot: false })

    expect(await screen.findByText(CLIENT_IP)).toBeVisible()
    expect(
      await screen.findByRole('button', { name: 'Add to IP blacklist' })
    ).toBeVisible()
  })

  it('hides the blacklist action from users who are not administrators', async () => {
    renderDialog({ isRoot: false, isAdmin: false })

    expect(await screen.findByText(CLIENT_IP)).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Add to IP blacklist' })
    ).toBeNull()
  })

  it('shows the client IP once for login audit logs', async () => {
    renderDialog({
      isRoot: true,
      log: {
        type: 7,
        other: JSON.stringify({
          login_method: 'password',
          user_agent: 'test-agent',
        }),
      },
    })

    expect(await screen.findByText(CLIENT_IP)).toBeVisible()
    expect(screen.getAllByText('IP Address')).toHaveLength(1)
  })
})

describe('details dialog request record export', () => {
  beforeEach(() => {
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: vi.fn(() => 'blob:request-records'),
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: vi.fn(),
    })
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(
      () => undefined
    )
  })

  it('renders request and response content as Markdown by default', async () => {
    const user = userEvent.setup()
    renderDialog({ isRoot: true })

    expect(await screen.findByRole('heading', { name: 'Answer' })).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'Raw' }))

    expect(
      screen.getByText('{"model":"gpt-4o-mini","stream":true}')
    ).toBeVisible()
  })

  it('exports readable request records as a text download', async () => {
    const user = userEvent.setup()
    renderDialog({ isRoot: true })

    await user.click(
      await screen.findByRole('button', { name: 'Export request records' })
    )
    await user.click(
      await screen.findByRole('menuitem', { name: 'Readable records (TXT)' })
    )

    await waitFor(() => {
      expect(api.exportRequestCaptures).toHaveBeenCalledWith(
        'req-1',
        'rendered'
      )
    })
    expect(URL.createObjectURL).toHaveBeenCalledWith(expect.any(Blob))
    expect(await screen.findByText('Request records exported')).toBeVisible()
  })

  it('disables the export trigger while a download is in flight', async () => {
    let resolveExport: (blob: Blob) => void = () => undefined
    api.exportRequestCaptures.mockReturnValue(
      new Promise<Blob>((resolve) => {
        resolveExport = resolve
      })
    )
    const user = userEvent.setup()
    renderDialog({ isRoot: true })

    await user.click(
      await screen.findByRole('button', { name: 'Export request records' })
    )
    await user.click(
      await screen.findByRole('menuitem', { name: 'Raw records (NDJSON)' })
    )

    const exporting = await screen.findByRole('button', {
      name: 'Exporting...',
    })
    expect(exporting).toBeDisabled()

    resolveExport(new Blob(['records'], { type: 'application/x-ndjson' }))
    expect(
      await screen.findByRole('button', { name: 'Export request records' })
    ).toBeEnabled()
  })
})
