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

import { ApiAccess } from '../api-access'

const statusHook = vi.hoisted(() => vi.fn())

vi.mock('@/hooks/use-status', () => ({
  useStatus: statusHook,
}))

describe('homepage API access', () => {
  beforeEach(() => {
    statusHook.mockReset()
  })

  it('shows only configured routes and values from public status', () => {
    statusHook.mockReturnValue({
      status: {
        api_info_enabled: true,
        server_address: 'http://localhost:3000',
        public_api_endpoint: '/group-api',
        api_info: [
          {
            url: 'https://api.example.com',
            route: 'Configured route',
            description: 'Configured description',
            color: 'green',
          },
        ],
        public_api_require_key: false,
        public_api_allow_no_model: true,
        public_api_has_fallback: true,
      },
    })

    render(<ApiAccess />)

    expect(screen.getByText('https://api.example.com')).toBeInTheDocument()
    expect(screen.getByText('POST /v1/chat/completions')).toBeInTheDocument()
    expect(screen.getByText('POST /v1/messages')).toBeInTheDocument()
    expect(screen.queryByText('Configured route')).not.toBeInTheDocument()
    expect(screen.getByText(/任意字符|Any API key value/i)).toBeInTheDocument()
    expect(
      screen.getByText(/支持相近模型名称|Similar model names are supported/i)
    ).toBeInTheDocument()
    expect(screen.getByText(/模型请求失败|model failure/i)).toBeInTheDocument()
    expect(
      screen.queryByText(/public-model|smart-model|fallback-model|backup/)
    ).toBeNull()
    expect(screen.queryByText('/v1/messages')).not.toBeInTheDocument()
  })

  it('shows required key, required model, and no-fallback guidance', () => {
    statusHook.mockReturnValue({
      status: {
        server_address: 'https://gateway.example.com',
        api_info: [],
        public_api_require_key: true,
        public_api_allow_no_model: false,
        public_api_has_fallback: false,
        public_api_fallback_times: 0,
      },
    })

    render(<ApiAccess />)

    expect(
      screen.getByText(/有效的 API 密钥|A valid API key is required/i)
    ).toBeInTheDocument()
    expect(
      screen.getByText(/请求需要提供模型名称|A model name is required/i)
    ).toBeInTheDocument()
    expect(
      screen.getByText(/未配置模型自动回退|No automatic model fallback/i)
    ).toBeInTheDocument()
  })

  it('uses the current site when the server address is the default localhost URL', () => {
    statusHook.mockReturnValue({
      status: {
        server_address: 'http://localhost:3000/',
        api_info: [],
      },
    })

    render(<ApiAccess />)

    expect(screen.getByText(window.location.origin)).toBeInTheDocument()
  })

  it('shows built-in routes when no custom route list is configured', () => {
    statusHook.mockReturnValue({
      status: {
        api_info_enabled: true,
        server_address: 'https://gateway.example.com',
        api_info: [],
      },
    })

    render(<ApiAccess />)

    expect(
      screen.getByRole('region', { name: 'API access' })
    ).toBeInTheDocument()
    expect(screen.getByText('POST /v1/chat/completions')).toBeInTheDocument()
    expect(screen.getByText('POST /v1/messages')).toBeInTheDocument()
  })

  it('uses built-in routes when custom API info is disabled', () => {
    statusHook.mockReturnValue({
      status: { api_info_enabled: false },
    })

    render(<ApiAccess />)

    expect(
      screen.getByRole('region', { name: 'API access' })
    ).toBeInTheDocument()
    expect(screen.getByText('POST /v1/chat/completions')).toBeInTheDocument()
  })

  it('hides the section when status is unavailable', () => {
    statusHook.mockReturnValue({ status: null })

    render(<ApiAccess />)

    expect(screen.queryByRole('region', { name: 'API access' })).toBeNull()
  })
})
