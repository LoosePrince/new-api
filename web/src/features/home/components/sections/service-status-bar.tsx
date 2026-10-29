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
import { Link } from '@tanstack/react-router'
import { ArrowRight, HeartPulse } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicStatusOverviewCard } from '@/features/public-status/components/status-overview-card'
import { StatusStrip } from '@/features/public-status/components/status-strip'
import { usePublicServiceStatus } from '@/features/public-status/hooks/use-public-service-status'
import {
  formatPublicAvailability,
  PUBLIC_STATUS_META,
} from '@/features/public-status/lib/status'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

/**
 * Site status of the public runtime data area.
 *
 * The operator chooses between two presentations. The compact one conveys only
 * that the site is serving, in a single horizontal bar. The detailed one reuses
 * the overview card of the status page, so the home page and the page never
 * disagree about what the numbers mean. The "View service status" link appears
 * only while the dedicated page stays published.
 */
export function ServiceStatusBar() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const {
    enabled,
    summary,
    detailed,
    pageEnabled,
    channelsVisible,
    modelsVisible,
    isPending,
  } = usePublicServiceStatus()

  if (!isPending && !enabled) return null

  if (detailed) {
    return (
      <div className='mt-4'>
        <PublicStatusOverviewCard
          summary={summary}
          locale={locale}
          isPending={isPending}
          channelsVisible={channelsVisible}
          modelsVisible={modelsVisible}
        />
        {pageEnabled && <StatusPageLink />}
      </div>
    )
  }

  const meta = PUBLIC_STATUS_META[summary?.status ?? 'no_traffic']

  return (
    <div className='border-border/50 bg-muted/10 mt-4 rounded-xl border p-4 sm:p-5'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <HeartPulse
            className={cn('size-4 shrink-0', meta.textClassName)}
            aria-hidden='true'
          />
          <span className='text-sm font-medium'>
            {isPending ? t('Checking service status...') : t(meta.labelKey)}
          </span>
          <span className='text-muted-foreground text-xs tabular-nums'>
            {formatPublicAvailability(summary?.availability_7d, locale)}
          </span>
        </div>
        {pageEnabled && <StatusPageLink />}
      </div>
      <StatusStrip
        history={summary?.history ?? []}
        today={summary?.today ?? []}
        size='sm'
        className='mt-3'
      />
    </div>
  )
}

function StatusPageLink() {
  const { t } = useTranslation()

  return (
    <Link
      to='/status'
      className='text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs font-medium transition-colors'
    >
      {t('View service status')}
      <ArrowRight className='size-3.5' aria-hidden='true' />
    </Link>
  )
}
