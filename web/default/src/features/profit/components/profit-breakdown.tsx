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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

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

import { formatProfitMoney, formatProfitPercent, getProfitTone } from '../lib'
import type { ProfitAggregate, ProfitOverview } from '../types'

type Dimension = 'user' | 'model' | 'group' | 'channel'

function rowLabel(row: ProfitAggregate, dimension: Dimension): string {
  if (dimension === 'user') return row.username || `#${row.user_id}`
  if (dimension === 'model') return row.model_name || '--'
  if (dimension === 'group') return row.group || '--'
  return row.channel_name || `#${row.channel_id}`
}

export function ProfitBreakdown(props: { overview?: ProfitOverview }) {
  const { t } = useTranslation()
  const [dimension, setDimension] = useState<Dimension>('user')
  const rowsByDimension: Record<Dimension, ProfitAggregate[]> = {
    user: props.overview?.by_user ?? [],
    model: props.overview?.by_model ?? [],
    group: props.overview?.by_group ?? [],
    channel: props.overview?.by_channel ?? [],
  }
  const labels: Record<Dimension, string> = {
    user: t('By user'),
    model: t('By model'),
    group: t('By group'),
    channel: t('By channel'),
  }

  return (
    <div className='border-border border'>
      <div className='border-border flex items-center justify-between border-b px-3 py-2'>
        <h2 className='text-sm font-medium'>{t('Profit breakdown')}</h2>
        <Tabs
          value={dimension}
          onValueChange={(value) => setDimension(value as Dimension)}
        >
          <TabsList>
            {(Object.keys(labels) as Dimension[]).map((key) => (
              <TabsTrigger key={key} value={key}>
                {labels[key]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{labels[dimension]}</TableHead>
            <TableHead className='text-right'>{t('Sales revenue')}</TableHead>
            <TableHead className='text-right'>{t('Purchase cost')}</TableHead>
            <TableHead className='text-right'>{t('Gross profit')}</TableHead>
            <TableHead className='text-right'>{t('Profit margin')}</TableHead>
            <TableHead className='text-right'>{t('Cost coverage')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rowsByDimension[dimension].map((row) => (
            <TableRow key={rowLabel(row, dimension)}>
              <TableCell className='font-medium'>
                {rowLabel(row, dimension)}
              </TableCell>
              <TableCell className='text-right'>
                {formatProfitMoney(row.revenue_micros)}
              </TableCell>
              <TableCell className='text-right'>
                {formatProfitMoney(row.cost_micros)}
              </TableCell>
              <TableCell
                className={cn(
                  'text-right font-medium',
                  getProfitTone(row.profit_micros)
                )}
              >
                {row.unpriced_record_count === row.record_count
                  ? '--'
                  : formatProfitMoney(row.profit_micros)}
              </TableCell>
              <TableCell className='text-right'>
                {formatProfitPercent(row.profit_margin)}
              </TableCell>
              <TableCell className='text-right'>
                {formatProfitPercent(row.cost_coverage)}
              </TableCell>
            </TableRow>
          ))}
          {rowsByDimension[dimension].length === 0 && (
            <TableRow>
              <TableCell
                colSpan={6}
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
