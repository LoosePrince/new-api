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
import { ArrowRight, HeartHandshake, ShieldCheck, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

interface PromoProps {
  isAuthenticated?: boolean
}

export function Promo(props: PromoProps) {
  const { t } = useTranslation()

  return (
    <section className='border-border/40 relative overflow-hidden border-b px-6 pt-28 pb-16 md:pt-36 md:pb-24'>
      <div className='absolute inset-0 -z-10 bg-[linear-gradient(120deg,oklch(0.98_0.02_80),transparent_48%,oklch(0.96_0.03_190))] dark:bg-[linear-gradient(120deg,oklch(0.2_0.03_80),transparent_48%,oklch(0.17_0.03_190))]' />
      <div className='mx-auto grid max-w-6xl gap-12 lg:grid-cols-[1.1fr_0.9fr] lg:items-center'>
        <div className='max-w-2xl'>
          <div className='text-muted-foreground mb-5 flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase'>
            <span
              className='size-2 rounded-full bg-emerald-500'
              aria-hidden='true'
            />
            {t('Free access · Service online')}
          </div>
          <h1 className='text-4xl leading-[1.08] font-semibold tracking-tight md:text-6xl'>
            {t('One address,')}
            <br />
            <span className='text-muted-foreground'>
              {t('connect your AI application.')}
            </span>
          </h1>
          <p className='text-muted-foreground mt-6 max-w-xl text-base leading-7 md:text-lg'>
            {t(
              'A free public AI gateway for公益 access, automatic model routing, and a unified API endpoint.'
            )}
          </p>
          <div className='mt-8 flex flex-wrap items-center gap-3'>
            <Button
              type='button'
              className='h-10 rounded-lg px-4'
              onClick={() =>
                document
                  .querySelector('#public-chat')
                  ?.scrollIntoView({ behavior: 'smooth' })
              }
            >
              <Sparkles className='size-4' aria-hidden='true' />
              {t('Try online now')}
              <ArrowRight className='size-4' aria-hidden='true' />
            </Button>
            <Button
              type='button'
              variant='outline'
              className='h-10 rounded-lg px-4'
              onClick={() =>
                document
                  .querySelector('#public-stats')
                  ?.scrollIntoView({ behavior: 'smooth' })
              }
            >
              {t('View public data')}
            </Button>
          </div>
          {props.isAuthenticated ? (
            <p className='text-muted-foreground mt-5 text-xs'>
              {t(
                'You are signed in. Your dashboard remains available from the top navigation.'
              )}
            </p>
          ) : null}
        </div>

        <div className='border-border/50 bg-background/70 grid gap-px overflow-hidden rounded-xl border shadow-sm'>
          <div className='bg-background/80 p-6'>
            <HeartHandshake
              className='size-6 text-emerald-600 dark:text-emerald-400'
              aria-hidden='true'
            />
            <h2 className='mt-5 text-lg font-semibold'>
              {t('Built for public benefit')}
            </h2>
            <p className='text-muted-foreground mt-2 text-sm leading-6'>
              {t(
                'No API key, no model selection, and no paid plan are required for the configured public experience.'
              )}
            </p>
          </div>
          <div className='bg-background/80 flex items-center gap-3 p-6'>
            <ShieldCheck
              className='text-muted-foreground size-5'
              aria-hidden='true'
            />
            <span className='text-muted-foreground text-sm'>
              {t('Requests are routed by the site operator.')}
            </span>
          </div>
        </div>
      </div>
    </section>
  )
}
