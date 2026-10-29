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
import { describe, expect, it } from 'vitest'

import { PublicStatusOverviewCard } from '../components/status-overview-card'
import type { PublicStatusSummary } from '../types'

function summary(): PublicStatusSummary {
  return {
    status: 'normal',
    latency_ms: 1200,
    availability_7d: 99.5,
    availability_30d: 99.1,
    attempts: 120,
    channel_count: 3,
    model_count: 5,
    history: [{ day: 1, status: 'normal', availability: 100 }],
    today: [{ hour: 1, status: 'normal', availability: 100 }],
  }
}

describe('public status overview card', () => {
  it('shows every headline figure while both sections are published', () => {
    render(<PublicStatusOverviewCard summary={summary()} />)

    expect(screen.getByText('Channels:')).toBeInTheDocument()
    expect(screen.getByText('Models:')).toBeInTheDocument()
    expect(screen.getByText('Attempts:')).toBeInTheDocument()
    expect(screen.getByText('7-day availability')).toBeInTheDocument()
  })

  it('drops the channel count when channel details are unpublished', () => {
    render(
      <PublicStatusOverviewCard summary={summary()} channelsVisible={false} />
    )

    expect(screen.queryByText('Channels:')).not.toBeInTheDocument()
    expect(screen.getByText('Models:')).toBeInTheDocument()
  })

  it('drops the model count when the model section is unpublished', () => {
    render(
      <PublicStatusOverviewCard summary={summary()} modelsVisible={false} />
    )

    expect(screen.queryByText('Models:')).not.toBeInTheDocument()
    expect(screen.getByText('Channels:')).toBeInTheDocument()
  })

  it('renders the strip for the days and the hours of today', () => {
    const { container } = render(
      <PublicStatusOverviewCard summary={summary()} />
    )

    const groups = container.querySelectorAll('[role="group"]')
    expect(groups).toHaveLength(1)
    const [dayCells, hourCells] = groups[0].children
    expect(dayCells.children).toHaveLength(1)
    expect(hourCells.children).toHaveLength(1)
  })
})
