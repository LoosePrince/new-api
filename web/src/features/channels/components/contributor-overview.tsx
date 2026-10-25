import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Radio, Server, TrendingUp, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatNumber } from '@/lib/format'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getContributorOverview } from '../api'

export function ContributorOverview() {
  const { t } = useTranslation()
  const overviewQuery = useQuery({
    queryKey: ['contributor-overview'],
    queryFn: async () => requireServerSuccess(await getContributorOverview()),
  })
  const data = overviewQuery.data?.data
  const cards = [
    {
      label: t('Contribution Channels'),
      value: data?.channel_count ?? 0,
      icon: Server,
    },
    {
      label: t('Enabled Contribution Channels'),
      value: data?.enabled_channel_count ?? 0,
      icon: Radio,
    },
    {
      label: t('Channel Requests'),
      value: data?.request_count ?? 0,
      icon: TrendingUp,
    },
    {
      label: t('Channel Tokens Used'),
      value: data?.token_used ?? 0,
      icon: Zap,
    },
  ]

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Contribution Overview')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button render={<Link to='/contributions/channels' />}>
          {t('Manage Contribution Channels')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
            {cards.map((card) => {
              const Icon = card.icon
              return (
                <Card key={card.label}>
                  <CardHeader className='flex flex-row items-center justify-between gap-3 pb-2'>
                    <CardTitle className='text-muted-foreground text-sm font-medium'>
                      {card.label}
                    </CardTitle>
                    <Icon
                      className='text-muted-foreground size-4'
                      aria-hidden='true'
                    />
                  </CardHeader>
                  <CardContent className='text-2xl font-semibold'>
                    {formatNumber(Number(card.value))}
                  </CardContent>
                </Card>
              )
            })}
          </div>
          <Card>
            <CardHeader>
              <CardTitle>{t('Channel Model Ranking')}</CardTitle>
            </CardHeader>
            <CardContent>
              {data?.model_ranking?.length ? (
                <div className='divide-y'>
                  {data.model_ranking.map((item) => (
                    <div
                      key={item.model_name}
                      className='flex items-center justify-between gap-3 py-3'
                    >
                      <span className='truncate'>
                        {item.model_name || t('Unknown model')}
                      </span>
                      <span className='text-muted-foreground shrink-0 text-sm'>
                        {formatNumber(item.request_count)} {t('requests')} ·{' '}
                        {formatNumber(item.token_used)} {t('tokens')}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className='text-muted-foreground text-sm'>
                  {t('No channel usage yet')}
                </p>
              )}
            </CardContent>
          </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
