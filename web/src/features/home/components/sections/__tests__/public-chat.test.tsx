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
*/
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { PublicChat } from '../public-chat'

const statusHook = vi.hoisted(() => vi.fn())

vi.mock('@/hooks/use-status', () => ({
  useStatus: statusHook,
}))

vi.mock('../../../api', () => ({
  sendPublicChatMessage: vi.fn(),
}))

describe('public chat availability', () => {
  beforeEach(() => {
    statusHook.mockReset()
  })

  it('enables the composer when status exposes public chat as enabled', () => {
    statusHook.mockReturnValue({
      status: { public_chat_enabled: true },
    })

    render(<PublicChat />)

    expect(screen.getByRole('textbox', { name: 'Message' })).toBeEnabled()
  })

  it('disables the composer when public chat is disabled', () => {
    statusHook.mockReturnValue({
      status: { public_chat_enabled: false },
    })

    render(<PublicChat />)

    expect(screen.getByRole('textbox', { name: 'Message' })).toBeDisabled()
  })
})
