import { createFileRoute } from '@tanstack/react-router'

import { ContributorOverview } from '@/features/channels/components/contributor-overview'

export const Route = createFileRoute('/_authenticated/contributions/')({
  component: ContributorOverview,
})
