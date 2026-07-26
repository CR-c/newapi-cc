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
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'

import {
  explainProfitLoss,
  formatProfitMoney,
  formatProfitPercent,
  getProfitTone,
  lossReasonHintKey,
  lossReasonLabelKey,
  normalizeProfitAggregate,
} from '../lib'
import type { ProfitAggregate } from '../types'

export function ProfitSummary(props: { summary?: ProfitAggregate }) {
  const { t } = useTranslation()
  const [detailsOpen, setDetailsOpen] = useState(false)
  const summary = props.summary
  const attributed = normalizeProfitAggregate(summary)
  const hasPricedRecords =
    (summary?.record_count ?? 0) > (summary?.unpriced_record_count ?? 0)
  const profitMicros = hasPricedRecords ? (summary?.profit_micros ?? 0) : 0
  const costMicros = hasPricedRecords ? (summary?.cost_micros ?? 0) : 0
  const unpriced = summary?.unpriced_record_count ?? 0
  const lossExplanation = explainProfitLoss(summary)

  return (
    <div className='space-y-3'>
      <div className='grid gap-3 lg:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,1fr))]'>
        <Card className='border-border bg-card'>
          <CardHeader className='pb-2'>
            <div className='flex items-center justify-between gap-2'>
              <CardTitle className='text-muted-foreground text-sm font-medium'>
                {t('Gross profit')}
              </CardTitle>
              {hasPricedRecords ? (
                <Badge
                  variant='outline'
                  className={cn('tabular-nums', getProfitTone(profitMicros))}
                >
                  {formatProfitPercent(summary?.profit_margin ?? null)}
                </Badge>
              ) : null}
            </div>
          </CardHeader>
          <CardContent>
            <div
              className={cn(
                'text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl',
                hasPricedRecords
                  ? getProfitTone(profitMicros)
                  : 'text-muted-foreground'
              )}
            >
              {hasPricedRecords ? formatProfitMoney(profitMicros) : '--'}
            </div>
            <p className='text-muted-foreground mt-2 text-xs'>
              {t('Paid revenue minus purchase cost')}
            </p>
            {lossExplanation.isLoss && lossExplanation.primary ? (
              <div className='mt-3 space-y-1'>
                <Badge variant='outline' className='text-rose-600'>
                  {t(lossReasonLabelKey(lossExplanation.primary))}
                </Badge>
                <p className='text-muted-foreground text-xs'>
                  {t(lossReasonHintKey(lossExplanation.primary))}
                </p>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <MetricCard
          label={t('Recognized revenue')}
          value={formatProfitMoney(attributed.recognizedRevenueMicros)}
          hint={t('Paid wallet only')}
        />
        <MetricCard
          label={t('Purchase cost')}
          value={hasPricedRecords ? formatProfitMoney(costMicros) : '--'}
          hint={t('From cost rules')}
        />
        <MetricCard
          label={t('Nominal consumption')}
          value={formatProfitMoney(attributed.nominalConsumptionMicros)}
          hint={t('All billed usage')}
        />
      </div>

      <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen}>
        <div className='border-border bg-muted/20 border px-3 py-2'>
          <CollapsibleTrigger className='flex w-full items-center justify-between gap-2 text-left text-sm'>
            <span className='font-medium'>{t('Consumption details')}</span>
            <span className='text-muted-foreground flex items-center gap-2 text-xs'>
              {t('Gift / admin / coverage')}
              <ChevronDown
                className={cn(
                  'h-4 w-4 transition-transform',
                  detailsOpen && 'rotate-180'
                )}
              />
            </span>
          </CollapsibleTrigger>
          <CollapsibleContent className='mt-3 space-y-3'>
            <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
              <DetailStat
                label={t('Gift consumption')}
                value={formatProfitMoney(attributed.promoConsumptionMicros)}
                tone='text-amber-600 dark:text-amber-400'
              />
              <DetailStat
                label={t('Gift cost')}
                value={
                  summary && (summary.promo_unpriced_record_count ?? 0) === 0
                    ? formatProfitMoney(attributed.promoCostMicros)
                    : '--'
                }
                tone='text-rose-600 dark:text-rose-400'
              />
              <DetailStat
                label={t('Admin consumption')}
                value={formatProfitMoney(attributed.adminConsumptionMicros)}
                tone='text-amber-600 dark:text-amber-400'
              />
              <DetailStat
                label={t('Admin cost')}
                value={
                  summary && (summary.admin_unpriced_record_count ?? 0) === 0
                    ? formatProfitMoney(attributed.adminCostMicros)
                    : '--'
                }
                tone='text-rose-600 dark:text-rose-400'
              />
              <DetailStat
                label={t('Legacy unattributed consumption')}
                value={formatProfitMoney(
                  attributed.legacyUnknownConsumptionMicros
                )}
                tone='text-muted-foreground'
              />
              <div className='border-border space-y-2 border px-3 py-2 sm:col-span-2 lg:col-span-3'>
                <div className='flex items-center justify-between text-xs'>
                  <span className='text-muted-foreground'>
                    {t('Cost coverage')}
                  </span>
                  <span className='font-medium tabular-nums'>
                    {formatProfitPercent(summary?.cost_coverage ?? 0)}
                  </span>
                </div>
                <Progress
                  value={(summary?.cost_coverage ?? 0) * 100}
                  className='h-1.5'
                />
                {unpriced > 0 ? (
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      '{{count}} unpriced billing records are excluded from profit margin.',
                      { count: unpriced }
                    )}
                  </p>
                ) : null}
              </div>
            </div>
          </CollapsibleContent>
        </div>
      </Collapsible>
    </div>
  )
}

function MetricCard(props: { label: string; value: string; hint: string }) {
  return (
    <Card size='sm'>
      <CardHeader className='pb-1'>
        <CardTitle className='text-muted-foreground text-xs font-medium'>
          {props.label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className='text-xl font-semibold tabular-nums'>{props.value}</div>
        <p className='text-muted-foreground mt-1 text-xs'>{props.hint}</p>
      </CardContent>
    </Card>
  )
}

function DetailStat(props: {
  label: string
  value: string
  tone?: string
}) {
  return (
    <div className='border-border space-y-1 border px-3 py-2'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className={cn('text-sm font-semibold tabular-nums', props.tone)}>
        {props.value}
      </div>
    </div>
  )
}
