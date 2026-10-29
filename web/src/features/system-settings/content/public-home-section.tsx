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
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { MultiSelect } from '@/components/multi-select'
import { Combobox } from '@/components/ui/combobox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { getChannels } from '@/features/channels/api'
import { getModels } from '@/features/models/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'

const publicHomeSchema = z.object({
  PublicStatsEnabled: z.boolean(),
  PublicChannelStatusEnabled: z.boolean(),
  PublicChannelStatusHomeMode: z.enum([
    'coarse',
    'coarse_no_page',
    'detailed',
    'detailed_no_page',
  ]),
  PublicChannelStatusChannelMode: z.enum(['all', 'none', 'selected']),
  PublicChannelStatusChannelIDs: z.string(),
  PublicChannelStatusModelEnabled: z.boolean(),
  PublicChatEnabled: z.boolean(),
  PublicChatGroup: z.string().max(100),
  PublicChatRoutingMode: z.enum(['group', 'model']),
  PublicChatModel: z.string().max(200),
  PublicApiGroup: z.string().max(100),
  PublicStatsRequestAdjustment: z.number().int(),
  GroupRatio: z.string(),
  UserUsableGroups: z.string(),
  GroupPolicies: z.string(),
  PublicStatsTokenAdjustment: z.number().int(),
  PublicStatsSuccessAdjustment: z.number().int(),
  PublicStatsFailureAdjustment: z.number().int(),
})

type PublicHomeFormValues = z.infer<typeof publicHomeSchema>

type PublicHomeSectionProps = {
  defaultValues: PublicHomeFormValues
}

const HOME_MODE_OPTIONS = [
  { value: 'coarse', label: 'Compact site bar with the status page' },
  {
    value: 'coarse_no_page',
    label: 'Compact site bar without the status page',
  },
  { value: 'detailed', label: 'Detailed overview with the status page' },
  {
    value: 'detailed_no_page',
    label: 'Detailed overview without the status page',
  },
] as const

const CHANNEL_MODE_OPTIONS = [
  { value: 'all', label: 'Show every channel' },
  { value: 'none', label: 'Hide channel details' },
  { value: 'selected', label: 'Show the selected channels' },
] as const

const adjustmentFields = [
  {
    name: 'PublicStatsRequestAdjustment',
    label: 'Historical request adjustment',
  },
  {
    name: 'PublicStatsTokenAdjustment',
    label: 'Historical token adjustment',
  },
  {
    name: 'PublicStatsSuccessAdjustment',
    label: 'Historical successful request adjustment',
  },
  {
    name: 'PublicStatsFailureAdjustment',
    label: 'Historical failed request adjustment',
  },
] as const

