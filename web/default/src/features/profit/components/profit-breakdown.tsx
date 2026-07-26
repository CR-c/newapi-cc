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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

import {
  explainProfitLoss,
  formatProfitMoney,
  formatProfitPercent,
  getProfitTone,
  lossReasonHintKey,
  lossReasonLabelKey,
  normalizeProfitAggregate,
  type LossReasonCode,
} from '../lib'
import type { ProfitAggregate, ProfitOverview } from '../types'

type Dimension = 'model' | 'group' | 'user' | 'channel'

function rowLabel(row: ProfitAggregate, dimension: Dimension): string {
  if (dimension === 'user') return row.username || `#${row.user_id}`
  if (dimension === 'model') return row.model_name || '--'
  if (dimension === 'group') return row.group || '--'
  return row.channel_name || `#${row.channel_id}`
}

function sortByProfit(rows: ProfitAggregate[]): ProfitAggregate[] {
  return [...rows].sort((left, right) => {
    const leftAllUnpriced = left.unpriced_record_count === left.record_count
    const rightAllUnpriced = right.unpriced_record_count === right.record_count
    if (leftAllUnpriced !== rightAllUnpriced) {
      return leftAllUnpriced ? 1 : -1
    }
    if (left.profit_micros !== right.profit_micros) {
      return left.profit_micros - right.profit_micros
    }
    return (right.revenue_micros ?? 0) - (left.revenue_micros ?? 0)
  })
}

function reasonBadgeClass(code: LossReasonCode): string {
  switch (code) {
    case 'gift':
    case 'gift_and_admin':
    case 'non_paid_drag':
      return 'border-amber-500/40 text-amber-700 dark:text-amber-400'
    case 'admin':
      return 'border-sky-500/40 text-sky-700 dark:text-sky-400'
    case 'pricing':
    case 'mixed':
      return 'border-rose-500/40 text-rose-700 dark:text-rose-400'
    case 'legacy':
    case 'no_cost':
    case 'unknown':
      return 'text-muted-foreground'
    default:
      return 'text-muted-foreground'
  }
}

