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
import {
  Activity,
  Boxes,
  RefreshCw,
  TriangleAlert,
  ShieldAlert,
  Radio,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampRelative } from '@/lib/format'
import { cn } from '@/lib/utils'

import { CHANNEL_STATUS_QUERY_KEY, getChannelStatus } from './api'
import { ModelStatusDialog } from './components/model-status-dialog'
import {
  StatusHistoryBlocks,
  StatusHistoryLegend,
} from './components/status-history-blocks'
import { CHANNEL_HEALTH_STATUS_META, formatAvailability } from './lib/status'
import type { ChannelStatusChannel } from './types'

const AUTO_REFRESH_INTERVAL_MS = 15_000

function SummaryCard(props: {
  icon: typeof Activity
  tone: IconBadgeTone
  label: string
  value: string
  description: string
}) {
  const Icon = props.icon
  return (
    <div className='bg-card flex items-center gap-3 rounded-xl border p-3 sm:p-4'>
      <IconBadge tone={props.tone} size='lg'>
        <Icon />
      </IconBadge>
      <div className='min-w-0'>
        <div className='text-muted-foreground truncate text-xs'>
          {props.label}
        </div>
        <div className='text-lg font-semibold tabular-nums'>{props.value}</div>
        <div className='text-muted-foreground truncate text-xs'>
          {props.description}
        </div>
      </div>
    </div>
  )
}

