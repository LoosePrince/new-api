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
import { useQuery } from '@tanstack/react-query'
import { Activity, CheckCircle2, Hash, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { usePublicServiceStatus } from '@/features/public-status/hooks/use-public-service-status'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatNumber } from '@/lib/format'

import { getPublicRuntimeStats } from '../../api'
import { ServiceStatusBar } from './service-status-bar'

export function RuntimeStats() {
  const { i18n, t } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const { data, isPending } = useQuery({
    queryKey: ['public-runtime-stats'],
    queryFn: getPublicRuntimeStats,
    refetchInterval: 60_000,
    meta: { errorToast: false },
  })
  // The site status is an independent switch, so this section stays on the page
  // when only the service status is published.
  const serviceStatus = usePublicServiceStatus()

  const stats = data?.data
  const showStats = isPending || stats?.enabled === true
  const showServiceStatus = serviceStatus.isPending || serviceStatus.enabled
  if (!showStats && !showServiceStatus) return null

  const items = [
    {
      label: t('Total requests'),
      value: stats ? formatCompactNumber(stats.requests, locale) : '—',
      icon: Hash,
    },
    {
      label: t('Total tokens'),
      value: stats ? formatCompactNumber(stats.tokens, locale) : '—',
      icon: Zap,
    },
    {
      label: t('Final success rate'),
      value: stats ? `${formatNumber(stats.success_rate, locale)}%` : '—',
      icon: CheckCircle2,
    },
  ]

  return (
    <section
      id='public-stats'
      className='border-border/40 border-b px-6 py-16 md:py-20'
    >
      <div className='mx-auto max-w-6xl'>
        <div className='mb-8 flex items-end justify-between gap-6'>
          <div>
            <div className='text-muted-foreground mb-3 flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase'>
              <Activity className='size-4' aria-hidden='true' />
              {t('Public runtime data')}
            </div>
            <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
              {t('A transparent view of the service')}
            </h2>
          </div>
          <span className='text-muted-foreground hidden text-xs md:block'>
            {t('Updated automatically')}
          </span>
        </div>
        {showStats && (
          <>
            <div className='border-border/50 bg-muted/10 grid overflow-hidden rounded-xl border md:grid-cols-3'>
              {items.map((item) => {
                const Icon = item.icon
                return (
                  <div
                    key={item.label}
                    className='border-border/50 flex min-h-32 flex-col justify-between border-b p-6 last:border-b-0 md:border-r md:border-b-0 md:last:border-r-0'
                  >
                    <Icon
                      className='text-muted-foreground size-5'
                      aria-hidden='true'
                    />
                    <div>
                      <div className='text-3xl font-semibold tracking-tight tabular-nums'>
                        {item.value}
                      </div>
                      <div className='text-muted-foreground mt-1 text-sm'>
                        {item.label}
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
            <p className='text-muted-foreground mt-4 text-xs leading-5'>
              {t(
                'Success rate counts a request once and considers it successful when the final fallback attempt succeeds.'
              )}
            </p>
          </>
        )}
        <ServiceStatusBar />
      </div>
    </section>
  )
}
