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
import { KeyRound, Route } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import type { ApiInfoItem } from '@/features/dashboard/types'
import { useStatus } from '@/hooks/use-status'

const DEFAULT_API_ROUTES = [
  { route: 'POST /v1/chat/completions', path: '/v1/chat/completions' },
  { route: 'POST /v1/responses', path: '/v1/responses' },
  { route: 'POST /v1/messages', path: '/v1/messages' },
  { route: 'POST /v1/embeddings', path: '/v1/embeddings' },
]

export function ApiAccess() {
  const { t } = useTranslation()
  const { status } = useStatus()

  if (!status) return null

  const configuredApiInfo = Array.isArray(status.api_info)
    ? (status.api_info as ApiInfoItem[]).filter((item) =>
        Boolean(item && typeof item.url === 'string' && item.url.trim())
      )
    : []
  const configuredApiAddress = configuredApiInfo[0]?.url
  const configuredServerAddress =
    typeof status.server_address === 'string'
      ? status.server_address.trim()
      : ''
  const apiAddress = (
    configuredApiAddress?.trim() || configuredServerAddress
  ).replace(/\/+$/, '')
  const apiBase =
    !apiAddress || apiAddress === 'http://localhost:3000'
      ? window.location.origin
      : apiAddress
  const endpointPrefix =
    typeof status.public_api_endpoint === 'string' &&
    status.public_api_endpoint !== '/'
      ? status.public_api_endpoint.replace(/\/+$/, '')
      : ''
  const routes: ApiInfoItem[] = DEFAULT_API_ROUTES.map((item) => ({
    route: item.route,
    url: `${apiBase}${endpointPrefix}${item.path}`,
    description: '',
    color: 'blue',
  }))
  const apiRequiresKey = status.public_api_require_key !== false
  const apiAllowNoModel = status.public_api_allow_no_model === true
  const hasFallback = status.public_api_has_fallback === true

  return (
    <section
      id='api-access'
      aria-labelledby='api-access-title'
      className='border-border/50 border-y px-4 py-12 sm:px-6 md:py-16'
    >
      <div className='mx-auto max-w-6xl'>
        <div className='mb-6 flex flex-wrap items-end justify-between gap-4'>
          <div>
            <h2 id='api-access-title' className='text-2xl font-semibold'>
              {t('API access')}
            </h2>
            <p className='text-muted-foreground mt-2 text-sm'>
              {t('Use the configured API base and compatible routes.')}
            </p>
          </div>
          <div className='flex min-w-0 items-center gap-2 rounded-md border px-3 py-2'>
            <span className='text-muted-foreground text-xs font-medium'>
              {t('API Base')}
            </span>
            <code className='max-w-[min(55vw,32rem)] truncate font-mono text-sm'>
              {apiBase}
            </code>
            <CopyButton
              value={apiBase}
              tooltip={t('Copy API Base')}
              aria-label={t('Copy API Base')}
            />
          </div>
        </div>

        <div className='grid gap-8 lg:grid-cols-[1.1fr_0.9fr]'>
          <div>
            <h3 className='mb-3 flex items-center gap-2 text-sm font-semibold'>
              <Route aria-hidden='true' />
              {t('Compatible endpoints')}
            </h3>
            <ul className='divide-border divide-y border-y'>
              {routes.map((item) => (
                <li
                  key={`${item.route}:${item.url}`}
                  className='flex min-w-0 items-center justify-between gap-3 py-3'
                >
                  <div className='min-w-0'>
                    <code className='block truncate font-mono text-sm'>
                      {item.route}
                    </code>
                    {item.description ? (
                      <span className='text-muted-foreground block truncate text-xs'>
                        {item.description}
                      </span>
                    ) : null}
                  </div>
                  <CopyButton
                    value={item.url}
                    tooltip={t('Copy URL')}
                    aria-label={t('Copy URL')}
                  />
                </li>
              ))}
            </ul>
          </div>

          <div className='border-border/60 flex flex-col gap-5 border-l-0 pl-0 lg:border-l lg:pl-8'>
            <div>
              <h3 className='mb-2 flex items-center gap-2 text-sm font-semibold'>
                <KeyRound aria-hidden='true' />
                {t('API key')}
              </h3>
              <p className='text-muted-foreground text-sm leading-6'>
                {apiRequiresKey
                  ? t('A valid API key is required for requests.')
                  : t('Any API key value or an empty key works for requests.')}
              </p>
            </div>
            <div>
              <h3 className='mb-2 text-sm font-semibold'>{t('Model name')}</h3>
              <p className='text-muted-foreground text-sm leading-6'>
                {apiAllowNoModel
                  ? t(
                      'Similar model names are supported; omit the model name to select automatically.'
                    )
                  : t('A model name is required for requests.')}
              </p>
            </div>
            <div>
              <h3 className='mb-2 text-sm font-semibold'>
                {t('Fallback policy')}
              </h3>
              <p className='text-muted-foreground text-sm leading-6'>
                {hasFallback
                  ? t(
                      'Automatically switches to another model after a model failure.'
                    )
                  : t('No automatic model fallback is configured.')}
              </p>
            </div>
            <div>
              <h3 className='mb-2 text-sm font-semibold'>
                {t('Reasoning fields')}
              </h3>
              <p className='text-muted-foreground text-sm leading-6'>
                {t(
                  'Reasoning output fields depend on the selected model and endpoint. Common fields include reasoning, thinking, and reasoning_content.'
                )}
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
