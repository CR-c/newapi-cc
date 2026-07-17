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
import { Gift, Pencil, Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { formatQuotaWithCurrency, getCurrencyDisplay } from '@/lib/currency'

import { safeJsonParseWithValidation } from '../utils/json-parser'
import { isObjectRecord } from '../utils/json-validators'
import { AmountBonusDialog, type AmountBonusData } from './amount-bonus-dialog'

type AmountBonusVisualEditorProps = {
  value: string
  onChange: (value: string) => void
}

export function AmountBonusVisualEditor(props: AmountBonusVisualEditorProps) {
  const { t } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editData, setEditData] = useState<AmountBonusData | null>(null)
  const { meta } = getCurrencyDisplay()
  const formatTierAmount = (amount: number) =>
    meta.kind === 'tokens'
      ? formatQuotaWithCurrency(amount, { abbreviate: false })
      : `$${amount}`

  const tiers = useMemo(() => {
    const parsed = safeJsonParseWithValidation<Record<string, unknown>>(
      props.value,
      {
        fallback: {},
        validator: isObjectRecord,
        validatorMessage: t('Amount gift must be a JSON object'),
        context: 'amount gifts',
      }
    )
    return Object.entries(parsed)
      .map(([amount, bonus]) => ({
        amount: Number.parseInt(amount, 10),
        bonus: Number(bonus),
      }))
      .filter(
        (item) =>
          Number.isFinite(item.amount) &&
          item.amount > 0 &&
          Number.isFinite(item.bonus) &&
          item.bonus > 0
      )
      .sort((a, b) => a.amount - b.amount)
  }, [props.value, t])

  const readValue = () =>
    safeJsonParseWithValidation<Record<string, unknown>>(props.value, {
      fallback: {},
      validator: isObjectRecord,
      silent: true,
    })

  const handleSave = (data: AmountBonusData) => {
    props.onChange(
      JSON.stringify(
        { ...readValue(), [data.amount.toString()]: data.bonus },
        null,
        2
      )
    )
  }

  const handleDelete = (amount: number) => {
    props.onChange(
      JSON.stringify(
        Object.fromEntries(
          Object.entries(readValue()).filter(([key]) => key !== String(amount))
        ),
        null,
        2
      )
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
        <p className='text-muted-foreground text-sm'>
          {t('Configure promotional gifts for exact recharge amounts')}
        </p>
        <Button
          type='button'
          size='sm'
          className='w-full sm:w-auto'
          onClick={() => {
            setEditData(null)
            setDialogOpen(true)
          }}
        >
          <Plus className='h-4 w-4 sm:mr-2' />
          {t('Add gift tier')}
        </Button>
      </div>
      {tiers.length === 0 ? (
        <div className='text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm'>
          {t('No gift tiers configured.')}
        </div>
      ) : (
        <div className='divide-y rounded-md border'>
          {tiers.map((tier) => (
            <div
              key={tier.amount}
              className='flex items-center justify-between gap-3 px-3 py-2.5'
            >
              <div className='flex min-w-0 items-center gap-3'>
                <Gift className='h-4 w-4 shrink-0 text-emerald-600' />
                <div className='min-w-0 text-sm'>
                  <span className='font-medium'>
                    {formatTierAmount(tier.amount)}
                  </span>
                  <span className='text-muted-foreground'> + </span>
                  <span className='font-medium text-emerald-600'>
                    {formatTierAmount(tier.bonus)} {t('gift')}
                  </span>
                  <span className='text-muted-foreground'> · </span>
                  <span>
                    {t('Total')} {formatTierAmount(tier.amount + tier.bonus)}
                  </span>
                </div>
              </div>
              <div className='flex shrink-0'>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Edit')}
                  onClick={() => {
                    setEditData(tier)
                    setDialogOpen(true)
                  }}
                >
                  <Pencil className='h-4 w-4' />
                </Button>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Delete')}
                  onClick={() => handleDelete(tier.amount)}
                >
                  <Trash2 className='h-4 w-4' />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
      <AmountBonusDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSave={handleSave}
        editData={editData}
      />
    </div>
  )
}
