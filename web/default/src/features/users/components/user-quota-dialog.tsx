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
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { adjustUserQuota } from '../api'
import { allocateWalletBalance } from '../lib'
import type { QuotaAdjustMode, QuotaFundingSource } from '../types'

interface UserQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
  currentQuota: number
  currentPaidQuota: number
  currentPromoQuota: number
  currentLegacyQuota: number
  canAttribute: boolean
  canCreditPaid: boolean
  onSuccess: () => void
}

type QuotaDialogMode = QuotaAdjustMode | 'attribute'

const QUOTA_MODE_LABELS: Record<QuotaDialogMode, string> = {
  add: 'Add',
  subtract: 'Subtract',
  override: 'Override',
  attribute: 'Set paid balance',
}

export function UserQuotaDialog(props: UserQuotaDialogProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<QuotaDialogMode>('add')
  const [amount, setAmount] = useState('')
  const [fundingSource, setFundingSource] =
    useState<QuotaFundingSource>('promo')
  const [reason, setReason] = useState('')
  const [loading, setLoading] = useState(false)
  const creditFundingSource = props.canCreditPaid ? fundingSource : 'promo'

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'

  const amountValue = Number.parseFloat(amount) || 0
  const quotaValue = parseQuotaFromDollars(Math.abs(amountValue))
  const currentWallet = {
    paid_quota: props.currentPaidQuota,
    promo_quota: props.currentPromoQuota,
    legacy_quota: props.currentLegacyQuota,
  }
  const finalPaidQuota = parseQuotaFromDollars(amountValue)
  const targetWallet = allocateWalletBalance(currentWallet, finalPaidQuota)
  const reasonLength = [...reason.trim()].length
  let attributionError = ''
  if (mode === 'attribute') {
    if (!amount.trim() || !Number.isFinite(Number.parseFloat(amount))) {
      attributionError = t('Enter the final paid balance.')
    } else if (!targetWallet) {
      attributionError = t(
        'Final paid balance must be between 0 and {{amount}}.',
        {
          amount: formatQuota(props.currentQuota),
        }
      )
    } else if (
      targetWallet.paid_quota === props.currentPaidQuota &&
      targetWallet.promo_quota === props.currentPromoQuota &&
      props.currentLegacyQuota === 0
    ) {
      attributionError = t('The balance allocation is unchanged.')
    } else if (reasonLength < 3 || reasonLength > 200) {
      attributionError = t('Reason must be between 3 and 200 characters.')
    }
  }

  const getPreviewText = () => {
    const current = props.currentQuota
    const val = quotaValue
    switch (mode) {
      case 'add':
        return `${t('Current quota')}: ${formatQuota(current)}  +${formatQuota(val)} = ${formatQuota(current + val)}`
      case 'subtract':
        return `${t('Current quota')}: ${formatQuota(current)}  -${formatQuota(val)} = ${formatQuota(current - val)}`
      case 'override': {
        const overrideQuota = parseQuotaFromDollars(amountValue)
        return `${t('Current quota')}: ${formatQuota(current)} → ${formatQuota(overrideQuota)}`
      }
      case 'attribute': {
        if (!targetWallet) return ''
        return `${t('Paid balance')}: ${formatQuota(targetWallet.paid_quota)}  ·  ${t('Gift balance')}: ${formatQuota(targetWallet.promo_quota)}`
      }
      default:
        return ''
    }
  }

  const handleConfirm = async () => {
    if (mode === 'attribute') {
      if (!targetWallet || attributionError) return
      setLoading(true)
      try {
        const result = await adjustUserQuota({
          id: props.userId,
          action: 'reclassify_wallet',
          expected_wallet: {
            paid_quota: props.currentPaidQuota,
            promo_quota: props.currentPromoQuota,
            legacy_quota: props.currentLegacyQuota,
          },
          target_wallet: targetWallet,
          reason: reason.trim(),
        })
        if (result.success) {
          toast.success(t('Balance attribution updated'))
          setAmount('')
          setReason('')
          setMode('add')
          props.onOpenChange(false)
          props.onSuccess()
        } else {
          toast.error(result.message || t('Failed to adjust quota'))
        }
      } catch (e: unknown) {
        toast.error(
          e instanceof Error ? e.message : t('Failed to adjust quota')
        )
      } finally {
        setLoading(false)
      }
      return
    }
    if (!amount && mode !== 'override') return
    if (quotaValue <= 0 && mode !== 'override') return

    setLoading(true)
    try {
      const value =
        mode === 'override' ? parseQuotaFromDollars(amountValue) : quotaValue
      const result = await adjustUserQuota({
        id: props.userId,
        action: 'add_quota',
        mode,
        value: mode === 'override' ? value : Math.abs(value),
        funding_source: mode === 'add' ? creditFundingSource : undefined,
      })
      if (result.success) {
        toast.success(t('Quota adjusted successfully'))
        setAmount('')
        setMode('add')
        setFundingSource('promo')
        props.onOpenChange(false)
        props.onSuccess()
      } else {
        toast.error(result.message || t('Failed to adjust quota'))
      }
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('Failed to adjust quota'))
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    setAmount('')
    setMode('add')
    setFundingSource('promo')
    setReason('')
    props.onOpenChange(false)
  }

  const placeholder = tokensOnly
    ? t('Enter amount in tokens')
    : t('Enter amount in {{currency}}', { currency: currencyLabel })

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!open) {
          setAmount('')
          setMode('add')
          setFundingSource('promo')
          setReason('')
        }
        props.onOpenChange(open)
      }}
      title={t('Adjust Quota')}
      description={t('Select an operation mode and enter the amount')}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={handleCancel}>
            {t('Cancel')}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={loading || (mode === 'attribute' && !!attributionError)}
          >
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <div className='text-muted-foreground text-sm'>{getPreviewText()}</div>

        <div className='space-y-2'>
          <Label>{t('Mode')}</Label>
          <div className='grid grid-cols-2 gap-1 sm:grid-cols-4'>
            {(props.canAttribute
              ? (['add', 'subtract', 'override', 'attribute'] as const)
              : (['add', 'subtract', 'override'] as const)
            ).map((m) => (
              <Button
                key={m}
                type='button'
                variant='outline'
                size='sm'
                className={cn(
                  mode === m &&
                    'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
                )}
                onClick={() => {
                  setMode(m)
                  setAmount(
                    m === 'attribute'
                      ? String(quotaUnitsToDollars(props.currentPaidQuota))
                      : ''
                  )
                }}
              >
                {t(QUOTA_MODE_LABELS[m])}
              </Button>
            ))}
          </div>
        </div>

        {mode === 'add' && props.canCreditPaid && (
          <div className='space-y-2'>
            <Label>{t('Balance source')}</Label>
            <ToggleGroup
              value={[fundingSource]}
              onValueChange={(value) => {
                const nextSource = value.find((item) => item !== fundingSource)
                if (nextSource === 'paid' || nextSource === 'promo') {
                  setFundingSource(nextSource)
                }
              }}
              variant='outline'
              spacing={2}
              className='grid w-full grid-cols-2 gap-2'
              aria-label={t('Balance source')}
            >
              <ToggleGroupItem value='paid' className='w-full'>
                {t('Paid balance')}
              </ToggleGroupItem>
              <ToggleGroupItem value='promo' className='w-full'>
                {t('Gift balance')}
              </ToggleGroupItem>
            </ToggleGroup>
            <p className='text-muted-foreground text-xs'>
              {fundingSource === 'promo'
                ? t('Gift balance produces no recognized revenue when spent.')
                : t(
                    'Use paid balance only after confirming payment was received.'
                  )}
            </p>
          </div>
        )}

        {mode === 'add' && !props.canCreditPaid && (
          <div className='space-y-2'>
            <Label>{t('Balance source')}</Label>
            <div className='bg-muted/30 border px-3 py-2 text-sm font-medium'>
              {t('Gift balance')}
            </div>
            <p className='text-muted-foreground text-xs'>
              {t('Gift balance produces no recognized revenue when spent.')}
            </p>
          </div>
        )}

        {mode === 'attribute' && (
          <div className='space-y-3'>
            <div className='bg-muted/30 grid grid-cols-1 gap-2 border px-3 py-2 text-xs sm:grid-cols-3'>
              <div>
                <span className='text-muted-foreground block'>
                  {t('Paid balance')}
                </span>
                <span className='font-medium'>
                  {formatQuota(props.currentPaidQuota)}
                </span>
              </div>
              <div>
                <span className='text-muted-foreground block'>
                  {t('Gift balance')}
                </span>
                <span className='font-medium'>
                  {formatQuota(props.currentPromoQuota)}
                </span>
              </div>
              <div>
                <span className='text-muted-foreground block'>
                  {t('Legacy unattributed')}
                </span>
                <span className='font-medium'>
                  {formatQuota(props.currentLegacyQuota)}
                </span>
              </div>
            </div>
            <p className='text-muted-foreground text-xs'>
              {t(
                'The remaining balance becomes gift balance and legacy attribution is cleared.'
              )}
            </p>
          </div>
        )}

        <div className='space-y-2'>
          <Label>
            {mode === 'attribute' ? t('Final paid balance') : t('Amount')} (
            {currencyLabel})
          </Label>
          <Input
            type='number'
            step={tokensOnly ? 1 : 0.000001}
            min={mode === 'override' ? undefined : 0}
            placeholder={placeholder}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleConfirm()
            }}
          />
          {mode === 'attribute' && attributionError && (
            <p className='text-destructive text-xs'>{attributionError}</p>
          )}
        </div>

        {mode === 'attribute' && (
          <div className='space-y-2'>
            <Label>{t('Attribution reason')}</Label>
            <Input
              value={reason}
              maxLength={200}
              placeholder={t('Enter attribution reason')}
              onChange={(event) => setReason(event.target.value)}
            />
          </div>
        )}
      </div>
    </Dialog>
  )
}