function ChannelRow(props: {
  channel: ChannelStatusChannel
  locale?: Intl.LocalesArgument
  onOpenModels: () => void
}) {
  const { t } = useTranslation()
  const meta = CHANNEL_HEALTH_STATUS_META[props.channel.status]
  const enabledModels = props.channel.models.filter((item) => item.enabled)

  return (
    <div className='hover:bg-muted/30 space-y-2 border-b px-3 py-3 transition-colors last:border-b-0 sm:px-4'>
      <div className='flex flex-wrap items-start justify-between gap-x-3 gap-y-2'>
        <div className='min-w-0 space-y-1'>
          <div className='flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1'>
            <span className='truncate text-sm font-medium'>
              {props.channel.name}
            </span>
            <span className='text-muted-foreground font-mono text-xs'>
              #{props.channel.id}
            </span>
            <StatusBadge
              label={t(meta.labelKey)}
              variant={meta.variant}
              copyable={false}
            />
            {!props.channel.enabled && (
              <StatusBadge
                label={t('Disabled')}
                variant='neutral'
                copyable={false}
              />
            )}
          </div>
          <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
            <span>
              {t('7-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatAvailability(
                  props.channel.availability_7d,
                  props.locale
                )}
              </span>
            </span>
            <span>
              {t('30-day availability')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatAvailability(
                  props.channel.availability_30d,
                  props.locale
                )}
              </span>
            </span>
            <span>
              {t('Attempts')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {formatNumber(props.channel.attempts_7d, props.locale)}
              </span>
            </span>
            <span>
              {t('Average latency')}:{' '}
              <span className='text-foreground font-medium tabular-nums'>
                {props.channel.latency_ms > 0
                  ? `${formatNumber(props.channel.latency_ms, props.locale)} ms`
                  : '—'}
              </span>
            </span>
            <span>
              {t('Last activity')}:{' '}
              <span className='text-foreground font-medium'>
                {formatTimestampRelative(
                  props.channel.last_activity,
                  'seconds',
                  props.locale
                )}
              </span>
            </span>
          </div>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          <span className='text-muted-foreground text-xs tabular-nums'>
            {t('{{enabled}} / {{total}} models', {
              enabled: enabledModels.length,
              total: props.channel.models.length,
            })}
          </span>
          <Button
            variant='outline'
            size='sm'
            onClick={props.onOpenModels}
            aria-label={t('View models of {{channel}}', {
              channel: props.channel.name,
            })}
          >
            <Boxes className='size-3.5' aria-hidden='true' />
            {t('View models')}
          </Button>
        </div>
      </div>
      <StatusHistoryBlocks days={props.channel.history} />
    </div>
  )
}

export function ChannelStatus() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(
    null
  )

  const statusQuery = useQuery({
    queryKey: CHANNEL_STATUS_QUERY_KEY,
    queryFn: getChannelStatus,
    refetchInterval: autoRefresh ? AUTO_REFRESH_INTERVAL_MS : false,
    refetchIntervalInBackground: false,
    staleTime: AUTO_REFRESH_INTERVAL_MS,
    retry: false,
  })

  const data = statusQuery.data
  const channels = useMemo(() => data?.channels ?? [], [data])
  const selectedChannel = useMemo(
    () => channels.find((channel) => channel.id === selectedChannelId) ?? null,
    [channels, selectedChannelId]
  )

  const summary = data?.summary
  const generatedAt = data?.generated_at

  let content = (
    <EmptyState
      icon={Activity}
      title={t('No channels')}
      description={t('Add a channel to start collecting status history.')}
    />
  )
  if (statusQuery.isLoading) {
    content = <LoadingState />
  } else if (statusQuery.isError) {
    content = (
      <ErrorState
        description={t('Refresh the list and try again.')}
        onRetry={() => void statusQuery.refetch()}
      />
    )
  } else if (channels.length > 0) {
    content = (
      <div className='space-y-4'>
        <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
          <SummaryCard
            icon={Radio}
            tone='success'
            label={t('Normal channels')}
            value={formatNumber(summary?.normal_channels ?? 0, locale)}
            description={t('Availability at or above 95%')}
          />
          <SummaryCard
            icon={TriangleAlert}
            tone='warning'
            label={t('Non-normal channels')}
            value={formatNumber(summary?.non_normal_channels ?? 0, locale)}
            description={t('Availability below 95%')}
          />
          <SummaryCard
            icon={ShieldAlert}
            tone='destructive'
            label={t('Unavailable channels')}
            value={formatNumber(summary?.unavailable_channels ?? 0, locale)}
            description={t('Availability below 30%')}
          />
          <SummaryCard
            icon={Boxes}
            tone='info'
            label={t('Monitored models')}
            value={formatNumber(summary?.monitored_models ?? 0, locale)}
            description={t('{{models}} models on {{channels}} channels', {
              models: summary?.model_count ?? 0,
              channels: summary?.channel_count ?? 0,
            })}
          />
        </div>

        <TitledCard
          title={t('Channel status')}
          description={t('Daily request availability for the last 30 days.')}
          icon={<Activity className='size-4' />}
          iconTone='chart-1'
          disableHoverEffect
          contentClassName='p-0'
          headerClassName='px-3 py-2.5 !pb-2.5 sm:px-4 sm:py-3 sm:!pb-3'
          action={
            generatedAt ? (
              <span className='text-muted-foreground text-xs'>
                {t('Updated {{time}}', {
                  time: formatTimestampRelative(generatedAt, 'seconds', locale),
                })}
              </span>
            ) : null
          }
        >
          <div className='border-b px-3 py-2.5 sm:px-4'>
            <StatusHistoryLegend />
          </div>
          {channels.map((channel) => (
            <ChannelRow
              key={channel.id}
              channel={channel}
              locale={locale}
              onOpenModels={() => setSelectedChannelId(channel.id)}
            />
          ))}
        </TitledCard>
      </div>
    )
  }

  return (
    <>
      <SectionPageLayout fixedContent stackActionsOnMobile>
        <SectionPageLayout.Title>{t('Channel status')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div className='flex items-center gap-2'>
            <Label
              htmlFor='channel-status-auto-refresh'
              className='text-muted-foreground text-xs'
            >
              {t('Auto refresh')}
            </Label>
            <Switch
              id='channel-status-auto-refresh'
              size='sm'
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
            />
          </div>
          <Button
            variant='outline'
            size='sm'
            onClick={() => void statusQuery.refetch()}
            disabled={statusQuery.isFetching}
          >
            <RefreshCw
              className={cn(
                'size-3.5',
                statusQuery.isFetching && 'animate-spin'
              )}
              aria-hidden='true'
            />
            {t('Refresh')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='h-full min-h-0 overflow-auto'>{content}</div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <ModelStatusDialog
        open={selectedChannel != null}
        onOpenChange={(open) => {
          if (!open) setSelectedChannelId(null)
        }}
        channel={selectedChannel}
      />
    </>
  )
}
