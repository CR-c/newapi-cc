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
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'

import { formatProfitMoney, formatProfitPercent, getProfitTone } from '../lib'
import type { ProfitAggregate } from '../types'

export function ProfitSummary(props: { summary?: ProfitAggregate }) {
  const { t } = useTranslation()
  const summary = props.summary
  const hasPricedRecords =
    (summary?.record_count ?? 0) > (summary?.unpriced_record_count ?? 0)
  const metrics = [
    {
      label: t('Sales revenue'),
      value: formatProfitMoney(summary?.revenue_micros ?? 0),
      tone: '',
    },
    {
      label: t('Purchase cost'),
      value: hasPricedRecords
        ? formatProfitMoney(summary?.cost_micros ?? 0)
        : '--',
      tone: '',
    },
    {
      label: t('Gross profit'),
      value: hasPricedRecords
        ? formatProfitMoney(summary?.profit_micros ?? 0)
        : '--',
      tone: getProfitTone(summary?.profit_micros ?? 0),
    },
    {
      label: t('Profit margin'),
      value: hasPricedRecords
        ? formatProfitPercent(summary?.profit_margin ?? null)
        : '--',
      tone: getProfitTone(summary?.profit_micros ?? 0),
    },
  ]

  return (
    <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      {metrics.map((metric) => (
        <Card key={metric.label} size='sm'>
          <CardHeader>
            <CardTitle className='text-muted-foreground text-xs font-medium'>
              {metric.label}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className={cn('text-xl font-semibold tabular-nums', metric.tone)}
            >
              {metric.value}
            </div>
          </CardContent>
        </Card>
      ))}
      <div className='border-border bg-muted/20 col-span-full space-y-2 border px-4 py-3'>
        <div className='flex items-center justify-between text-xs'>
          <span className='text-muted-foreground'>{t('Cost coverage')}</span>
          <span className='font-medium tabular-nums'>
            {formatProfitPercent(summary?.cost_coverage ?? 0)}
          </span>
        </div>
        <Progress
          value={(summary?.cost_coverage ?? 0) * 100}
          className='h-1.5'
        />
        {(summary?.unpriced_record_count ?? 0) > 0 && (
          <p className='text-muted-foreground text-xs'>
            {t(
              '{{count}} unpriced billing records are excluded from profit margin.',
              {
                count: summary?.unpriced_record_count ?? 0,
              }
            )}
          </p>
        )}
      </div>
    </div>
  )
}