export function PublicHomeSection(props: PublicHomeSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const modelQuery = useQuery({
    queryKey: ['public-home-models'],
    queryFn: async () =>
      requireServerSuccess(
        await getModels({ include_channel_models: true, p: 1, page_size: 100 })
      ),
  })
  const channelQuery = useQuery({
    queryKey: ['public-home-status-channels'],
    queryFn: async () =>
      requireServerSuccess(
        await getChannels({ p: 1, page_size: 1000, id_sort: true })
      ),
  })
  const form = useForm<PublicHomeFormValues>({
    resolver: zodResolver(publicHomeSchema),
    defaultValues: props.defaultValues,
  })

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [form, props.defaultValues])

  const onSubmit = async (values: PublicHomeFormValues) => {
    const nextValues = { ...values }
    if (values.PublicChatRoutingMode === 'group' && values.PublicChatGroup) {
      try {
        const policies = JSON.parse(values.GroupPolicies || '{}') as Record<
          string,
          Record<string, unknown>
        >
        const currentPolicy = policies[values.PublicChatGroup] ?? {
          endpoint: '/',
          require_key: true,
          auto_route: false,
          route_strategy: 'sequential',
        }
        policies[values.PublicChatGroup] = {
          ...currentPolicy,
          allow_no_model: true,
          no_model_strategy: currentPolicy.no_model_strategy ?? 'sequential',
        }
        nextValues.GroupPolicies = JSON.stringify(policies, null, 2)
      } catch {
        form.setError('GroupPolicies', {
          type: 'validate',
          message: t('Group policy configuration is invalid.'),
        })
        return
      }
    }

    const updates = Object.entries(nextValues).filter(
      ([key, value]) =>
        value !== props.defaultValues[key as keyof PublicHomeFormValues]
    )

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  const statsEnabled = form.watch('PublicStatsEnabled')
  const channelStatusEnabled = form.watch('PublicChannelStatusEnabled')
  const channelMode = form.watch('PublicChannelStatusChannelMode')
  const selectedChannelIDs = form.watch('PublicChannelStatusChannelIDs')
  const chatEnabled = form.watch('PublicChatEnabled')
  const selectedGroup = form.watch('PublicChatGroup')
  const routingMode = form.watch('PublicChatRoutingMode')
  const groupOptions = useMemo(() => {
    try {
      const ratioMap = JSON.parse(
        props.defaultValues.GroupRatio || '{}'
      ) as Record<string, unknown>
      return Object.keys(ratioMap)
        .filter((group) => group !== 'auto')
        .sort()
    } catch {
      return []
    }
  }, [props.defaultValues.GroupRatio])
  const selectedGroupModels = useMemo(
    () =>
      (modelQuery.data?.data?.items ?? [])
        .filter((model) => model.enable_groups?.includes(selectedGroup))
        .map((model) => model.model_name)
        .sort(),
    [modelQuery.data, selectedGroup]
  )
  const channelOptions = useMemo(
    () =>
      (channelQuery.data?.data?.items ?? [])
        .map((channel) => ({
          value: String(channel.id),
          label: channel.name
            ? `${channel.name} (#${channel.id})`
            : `#${channel.id}`,
        }))
        .sort((a, b) => a.label.localeCompare(b.label)),
    [channelQuery.data]
  )
  // The option stores a JSON array so an empty selection and "no channel" stay
  // distinguishable from a list of one.
  const selectedChannelValues = useMemo(() => {
    try {
      const parsed = JSON.parse(selectedChannelIDs || '[]')
      return Array.isArray(parsed) ? parsed.map((id) => String(id)) : []
    } catch {
      return []
    }
  }, [selectedChannelIDs])

  useEffect(() => {
    if (
      routingMode === 'model' &&
      !selectedGroupModels.includes(form.getValues('PublicChatModel'))
    ) {
      form.setValue('PublicChatModel', '')
    }
  }, [form, routingMode, selectedGroupModels])

  return (
    <SettingsSection title={t('Public Home')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='PublicStatsEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable public runtime statistics')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Expose request, token, and final success metrics on the home page.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <FormField
            control={form.control}
            name='PublicChannelStatusEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable public service status')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Publish the read-only site status under the public runtime data. The options below decide how much of it is shown.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <div className='grid gap-6 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='PublicChannelStatusHomeMode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Home page service status detail')}</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={!channelStatusEnabled}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {HOME_MODE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'A compact bar shows only the whole-site status; the detailed overview repeats the status page card. Both come with or without the dedicated page.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='PublicChannelStatusChannelMode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Channel details on the status page')}
                  </FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={!channelStatusEnabled}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {CHANNEL_MODE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'Hiding channel details removes channel names only; the site and model figures keep counting every attempt.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          {channelMode === 'selected' && (
            <FormField
              control={form.control}
              name='PublicChannelStatusChannelIDs'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Published channels')}</FormLabel>
                  <FormControl>
                    <MultiSelect
                      options={channelOptions}
                      selected={selectedChannelValues}
                      onChange={(values) =>
                        field.onChange(JSON.stringify(values.map(Number)))
                      }
                      placeholder={t('Select channels')}
                      emptyText={t('No matching channels')}
                      disabled={!channelStatusEnabled || channelQuery.isPending}
                      maxVisibleChips={20}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Only the selected channels are named on the status page. Disabled channels are never listed.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
          <FormField
            control={form.control}
            name='PublicChannelStatusModelEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Show model status')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Publish the per-model availability section of the status page.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={!channelStatusEnabled}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <div className='grid gap-6 sm:grid-cols-2'>
            {adjustmentFields.map((fieldConfig) => (
              <FormField
                key={fieldConfig.name}
                control={form.control}
                name={fieldConfig.name}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t(fieldConfig.label)}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step={1}
                        disabled={!statsEnabled}
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Add or subtract historical values from the live totals.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ))}
          </div>
          <FormField
            control={form.control}
            name='PublicChatEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable keyless online chat')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Allow visitors to test the configured routing group without an API key.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <div className='grid gap-6 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='PublicChatGroup'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Public chat routing group')}</FormLabel>
                  <FormControl>
                    <Combobox
                      options={groupOptions.map((group) => ({
                        value: group,
                        label: group,
                      }))}
                      value={field.value || null}
                      onValueChange={(value) => field.onChange(value ?? '')}
                      placeholder={t('Select a group')}
                      aria-label={t('Public chat routing group')}
                      disabled={!chatEnabled}
                      className='w-full'
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Choose an existing group. Group mode also enables no-model routing in its policy.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='PublicChatRoutingMode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Public chat routing mode')}</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={!chatEnabled}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value='group'>
                        {t('Follow group configuration')}
                      </SelectItem>
                      <SelectItem value='model'>
                        {t('Use a specified group model')}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'Follow group configuration uses its no-model strategy; specified model pins one model in the selected group.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name='PublicApiGroup'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Homepage API reference group')}</FormLabel>
                <FormControl>
                  <Combobox
                    options={groupOptions.map((group) => ({
                      value: group,
                      label: group,
                    }))}
                    value={field.value || null}
                    onValueChange={(value) => field.onChange(value ?? '')}
                    placeholder={t('Select a group')}
                    aria-label={t('Homepage API reference group')}
                    className='w-full'
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'The homepage API reference uses this group to describe API key, model selection, and fallback behavior.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          {routingMode === 'model' && (
            <FormField
              control={form.control}
              name='PublicChatModel'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Public chat model')}</FormLabel>
                  <FormControl>
                    <Combobox
                      options={selectedGroupModels.map((model) => ({
                        value: model,
                        label: model,
                      }))}
                      value={field.value || null}
                      onValueChange={(value) => field.onChange(value ?? '')}
                      placeholder={t('Select a model')}
                      aria-label={t('Public chat model')}
                      disabled={!chatEnabled || !selectedGroup}
                      className='w-full'
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Only models enabled in the selected group can be used.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
