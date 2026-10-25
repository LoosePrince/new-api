import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, PowerOff, Plus, Trash2 } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { handleServerError } from '@/lib/handle-server-error'
import { requireServerSuccess } from '@/lib/server-error-message'

import {
  deleteContributorChannel,
  disableContributorChannel,
  getContributorChannels,
} from '../api'
import { CHANNEL_STATUS } from '../constants'
import { getChannelTypeLabel } from '../lib/channel-utils'
import type { Channel } from '../types'
import { ChannelsProvider, useChannels } from './channels-provider'
import { ChannelMutateDrawer } from './drawers/channel-mutate-drawer'

function ContributorChannelActions(props: { channel: Channel }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { setCurrentRow, setOpen } = useChannels()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const disableMutation = useMutation({
    mutationFn: () => disableContributorChannel(props.channel.id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['contributor-channels'] }),
    onError: (error) => handleServerError(error, t('Failed to update channel')),
  })
  const deleteMutation = useMutation({
    mutationFn: () => deleteContributorChannel(props.channel.id),
    onSuccess: () => {
      setConfirmDelete(false)
      queryClient.invalidateQueries({ queryKey: ['contributor-channels'] })
    },
    onError: (error) => handleServerError(error, t('Failed to delete channel')),
  })

  return (
    <>
      <div className='flex items-center gap-1'>
        <Button
          variant='ghost'
          size='icon-sm'
          aria-label={t('Edit')}
          onClick={() => {
            setCurrentRow(props.channel)
            setOpen('update-channel')
          }}
        >
          <Pencil />
        </Button>
        {props.channel.status === CHANNEL_STATUS.ENABLED && (
          <Button
            variant='ghost'
            size='icon-sm'
            aria-label={t('Disable')}
            disabled={disableMutation.isPending}
            onClick={() => disableMutation.mutate()}
          >
            <PowerOff />
          </Button>
        )}
        <Button
          variant='ghost'
          size='icon-sm'
          aria-label={t('Delete')}
          onClick={() => setConfirmDelete(true)}
        >
          <Trash2 />
        </Button>
      </div>
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title={t('Delete channel')}
        desc={t('This action cannot be undone.')}
        confirmText={t('Delete')}
        destructive
        handleConfirm={() => deleteMutation.mutate()}
        isLoading={deleteMutation.isPending}
      />
    </>
  )
}

function ContributorChannelsContent() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useChannels()
  const channelsQuery = useQuery({
    queryKey: ['contributor-channels'],
    queryFn: async () => requireServerSuccess(await getContributorChannels()),
  })
  const channels = channelsQuery.data?.data?.items ?? []
  let channelContent: ReactNode
  if (channelsQuery.isLoading) {
    channelContent = (
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {Array.from({ length: 3 }, (_, index) => (
          <Skeleton key={index} className='h-32' />
        ))}
      </div>
    )
  } else if (channels.length === 0) {
    channelContent = (
      <Card>
        <CardContent className='text-muted-foreground py-10 text-center'>
          {t('No contribution channels yet')}
        </CardContent>
      </Card>
    )
  } else {
    channelContent = (
      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {channels.map((channel) => (
          <Card key={channel.id}>
            <CardHeader className='flex flex-row items-start justify-between gap-3'>
              <div className='min-w-0'>
                <CardTitle className='truncate'>{channel.name}</CardTitle>
                <p className='text-muted-foreground mt-1 text-xs'>
                  {getChannelTypeLabel(channel.type)} · #{channel.id}
                </p>
              </div>
              <ContributorChannelActions channel={channel} />
            </CardHeader>
            <CardContent className='text-muted-foreground text-sm'>
              {channel.status === CHANNEL_STATUS.ENABLED
                ? t('Enabled')
                : t('Disabled, awaiting administrator review')}
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>
          {t('Contribution Channels')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button onClick={() => setOpen('create-channel')}>
            <Plus data-icon='inline-start' />
            {t('Add Channel')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>{channelContent}</SectionPageLayout.Content>
      </SectionPageLayout>
      <ChannelMutateDrawer
        open={open === 'create-channel' || open === 'update-channel'}
        onOpenChange={(value) => !value && setOpen(null)}
        currentRow={open === 'update-channel' ? currentRow : null}
        contributorMode
      />
    </>
  )
}

export function ContributorChannels() {
  return (
    <ChannelsProvider>
      <ContributorChannelsContent />
    </ChannelsProvider>
  )
}
