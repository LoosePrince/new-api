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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { PublicChatStreamChunk } from '../../../types'
import { PublicChat } from '../public-chat'

const statusHook = vi.hoisted(() => vi.fn())
const sendPublicChatMessage = vi.hoisted(() => vi.fn())

vi.mock('@/hooks/use-status', () => ({
  useStatus: statusHook,
}))

vi.mock('../../../api', () => ({
  sendPublicChatMessage,
}))

describe('public chat availability', () => {
  beforeEach(() => {
    statusHook.mockReset()
    sendPublicChatMessage.mockReset()
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

  it('renders streamed assistant text and the model returned by the route', async () => {
    statusHook.mockReturnValue({
      status: {
        public_chat_enabled: true,
        public_chat_group: 'public',
      },
    })
    sendPublicChatMessage.mockImplementation(
      async (
        _endpoint: string,
        _group: string,
        _routingMode: string,
        _model: string,
        _messages: unknown[],
        onChunk: (chunk: PublicChatStreamChunk) => void
      ) => {
        onChunk({ content: 'Hello', model: 'routed-model' })
        await Promise.resolve()
        onChunk({ content: ' there' })
      }
    )

    const user = userEvent.setup()
    render(<PublicChat />)

    await user.type(screen.getByRole('textbox', { name: 'Message' }), 'Hi')
    await user.click(screen.getByRole('button', { name: 'Send' }))

    expect(await screen.findByText('Hello there')).toBeInTheDocument()
    expect(screen.getByText('Model routed-model')).toBeInTheDocument()
  })
})
