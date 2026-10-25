import { createFileRoute } from '@tanstack/react-router'

import { ContributorChannels } from '@/features/channels/components/contributor-channels'

export const Route = createFileRoute('/_authenticated/contributions/channels')({
  component: ContributorChannels,
})
