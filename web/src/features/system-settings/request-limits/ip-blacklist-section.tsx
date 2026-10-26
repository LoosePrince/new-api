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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const ipBlacklistSchema = z.object({
  ip_blacklist_setting: z.object({
    enabled: z.boolean(),
    ip_list: z.string(),
  }),
})

type IPBlacklistFormValues = z.output<typeof ipBlacklistSchema>
type IPBlacklistFormInput = z.input<typeof ipBlacklistSchema>

type IPBlacklistSectionProps = {
  defaultValues: {
    'ip_blacklist_setting.enabled': boolean
    'ip_blacklist_setting.ip_list': string[]
  }
}

const splitLines = (value: string) =>
  value
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean)

const buildFormDefaults = (
  defaults: IPBlacklistSectionProps['defaultValues']
): IPBlacklistFormInput => ({
  ip_blacklist_setting: {
    enabled: defaults['ip_blacklist_setting.enabled'],
    ip_list: defaults['ip_blacklist_setting.ip_list'].join('\n'),
  },
})

export function IPBlacklistSection(props: IPBlacklistSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef(props.defaultValues)
  const formDefaults = useMemo(
    () => buildFormDefaults(props.defaultValues),
    [props.defaultValues]
  )

  const form = useForm<IPBlacklistFormInput, unknown, IPBlacklistFormValues>({
    resolver: zodResolver(ipBlacklistSchema),
    defaultValues: formDefaults,
  })

  useEffect(() => {
    baselineRef.current = props.defaultValues
    form.reset(buildFormDefaults(props.defaultValues))
  }, [props.defaultValues, form])

  const onSubmit = async (data: IPBlacklistFormValues) => {
    const enabled = data.ip_blacklist_setting.enabled
    const ipList = splitLines(data.ip_blacklist_setting.ip_list)
    const updates: Array<{ key: string; value: string }> = []
    if (enabled !== baselineRef.current['ip_blacklist_setting.enabled']) {
      updates.push({
        key: 'ip_blacklist_setting.enabled',
        value: String(enabled),
      })
    }
    if (
      JSON.stringify(ipList) !==
      JSON.stringify(baselineRef.current['ip_blacklist_setting.ip_list'])
    ) {
      updates.push({
        key: 'ip_blacklist_setting.ip_list',
        value: JSON.stringify(ipList),
      })
    }
    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }
    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
    baselineRef.current = {
      'ip_blacklist_setting.enabled': enabled,
      'ip_blacklist_setting.ip_list': ipList,
    }
  }

  return (
    <SettingsSection title={t('IP Blacklist')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save IP blacklist'
          />
          <FormField
            control={form.control}
            name='ip_blacklist_setting.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable IP Blacklist')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Block requests from listed client IP addresses before authentication'
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
            name='ip_blacklist_setting.ip_list'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Blacklisted IPs')}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={t('192.168.1.1&#10;10.0.0.0/8')}
                    rows={6}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('One IP or CIDR range per line')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
