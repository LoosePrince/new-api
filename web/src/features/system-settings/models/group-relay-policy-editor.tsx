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
import { Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

export type GroupRelayPolicy = {
  endpoint?: string
  require_key?: boolean
  auto_route?: boolean
  route_strategy?: 'sequential' | 'random' | 'nearest'
  fallback_times?: number
  fallback_group?: string
  fallback_model?: string
  ignore_model?: boolean
  allow_no_model?: boolean
  no_model_strategy?: 'sequential' | 'random' | 'specified'
  no_model?: string
}

type GroupRelayPolicyEditorProps = {
  value: string
  groupOptions: string[]
  modelOptionsByGroup: Record<string, string[]>
  onChange: (value: string) => void
}

function parsePolicies(value: string): Record<string, GroupRelayPolicy> | null {
  try {
    const parsed: unknown = JSON.parse(value || '{}')
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return null
    }

    const policies: Record<string, GroupRelayPolicy> = {}
    for (const [group, rawPolicy] of Object.entries(parsed)) {
      if (
        !rawPolicy ||
        typeof rawPolicy !== 'object' ||
        Array.isArray(rawPolicy)
      ) {
        return null
      }
      const policy = rawPolicy as Record<string, unknown>
      policies[group] = {
        endpoint: typeof policy.endpoint === 'string' ? policy.endpoint : '/',
        require_key:
          typeof policy.require_key === 'boolean' ? policy.require_key : true,
        auto_route:
          typeof policy.auto_route === 'boolean' ? policy.auto_route : false,
        route_strategy:
          policy.route_strategy === 'random' ||
          policy.route_strategy === 'nearest'
            ? policy.route_strategy
            : 'sequential',
        fallback_times:
          typeof policy.fallback_times === 'number' &&
          Number.isInteger(policy.fallback_times) &&
          policy.fallback_times >= 0
            ? policy.fallback_times
            : 0,
        fallback_group:
          typeof policy.fallback_group === 'string'
            ? policy.fallback_group
            : '',
        fallback_model:
          typeof policy.fallback_model === 'string'
            ? policy.fallback_model
            : '',
        ignore_model: policy.ignore_model === true,
        allow_no_model: policy.allow_no_model === true,
        no_model_strategy:
          policy.no_model_strategy === 'random' ||
          policy.no_model_strategy === 'specified'
            ? policy.no_model_strategy
            : 'sequential',
        no_model: typeof policy.no_model === 'string' ? policy.no_model : '',
      }
    }
    return policies
  } catch {
    return null
  }
}

