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
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

import { getPublicChannelStatus, PUBLIC_STATUS_QUERY_KEY } from './api'
import { PublicStatusModelCard } from './components/public-status-model-card'
import { PublicStatusOverviewCard } from './components/status-overview-card'

const REFRESH_INTERVAL_MS = 60_000

export function PublicStatus() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const statusQuery = useQuery({
    queryKey: [...PUBLIC_STATUS_QUERY_KEY, 'full'],
    queryFn: () => getPublicChannelStatus(false),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: REFRESH_INTERVAL_MS,
    meta: { errorToast: false },
  })

  const data = statusQuery.data
  const summary = data?.summary
  const models = data?.models ?? []
  // The per-model section is an independent publication switch; a site may
  // publish its own status without listing which models it serves.
  const showModels = data?.models_enabled !== false

  if (statusQuery.isPending) {
    return (
      <PublicLayout>
        <div className='mx-auto flex max-w-5xl flex-col gap-4 py-10'>
          <Skeleton className='h-9 w-64' />
          <Skeleton className='h-4 w-full max-w-lg' />
          <Skeleton className='h-40 w-full rounded-xl' />
          <Skeleton className='h-32 w-full rounded-xl' />
          <Skeleton className='h-32 w-full rounded-xl' />
        </div>
      </PublicLayout>
    )
  }

  if (!data?.enabled) {
    return (
      <PublicLayout>
        <div className='py-16'>
          <EmptyState
            title={t('Service status is not published')}
            description={t(
              'The administrator has not enabled the public service status page.'
            )}
          />
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout>
      <PageTransition>
        <div className='mx-auto flex max-w-5xl flex-col gap-6 py-8 sm:py-10'>
          <div className='flex flex-wrap items-start justify-between gap-4'>
            <div>
              <h1 className='text-2xl font-semibold tracking-tight sm:text-3xl'>
                {t('Service Status')}
              </h1>
              <p className='text-muted-foreground mt-2 max-w-2xl text-sm leading-6'>
                {t(
                  'Availability is measured from real request outcomes of the last 30 days. Each upstream attempt counts on its own, so a fallback that succeeds does not hide the model that failed before it.'
                )}
              </p>
            </div>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void statusQuery.refetch()}
              disabled={statusQuery.isFetching}
            >
              <RefreshCw
                className={cn(
                  'mr-2 size-3.5',
                  statusQuery.isFetching && 'animate-spin'
                )}
                aria-hidden='true'
              />
              {t('Refresh')}
            </Button>
          </div>

          <PublicStatusOverviewCard
            summary={summary}
            locale={locale}
            channelsVisible={data.channels_visible}
            modelsVisible={showModels}
          />

          {showModels && (
            <section className='space-y-3'>
              <div className='flex flex-wrap items-end justify-between gap-2'>
                <h2 className='text-lg font-semibold tracking-tight'>
                  {t('Model availability')}
                </h2>
                <span className='text-muted-foreground text-xs'>
                  {t('{{count}} models with traffic in the last 7 days', {
                    count: models.length,
                  })}
                </span>
              </div>

              {models.length === 0 ? (
                <EmptyState
                  title={t('No model traffic yet')}
                  description={t(
                    'Model availability appears once the site has served requests.'
                  )}
                />
              ) : (
                <div className='space-y-3'>
                  {models.map((model) => (
                    <PublicStatusModelCard
                      key={model.name}
                      model={model}
                      locale={locale}
                    />
                  ))}
                </div>
              )}
            </section>
          )}

          <p className='text-muted-foreground text-xs leading-5'>
            {t(
              'Availability counts each upstream attempt separately. Latency is the average response time of the model that answered, not of the whole fallback chain.'
            )}
          </p>
        </div>
      </PageTransition>
    </PublicLayout>
  )
}
