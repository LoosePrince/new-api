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
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import type { ChannelStatusChannel, ChannelStatusDay } from '../../types'

const api = vi.hoisted(() => ({
  updateChannelModelStatus: vi.fn(),
  getChannelStatus: vi.fn(),
}))

vi.mock('../../api', async () => {
  const actual = await vi.importActual<typeof import('../../api')>('../../api')
  return { ...actual, ...api }
})

// eslint-disable-next-line import/first
import { ModelStatusDialog } from '../model-status-dialog'

const i18n = createInstance()
let client: QueryClient

function makeDay(overrides: Partial<ChannelStatusDay> = {}): ChannelStatusDay {
  return {
    day: 1_700_000_000,
    status: 'normal',
    availability: 100,
    attempts: 10,
    average_latency_ms: 120,
    ...overrides,
  }
}

function makeChannel(): ChannelStatusChannel {
  return {
    id: 5,
    name: 'primary',
    enabled: true,
    status: 'normal',
    latency_ms: 120,
    availability_7d: 100,
    availability_30d: 99,
    attempts_7d: 20,
    last_activity: 1_700_000_000,
    history: [makeDay()],
    models: [
      {
        name: 'gpt-live',
        enabled: true,
        historical: true,
        status: 'normal',
        latency_ms: 120,
        availability_7d: 100,
        availability_30d: 100,
        attempts_7d: 12,
        attempts_30d: 30,
        last_activity: 1_700_000_000,
        history: [makeDay()],
      },
      {
        name: 'gpt-removed',
        enabled: false,
        historical: true,
        status: 'no_traffic',
        latency_ms: 0,
        availability_7d: null,
        availability_30d: 95,
        attempts_7d: 0,
        attempts_30d: 4,
        last_activity: 1_690_000_000,
        history: [makeDay({ status: 'no_traffic', availability: null })],
      },
      {
        name: 'gpt-forgotten',
        enabled: false,
        historical: false,
        status: 'no_traffic',
        latency_ms: 0,
        availability_7d: null,
        availability_30d: null,
        attempts_7d: 0,
        attempts_30d: 0,
        last_activity: 0,
        history: [makeDay({ status: 'no_traffic', availability: null })],
      },
    ],
  }
}

beforeEach(async () => {
  await i18n.init({
    lng: 'en',
    fallbackLng: 'en',
    resources: { en },
    interpolation: { escapeValue: false },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  api.updateChannelModelStatus.mockReset()
  api.updateChannelModelStatus.mockResolvedValue({ success: true, data: {} })
})

afterEach(() => {
  cleanup()
  client.clear()
  vi.restoreAllMocks()
})

function renderDialog() {
  render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <ModelStatusDialog
          open
          onOpenChange={() => undefined}
          channel={makeChannel()}
        />
      </QueryClientProvider>
    </I18nextProvider>
  )
}

test('removing a configured model asks for confirmation before changing the channel', async () => {
  const user = userEvent.setup()
  renderDialog()

  await user.click(screen.getByRole('switch', { name: 'Enable gpt-live' }))

  const confirm = await screen.findByRole('button', { name: 'Remove' })
  expect(api.updateChannelModelStatus).not.toHaveBeenCalled()

  await user.click(confirm)

  await waitFor(() =>
    expect(api.updateChannelModelStatus).toHaveBeenCalledWith(
      5,
      'gpt-live',
      false
    )
  )
})

test('restoring a removed model with history is applied without a confirmation dialog', async () => {
  const user = userEvent.setup()
  renderDialog()

  await user.click(screen.getByRole('switch', { name: 'Enable gpt-removed' }))

  await waitFor(() =>
    expect(api.updateChannelModelStatus).toHaveBeenCalledWith(
      5,
      'gpt-removed',
      true
    )
  )
  expect(screen.queryByRole('button', { name: 'Remove' })).toBeNull()
})

test('a model without request history cannot be restored', () => {
  renderDialog()

  const toggle = screen.getByRole('switch', { name: 'Enable gpt-forgotten' })
  expect(toggle).toHaveAttribute('aria-disabled', 'true')
  expect(screen.getByText('No request history, cannot restore')).toBeVisible()
})