export function GroupRelayPolicyEditor(props: GroupRelayPolicyEditorProps) {
  const { t } = useTranslation()
  const [newGroup, setNewGroup] = useState<string | null>(null)
  const policies = useMemo(() => parsePolicies(props.value), [props.value])
  const configuredGroups = useMemo(
    () => Object.keys(policies ?? {}),
    [policies]
  )
  const availableGroups = useMemo(
    () =>
      props.groupOptions.filter((group) => !configuredGroups.includes(group)),
    [configuredGroups, props.groupOptions]
  )

  const emit = (next: Record<string, GroupRelayPolicy>) => {
    props.onChange(JSON.stringify(next, null, 2))
  }

  const updatePolicy = (group: string, updates: Partial<GroupRelayPolicy>) => {
    if (!policies) return
    emit({ ...policies, [group]: { ...policies[group], ...updates } })
  }

  const addPolicy = (group: string) => {
    if (!policies || !group) return
    emit({
      ...policies,
      [group]: {
        endpoint: '/',
        require_key: true,
        auto_route: false,
        route_strategy: 'sequential',
        fallback_times: 0,
        fallback_group: '',
        fallback_model: '',
        ignore_model: false,
        allow_no_model: false,
        no_model_strategy: 'sequential',
        no_model: '',
      },
    })
    setNewGroup(null)
  }

  const deletePolicy = (group: string) => {
    if (!policies) return
    const next = { ...policies }
    delete next[group]
    emit(next)
  }

  return (
    <div className='flex flex-col gap-4'>
      <div className='flex flex-col gap-2 sm:flex-row sm:items-end'>
        <div className='min-w-0 flex-1'>
          <Label htmlFor='relay-policy-group'>
            {t('Add policy for group')}
          </Label>
          <Combobox
            id='relay-policy-group'
            options={availableGroups.map((group) => ({
              value: group,
              label: group,
            }))}
            value={newGroup}
            onValueChange={setNewGroup}
            placeholder={t('Select a group')}
            aria-label={t('Add policy for group')}
            className='mt-2 w-full'
            disabled={!policies || availableGroups.length === 0}
          />
        </div>
        <Button
          type='button'
          size='sm'
          onClick={() => newGroup && addPolicy(newGroup)}
          disabled={!newGroup || !policies}
        >
          <Plus data-icon='inline-start' />
          {t('Add policy')}
        </Button>
      </div>

      {policies === null && (
        <p role='alert' className='text-destructive text-sm'>
          {t('Policies JSON is invalid. Switch to JSON mode to fix it.')}
        </p>
      )}
      {policies !== null && configuredGroups.length === 0 && (
        <EmptyState
          className='min-h-40'
          title={t('No relay policies configured')}
          description={t(
            'Add a group policy to configure its endpoint and routing.'
          )}
        />
      )}
      {policies !== null && configuredGroups.length > 0 && (
        <div className='flex flex-col gap-3'>
          {configuredGroups.map((group) => {
            const policy = policies[group] ?? {}
            const requireKey = policy.require_key !== false
            const autoRoute = policy.auto_route === true
            const strategy = policy.route_strategy ?? 'sequential'
            return (
              <section
                key={group}
                className='flex flex-col gap-4 rounded-md border p-3'
              >
                <header className='flex items-center justify-between gap-3'>
                  <h3 className='min-w-0 text-sm font-medium break-all'>
                    {group}
                  </h3>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon-sm'
                    aria-label={t('Delete policy for {{group}}', { group })}
                    onClick={() => deletePolicy(group)}
                  >
                    <Trash2 aria-hidden='true' />
                  </Button>
                </header>
                <div className='flex flex-col gap-4'>
                  <div className='flex flex-col gap-2'>
                    <Label htmlFor={`relay-endpoint-${group}`}>
                      {t('Endpoint prefix')}
                    </Label>
                    <Input
                      id={`relay-endpoint-${group}`}
                      value={policy.endpoint ?? '/'}
                      placeholder='/'
                      onChange={(event) =>
                        updatePolicy(group, { endpoint: event.target.value })
                      }
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        'Use / for the default API path or a prefix such as /123.'
                      )}
                    </p>
                  </div>

                  <div className='grid gap-3 sm:grid-cols-2'>
                    <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
                      <Label htmlFor={`relay-require-key-${group}`}>
                        {t('Require API key')}
                      </Label>
                      <Switch
                        id={`relay-require-key-${group}`}
                        checked={requireKey}
                        onCheckedChange={(checked) =>
                          updatePolicy(group, { require_key: checked })
                        }
                      />
                    </div>
                    <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
                      <Label htmlFor={`relay-auto-route-${group}`}>
                        {t('Enable model fallback')}
                      </Label>
                      <Switch
                        id={`relay-auto-route-${group}`}
                        checked={autoRoute}
                        onCheckedChange={(checked) =>
                          updatePolicy(group, { auto_route: checked })
                        }
                      />
                    </div>
                  </div>

                  {autoRoute && (
                    <div className='grid gap-4 sm:grid-cols-2'>
                      <div className='flex flex-col gap-2'>
                        <Label htmlFor={`relay-route-strategy-${group}`}>
                          {t('Fallback strategy')}
                        </Label>
                        <Select
                          value={strategy}
                          onValueChange={(value) =>
                            value &&
                            updatePolicy(group, {
                              route_strategy:
                                value as GroupRelayPolicy['route_strategy'],
                            })
                          }
                        >
                          <SelectTrigger
                            id={`relay-route-strategy-${group}`}
                            className='w-full'
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value='sequential'>
                              {t('Sequential')}
                            </SelectItem>
                            <SelectItem value='random'>
                              {t('Random')}
                            </SelectItem>
                            <SelectItem value='nearest'>
                              {t('Nearest model')}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className='flex flex-col gap-2'>
                        <Label htmlFor={`relay-fallback-times-${group}`}>
                          {t('Fallback attempts')}
                        </Label>
                        <Input
                          id={`relay-fallback-times-${group}`}
                          type='number'
                          min={0}
                          step={1}
                          value={policy.fallback_times ?? 0}
                          onChange={(event) =>
                            updatePolicy(group, {
                              fallback_times: Math.max(
                                0,
                                Number.parseInt(event.target.value, 10) || 0
                              ),
                            })
                          }
                        />
                      </div>
                      <div className='flex flex-col gap-2'>
                        <Label htmlFor={`relay-fallback-group-${group}`}>
                          {t('Fallback group')}
                        </Label>
                        <Select
                          value={policy.fallback_group || 'none'}
                          onValueChange={(value) =>
                            updatePolicy(group, {
                              fallback_group:
                                value && value !== 'none' ? value : '',
                              fallback_model: '',
                            })
                          }
                        >
                          <SelectTrigger
                            id={`relay-fallback-group-${group}`}
                            className='w-full'
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value='none'>{t('None')}</SelectItem>
                            {props.groupOptions
                              .filter((candidate) => candidate !== group)
                              .map((candidate) => (
                                <SelectItem key={candidate} value={candidate}>
                                  {candidate}
                                </SelectItem>
                              ))}
                          </SelectContent>
                        </Select>
                      </div>
                      {policy.fallback_group ? (
                        <div className='flex flex-col gap-2'>
                          <Label htmlFor={`relay-fallback-model-${group}`}>
                            {t('Fallback model')}
                          </Label>
                          <Combobox
                            id={`relay-fallback-model-${group}`}
                            options={[
                              { value: 'random', label: t('Random model') },
                              ...(
                                props.modelOptionsByGroup[
                                  policy.fallback_group
                                ] ?? []
                              ).map((model) => ({
                                value: model,
                                label: model,
                              })),
                            ]}
                            value={policy.fallback_model || undefined}
                            onValueChange={(value) =>
                              updatePolicy(group, {
                                fallback_model: value ?? '',
                              })
                            }
                            placeholder={t('Select a model')}
                            aria-label={t('Fallback model')}
                            className='w-full'
                          />
                        </div>
                      ) : null}
                    </div>
                  )}
                  <div className='grid gap-3 sm:grid-cols-2'>
                    <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
                      <Label htmlFor={`relay-ignore-model-${group}`}>
                        {t('Ignore requested model')}
                      </Label>
                      <Switch
                        id={`relay-ignore-model-${group}`}
                        checked={policy.ignore_model === true}
                        onCheckedChange={(checked) =>
                          updatePolicy(group, { ignore_model: checked })
                        }
                      />
                    </div>
                    <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
                      <Label htmlFor={`relay-allow-no-model-${group}`}>
                        {t('Allow requests without a model')}
                      </Label>
                      <Switch
                        id={`relay-allow-no-model-${group}`}
                        checked={policy.allow_no_model === true}
                        onCheckedChange={(checked) =>
                          updatePolicy(group, { allow_no_model: checked })
                        }
                      />
                    </div>
                  </div>
                  {policy.allow_no_model && (
                    <div className='grid gap-4 sm:grid-cols-2'>
                      <div className='flex flex-col gap-2'>
                        <Label htmlFor={`relay-no-model-strategy-${group}`}>
                          {t('No-model selection strategy')}
                        </Label>
                        <Select
                          value={policy.no_model_strategy ?? 'sequential'}
                          onValueChange={(value) =>
                            value &&
                            updatePolicy(group, {
                              no_model_strategy:
                                value as GroupRelayPolicy['no_model_strategy'],
                            })
                          }
                        >
                          <SelectTrigger
                            id={`relay-no-model-strategy-${group}`}
                            className='w-full'
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value='sequential'>
                              {t('Sequential')}
                            </SelectItem>
                            <SelectItem value='random'>
                              {t('Random')}
                            </SelectItem>
                            <SelectItem value='specified'>
                              {t('Specified model')}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      {policy.no_model_strategy === 'specified' && (
                        <div className='flex flex-col gap-2'>
                          <Label htmlFor={`relay-no-model-${group}`}>
                            {t('Specified model')}
                          </Label>
                          <Combobox
                            id={`relay-no-model-${group}`}
                            options={(
                              props.modelOptionsByGroup[group] ?? []
                            ).map((model) => ({ value: model, label: model }))}
                            value={policy.no_model || undefined}
                            onValueChange={(value) =>
                              updatePolicy(group, { no_model: value ?? '' })
                            }
                            placeholder={t('Select a model')}
                            aria-label={t('Specified model')}
                            className='w-full'
                          />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </section>
            )
          })}
        </div>
      )}
    </div>
  )
}
