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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { GroupRelayPolicyEditor } from '@/features/system-settings/models/group-relay-policy-editor'

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

function modelNamesFromOptions(options: Array<{ key: string; value: string }>) {
  const models = new Set<string>()
  for (const key of ['ModelPrice', 'ModelRatio']) {
    for (const model of Object.keys(parseObject(options.find((option) => option.key === key)?.value ?? '{}'))) {
      models.add(model)
    }
  }
  return [...models].sort()
}

export function RelayPolicySection() {
  const { t } = useTranslation()
  const query = useSystemOptions()
  const updateOption = useUpdateOption()
  const policyOption = query.data?.data.find((option) => option.key === 'GroupPolicies')
  const currentValue = policyOption?.value ?? '{}'
  const [value, setValue] = useState(currentValue)
  const groups = useMemo(
    () => Object.keys(parseObject(query.data?.data.find((option) => option.key === 'GroupRatio')?.value ?? '{}')).sort(),
    [query.data]
  )
  const models = useMemo(
    () => modelNamesFromOptions(query.data?.data ?? []),
    [query.data]
  )

  useEffect(() => setValue(currentValue), [currentValue])

  if (query.isPending) return <LoadingState />
  if (query.isError) {
    return <ErrorState title={t('Failed to load settings')} onRetry={() => void query.refetch()} />
  }

  return (
    <div className='space-y-4'>
      <SettingsCard
        title={t('Relay access policies')}
        description={t('Endpoint aliases are access paths only; API-key group authorization remains authoritative.')}
        className='shadow-none'
      >
        <GroupRelayPolicyEditor
          value={value}
          groupOptions={groups}
          modelOptions={models}
          onChange={setValue}
        />
      </SettingsCard>
      <div className='flex justify-end'>
        <Button
          type='button'
          size='sm'
          disabled={updateOption.isPending || value === currentValue}
          onClick={() => void updateOption.mutateAsync({ key: 'GroupPolicies', value })}
        >
          {updateOption.isPending ? t('Saving...') : t('Save Changes')}
        </Button>
      </div>
    </div>
  )
}