export function ProfitBreakdown(props: { overview?: ProfitOverview }) {
  const { t } = useTranslation()
  const [dimension, setDimension] = useState<Dimension>('model')
  const [showDetails, setShowDetails] = useState(false)

  const labels: Record<Dimension, string> = {
    model: t('By model'),
    group: t('By group'),
    user: t('By user'),
    channel: t('By channel'),
  }

  const rows = useMemo(() => {
    if (dimension === 'model') {
      return sortByProfit(props.overview?.by_model ?? [])
    }
    if (dimension === 'group') {
      return sortByProfit(props.overview?.by_group ?? [])
    }
    if (dimension === 'user') {
      return sortByProfit(props.overview?.by_user ?? [])
    }
    return sortByProfit(props.overview?.by_channel ?? [])
  }, [
    dimension,
    props.overview?.by_channel,
    props.overview?.by_group,
    props.overview?.by_model,
    props.overview?.by_user,
  ])

  const lossStats = useMemo(() => {
    let lossCount = 0
    let giftCount = 0
    let adminCount = 0
    let pricingCount = 0
    for (const row of rows) {
      const explanation = explainProfitLoss(row)
      if (!explanation.isLoss && explanation.primary !== 'no_cost') continue
      if (explanation.isLoss) lossCount += 1
      if (
        explanation.primary === 'gift' ||
        explanation.primary === 'gift_and_admin' ||
        explanation.primary === 'non_paid_drag'
      ) {
        giftCount += 1
      }
      if (
        explanation.primary === 'admin' ||
        explanation.primary === 'gift_and_admin'
      ) {
        adminCount += 1
      }
      if (
        explanation.primary === 'pricing' ||
        explanation.primary === 'mixed'
      ) {
        pricingCount += 1
      }
    }
    return { lossCount, giftCount, adminCount, pricingCount }
  }, [rows])

  return (
    <div className='border-border border'>
      <div className='border-border flex flex-col gap-3 border-b px-3 py-2 sm:flex-row sm:items-center sm:justify-between'>
        <div className='space-y-1'>
          <h2 className='text-sm font-medium'>{t('Profit breakdown')}</h2>
          <p className='text-muted-foreground text-xs'>
            {t('Losses first. Reason column shows gift vs paid underpricing.')}
            {lossStats.lossCount > 0 ? (
              <span className='text-rose-600 dark:text-rose-400'>
                {' '}
                · {t('{{count}} losing rows', { count: lossStats.lossCount })}
              </span>
            ) : null}
            {lossStats.giftCount > 0 ? (
              <span className='text-amber-600 dark:text-amber-400'>
                {' '}
                · {t('{{count}} gift-related', { count: lossStats.giftCount })}
              </span>
            ) : null}
            {lossStats.pricingCount > 0 ? (
              <span className='text-rose-600 dark:text-rose-400'>
                {' '}
                ·{' '}
                {t('{{count}} paid underpricing', {
                  count: lossStats.pricingCount,
                })}
              </span>
            ) : null}
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-3'>
          <label className='text-muted-foreground flex items-center gap-2 text-xs'>
            <Switch
              checked={showDetails}
              onCheckedChange={setShowDetails}
              aria-label={t('Show details')}
            />
            {t('Show details')}
          </label>
          <Tabs
            value={dimension}
            onValueChange={(value) => setDimension(value as Dimension)}
          >
            <TabsList className='w-max'>
              {(Object.keys(labels) as Dimension[]).map((key) => (
                <TabsTrigger key={key} value={key}>
                  {labels[key]}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{labels[dimension]}</TableHead>
            <TableHead>{t('Loss reason')}</TableHead>
            <TableHead className='text-right'>{t('Revenue')}</TableHead>
            <TableHead className='text-right'>{t('Purchase cost')}</TableHead>
            <TableHead className='text-right'>{t('Gross profit')}</TableHead>
            <TableHead className='text-right'>{t('Profit margin')}</TableHead>
            {showDetails ? (
              <>
                <TableHead className='text-right'>
                  {t('Gift consumption')}
                </TableHead>
                <TableHead className='text-right'>
                  {t('Admin consumption')}
                </TableHead>
                <TableHead className='text-right'>
                  {t('Nominal consumption')}
                </TableHead>
              </>
            ) : null}
            <TableHead className='text-right'>{t('Records')}</TableHead>
            <TableHead className='text-right'>{t('Cost coverage')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => {
            const attributed = normalizeProfitAggregate(row)
            const explanation = explainProfitLoss(row)
            const allUnpriced =
              row.record_count > 0 &&
              row.unpriced_record_count === row.record_count
            const isLoss = explanation.isLoss
            const reasonCode = explanation.primary
            return (
              <TableRow
                key={rowLabel(row, dimension)}
                className={cn(
                  isLoss &&
                    (reasonCode === 'pricing' || reasonCode === 'mixed') &&
                    'bg-rose-500/5',
                  isLoss &&
                    (reasonCode === 'gift' ||
                      reasonCode === 'gift_and_admin' ||
                      reasonCode === 'non_paid_drag') &&
                    'bg-amber-500/5',
                  isLoss && reasonCode === 'admin' && 'bg-sky-500/5'
                )}
              >
                <TableCell className='font-medium'>
                  <span className='truncate'>{rowLabel(row, dimension)}</span>
                </TableCell>
                <TableCell className='min-w-44'>
                  {reasonCode ? (
                    <div className='space-y-1'>
                      <Badge
                        variant='outline'
                        className={cn(
                          'max-w-full font-medium',
                          reasonBadgeClass(reasonCode)
                        )}
                        title={t(lossReasonHintKey(reasonCode))}
                      >
                        {t(lossReasonLabelKey(reasonCode))}
                      </Badge>
                      <p className='text-muted-foreground text-xs leading-snug'>
                        {t(lossReasonHintKey(reasonCode))}
                      </p>
                      {showDetails && explanation.codes.length > 1 ? (
                        <div className='flex flex-wrap gap-1'>
                          {explanation.codes
                            .filter((code) => code !== reasonCode)
                            .map((code) => (
                              <Badge
                                key={`${rowLabel(row, dimension)}-${code}`}
                                variant='outline'
                                className={cn(
                                  'text-[10px]',
                                  reasonBadgeClass(code)
                                )}
                              >
                                {t(lossReasonLabelKey(code))}
                              </Badge>
                            ))}
                        </div>
                      ) : null}
                    </div>
                  ) : (
                    <span className='text-muted-foreground text-xs'>
                      {t('Profitable')}
                    </span>
                  )}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatProfitMoney(attributed.recognizedRevenueMicros)}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {allUnpriced ? '--' : formatProfitMoney(row.cost_micros)}
                </TableCell>
                <TableCell
                  className={cn(
                    'text-right font-medium tabular-nums',
                    allUnpriced ? '' : getProfitTone(row.profit_micros)
                  )}
                >
                  {allUnpriced ? '--' : formatProfitMoney(row.profit_micros)}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatProfitPercent(row.profit_margin)}
                </TableCell>
                {showDetails ? (
                  <>
                    <TableCell className='text-right text-amber-600 tabular-nums dark:text-amber-400'>
                      {formatProfitMoney(attributed.promoConsumptionMicros)}
                    </TableCell>
                    <TableCell className='text-right text-sky-600 tabular-nums dark:text-sky-400'>
                      {formatProfitMoney(attributed.adminConsumptionMicros)}
                    </TableCell>
                    <TableCell className='text-right tabular-nums'>
                      {formatProfitMoney(attributed.nominalConsumptionMicros)}
                    </TableCell>
                  </>
                ) : null}
                <TableCell className='text-right tabular-nums'>
                  {row.record_count}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatProfitPercent(row.cost_coverage)}
                </TableCell>
              </TableRow>
            )
          })}
          {rows.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={showDetails ? 11 : 8}
                className='text-muted-foreground h-24 text-center'
              >
                {t('No profit data')}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
