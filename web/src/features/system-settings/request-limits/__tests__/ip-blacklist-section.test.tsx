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

import { SettingsPageProvider } from '../../components/settings-page-context'
import { IPBlacklistSection } from '../ip-blacklist-section'

const api = vi.hoisted(() => ({
  updateSystemOption: vi.fn(),
}))

vi.mock('../../api', () => api)

const i18n = createInstance()
let client: QueryClient
let actionsContainer: HTMLDivElement

beforeEach(async () => {
  await i18n.init({
    lng: 'en',
    resources: { en },
    interpolation: { escapeValue: false },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  api.updateSystemOption.mockResolvedValue({ success: true })
  actionsContainer = document.createElement('div')
  document.body.appendChild(actionsContainer)
})

afterEach(() => {
  cleanup()
  actionsContainer.remove()
  client.clear()
  vi.clearAllMocks()
})

function renderSection(defaultValues: { enabled: boolean; ipList: string[] }) {
  render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <SettingsPageProvider actionsContainer={actionsContainer}>
          <IPBlacklistSection
            defaultValues={{
              'ip_blacklist_setting.enabled': defaultValues.enabled,
              'ip_blacklist_setting.ip_list': defaultValues.ipList,
            }}
          />
        </SettingsPageProvider>
      </QueryClientProvider>
      <Toaster />
    </I18nextProvider>
  )
}

describe('IP blacklist settings section', () => {
  it('saves the enabled state and line-separated entries as a JSON list', async () => {
    const user = userEvent.setup()
    renderSection({ enabled: false, ipList: ['198.51.100.4'] })

    await user.click(
      screen.getByRole('switch', { name: 'Enable IP Blacklist' })
    )
    const textarea = screen.getByRole('textbox', { name: 'Blacklisted IPs' })
    await user.clear(textarea)
    await user.type(textarea, '198.51.100.4{enter}10.0.0.0/8{enter}203.0.113.9')
    await user.click(screen.getByRole('button', { name: 'Save IP blacklist' }))

    await waitFor(() => {
      expect(api.updateSystemOption).toHaveBeenCalledTimes(2)
    })
    expect(api.updateSystemOption).toHaveBeenNthCalledWith(1, {
      key: 'ip_blacklist_setting.enabled',
      value: 'true',
    })
    expect(api.updateSystemOption).toHaveBeenNthCalledWith(2, {
      key: 'ip_blacklist_setting.ip_list',
      value: JSON.stringify(['198.51.100.4', '10.0.0.0/8', '203.0.113.9']),
    })
  })

  it('does not send an update when nothing changed', async () => {
    const user = userEvent.setup()
    renderSection({ enabled: true, ipList: ['203.0.113.9'] })

    await user.click(screen.getByRole('button', { name: 'Save IP blacklist' }))

    await waitFor(() => {
      expect(screen.getByText('No changes to save')).toBeVisible()
    })
    expect(api.updateSystemOption).not.toHaveBeenCalled()
  })
})
