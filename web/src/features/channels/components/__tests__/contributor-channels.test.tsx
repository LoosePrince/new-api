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
import { cleanup, render, screen } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import type { Channel } from '../../types'
import { ContributorChannels } from '../contributor-channels'

const api = vi.hoisted(() => ({
  getContributorChannels: vi.fn(),
}))

vi.mock('../../api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api')>()
  return { ...actual, ...api }
})

const i18n = createInstance()
let client: QueryClient

const channel: Channel = {
  id: 11,
  type: 1,
  key: '',
  openai_organization: null,
  test_model: null,
  status: 1,
  name: 'alice-channel',
  weight: null,
  created_time: 0,
  test_time: 0,
  response_time: 0,
  base_url: null,
  other: '',
  balance: 0,
  balance_updated_time: 0,
  models: '',
  group: 'default',
  used_quota: 0,
  model_mapping: null,
  status_code_mapping: null,
  priority: null,
  auto_ban: null,
  other_info: '',
  tag: null,
  setting: null,
  param_override: null,
  header_override: null,
  remark: '',
  max_input_tokens: 0,
  channel_info: {
    is_multi_key: false,
    multi_key_size: 0,
    multi_key_polling_index: 0,
    multi_key_mode: 'random',
  },
  settings: '{}',
  contributor_username: 'alice',
}

beforeEach(async () => {
  await i18n.init({
    lng: 'en',
    resources: { en },
    interpolation: { escapeValue: false },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  api.getContributorChannels.mockResolvedValue({
    success: true,
    data: { items: [channel], total: 1 },
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  vi.clearAllMocks()
})

describe('contributor channels', () => {
  it('shows which contributor owns each channel to administrators', async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={client}>
          <ContributorChannels />
        </QueryClientProvider>
      </I18nextProvider>
    )

    expect(await screen.findByText('alice-channel')).toBeVisible()
    expect(await screen.findByText('Contributor: alice')).toBeVisible()
  })
})
