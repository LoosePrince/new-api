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
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { PublicStatusSummary } from '@/features/public-status/types'

import { ServiceStatusBar } from '../service-status-bar'

const usePublicServiceStatus = vi.hoisted(() => vi.fn())

vi.mock('@/features/public-status/hooks/use-public-service-status', () => ({
  usePublicServiceStatus,
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { children?: React.ReactNode }) => (
    <a href='/status'>{props.children}</a>
  ),
}))

function summary(status: PublicStatusSummary['status']): PublicStatusSummary {
  return {
    status,
    latency_ms: 1200,
    availability_7d: 99.5,
    availability_30d: 99.1,
    attempts: 120,
    channel_count: 3,
    model_count: 5,
    history: [
      { day: 1, status: 'normal', availability: 100 },
      { day: 2, status: 'degraded', availability: 70 },
    ],
    today: [
      { hour: 1, status: 'normal', availability: 100 },
      { hour: 2, status: 'normal', availability: 100 },
      { hour: 3, status: 'degraded', availability: 70 },
    ],
  }
}

describe('public service status bar', () => {
  beforeEach(() => {
    usePublicServiceStatus.mockReset()
  })

  it('renders nothing while the operator keeps the status unpublished', () => {
    usePublicServiceStatus.mockReturnValue({
      enabled: false,
      summary: undefined,
      detailed: false,
      pageEnabled: false,
      isPending: false,
    })

    const { container } = render(<ServiceStatusBar />)

    expect(container).toBeEmptyDOMElement()
  })

  it('splits the strip into days and the hours of today', () => {
    usePublicServiceStatus.mockReturnValue({
      enabled: true,
      summary: summary('normal'),
      detailed: false,
      pageEnabled: true,
      isPending: false,
    })

    const { container } = render(<ServiceStatusBar />)

    expect(screen.getByText('Operating normally')).toBeInTheDocument()
    expect(screen.getByText('99.5%')).toBeInTheDocument()

    const groups = container.querySelectorAll('[role="group"]')
    expect(groups).toHaveLength(1)
    const [dayCells, hourCells] = groups[0].children
    // Two completed days plus three elapsed hours, never one cell per model.
    expect(dayCells.children).toHaveLength(2)
    expect(hourCells.children).toHaveLength(3)
  })

  it('reuses the status page overview when the detailed mode is published', () => {
    usePublicServiceStatus.mockReturnValue({
      enabled: true,
      summary: summary('normal'),
      detailed: true,
      pageEnabled: true,
      isPending: false,
    })

    render(<ServiceStatusBar />)

    // The detailed overview carries the same headline figures as the page.
    expect(screen.getByText('Overall service status')).toBeInTheDocument()
    expect(screen.getByText('7-day availability')).toBeInTheDocument()
    expect(screen.getByText('30-day availability')).toBeInTheDocument()
    expect(screen.getByText('Average latency')).toBeInTheDocument()
  })

  it('hides the status page link when the page is not published', () => {
    usePublicServiceStatus.mockReturnValue({
      enabled: true,
      summary: summary('normal'),
      detailed: false,
      pageEnabled: false,
      isPending: false,
    })

    render(<ServiceStatusBar />)

    expect(screen.queryByText('View service status')).not.toBeInTheDocument()
    expect(screen.getByText('Operating normally')).toBeInTheDocument()
  })

  it('shows a live check while the first status request is pending', () => {
    usePublicServiceStatus.mockReturnValue({
      enabled: false,
      summary: undefined,
      detailed: false,
      pageEnabled: false,
      isPending: true,
    })

    render(<ServiceStatusBar />)

    expect(screen.getByText('Checking service status...')).toBeInTheDocument()
  })
})
