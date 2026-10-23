/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { getModels } from '@/features/models/api'
import { GroupRelayPolicyEditor } from '@/features/system-settings/models/group-relay-policy-editor'
import { requireServerSuccess } from '@/lib/server-error-message'

import { SettingsCard } from '../components/settings-card'
import { useSystemOptions } from '../hooks/use-system-options'
import { useUpdateOption } from '../hooks/use-update-option'

function parseObject(value: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(value || '{}')
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
  } catch {
    // Keep malformed settings visible to the editor so they can be repaired.
  }
  return {}
}

export function RelayPolicySection() {
  const { t } = useTranslation()
  const query = useSystemOptions()
  const modelsQuery = useQuery({
    queryKey: ['relay-policy-models'],
    queryFn: async () => {
      const firstPage = requireServerSuccess(
        await getModels({ include_channel_models: true, p: 1, page_size: 100 })
      )
      const models = [...(firstPage.data?.items ?? [])]
      const total = firstPage.data?.total ?? models.length
      let page = 2
      while (models.length < total) {
        const nextPage = requireServerSuccess(
          await getModels({
            include_channel_models: true,
            p: page,
            page_size: 100,
          })
        )
        const items = nextPage.data?.items ?? []
        if (items.length === 0) break
        models.push(...items)
        page++
      }
      return models
    },
  })
  const updateOption = useUpdateOption()
  const policyOption = query.data?.data.find(
    (option) => option.key === 'GroupPolicies'
  )
  const currentValue = policyOption?.value ?? '{}'
  const [value, setValue] = useState(currentValue)
  const groups = useMemo(
    () =>
      Object.keys(
        parseObject(
          query.data?.data.find((option) => option.key === 'GroupRatio')
            ?.value ?? '{}'
        )
      ).sort(),
    [query.data]
  )
  const modelsByGroup = useMemo(() => {
    const result: Record<string, string[]> = {}
    for (const model of modelsQuery.data ?? []) {
      for (const group of model.enable_groups ?? []) {
        if (group === 'auto') continue
        result[group] ??= []
        result[group].push(model.model_name)
      }
    }
    for (const group of Object.keys(result)) {
      result[group] = [...new Set(result[group])].sort()
    }
    return result
  }, [modelsQuery.data])

  useEffect(() => setValue(currentValue), [currentValue])

  if (query.isPending) return <LoadingState />
  if (query.isError) {
    return (
      <ErrorState
        title={t('Failed to load settings')}
        onRetry={() => void query.refetch()}
      />
    )
  }

  return (
    <div className='space-y-4'>
      <SettingsCard
        title={t('Relay access policies')}
        description={t(
          'Endpoint aliases are access paths only; API-key group authorization remains authoritative.'
        )}
        className='shadow-none'
      >
        <GroupRelayPolicyEditor
          value={value}
          groupOptions={groups}
          modelOptionsByGroup={modelsByGroup}
          onChange={setValue}
        />
      </SettingsCard>
      <div className='flex justify-end'>
        <Button
          type='button'
          size='sm'
          disabled={updateOption.isPending || value === currentValue}
          onClick={() =>
            void updateOption.mutateAsync({ key: 'GroupPolicies', value })
          }
        >
          {updateOption.isPending ? t('Saving...') : t('Save Changes')}
        </Button>
      </div>
    </div>
  )
}
