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
import type { ProfitAggregate } from './types'

const MICROS_PER_UNIT = 1_000_000

export type NormalizedProfitAggregate = {
  nominalConsumptionMicros: number
  recognizedRevenueMicros: number
  promoConsumptionMicros: number
  promoCostMicros: number
  adminConsumptionMicros: number
  adminCostMicros: number
  legacyUnknownConsumptionMicros: number
}

export function normalizeProfitAggregate(
  aggregate?: ProfitAggregate
): NormalizedProfitAggregate {
  return {
    nominalConsumptionMicros:
      aggregate?.gross_consumption_micros ??
      aggregate?.nominal_consumption_micros ??
      aggregate?.revenue_micros ??
      0,
    recognizedRevenueMicros:
      aggregate?.recognized_revenue_micros ?? aggregate?.revenue_micros ?? 0,
    promoConsumptionMicros: aggregate?.promo_consumption_micros ?? 0,
    promoCostMicros: aggregate?.promo_cost_micros ?? 0,
    adminConsumptionMicros: aggregate?.admin_consumption_micros ?? 0,
    adminCostMicros: aggregate?.admin_cost_micros ?? 0,
    legacyUnknownConsumptionMicros:
      aggregate?.legacy_consumption_micros ??
      aggregate?.legacy_unknown_consumption_micros ??
      0,
  }
}

export function formatProfitMoney(micros: number): string {
  const amount = micros / MICROS_PER_UNIT
  const sign = amount < 0 ? '-' : ''
  const formatted = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(Math.abs(amount))
  return `${sign}¥ ${formatted}`
}

export function formatProfitPercent(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '--'
  return new Intl.NumberFormat(undefined, {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 2,
  }).format(value)
}

export function getProfitTone(micros: number): string {
  if (micros > 0) return 'text-emerald-600 dark:text-emerald-400'
  if (micros < 0) return 'text-rose-600 dark:text-rose-400'
  return 'text-muted-foreground'
}

/** Why a profit row is negative (or unpriced). */
export type LossReasonCode =
  | 'no_cost'
  | 'gift'
  | 'admin'
  | 'gift_and_admin'
  | 'legacy'
  | 'pricing'
  | 'non_paid_drag'
  | 'mixed'
  | 'unknown'

export type LossExplanation = {
  isLoss: boolean
  /** Primary badge / column reason. */
  primary: LossReasonCode | null
  /** All contributing factors, primary first. */
  codes: LossReasonCode[]
  /** Paid revenue still covered its own cost (loss is gift/admin drag). */
  paidCoversOwnCost: boolean
}

/**
 * Classify a negative profit row so operators can tell gift/admin drag
 * apart from real underpricing on paid traffic.
 *
 * Platform profit = paid revenue − all costs (including gift/admin costs).
 * So gift/admin usage with cost and no paid revenue always shows as a loss.
 */
export function explainProfitLoss(
  aggregate?: ProfitAggregate
): LossExplanation {
  if (!aggregate) {
    return {
      isLoss: false,
      primary: null,
      codes: [],
      paidCoversOwnCost: true,
    }
  }

  const normalized = normalizeProfitAggregate(aggregate)
  const allUnpriced =
    aggregate.record_count > 0 &&
    aggregate.unpriced_record_count === aggregate.record_count

  if (allUnpriced) {
    return {
      isLoss: false,
      primary: 'no_cost',
      codes: ['no_cost'],
      paidCoversOwnCost: true,
    }
  }

  if (aggregate.profit_micros >= 0) {
    return {
      isLoss: false,
      primary: null,
      codes: [],
      paidCoversOwnCost: true,
    }
  }

  const paid = normalized.recognizedRevenueMicros
  const promo = normalized.promoConsumptionMicros
  const admin = normalized.adminConsumptionMicros
  const legacy = normalized.legacyUnknownConsumptionMicros
  const promoCost = normalized.promoCostMicros
  const adminCost = normalized.adminCostMicros
  const cost = aggregate.cost_micros
  // Cost left after removing gift/admin cost slices ≈ paid (+ legacy) cost.
  const paidLikeCost = Math.max(0, cost - promoCost - adminCost)
  const paidCoversOwnCost = paidLikeCost <= paid

  if (paid <= 0) {
    if (promo > 0 && admin > 0) {
      return {
        isLoss: true,
        primary: 'gift_and_admin',
        codes: ['gift_and_admin', 'gift', 'admin'],
        paidCoversOwnCost: true,
      }
    }
    if (promo > 0) {
      return {
        isLoss: true,
        primary: 'gift',
        codes: ['gift'],
        paidCoversOwnCost: true,
      }
    }
    if (admin > 0) {
      return {
        isLoss: true,
        primary: 'admin',
        codes: ['admin'],
        paidCoversOwnCost: true,
      }
    }
    if (legacy > 0) {
      return {
        isLoss: true,
        primary: 'legacy',
        codes: ['legacy'],
        paidCoversOwnCost: true,
      }
    }
    return {
      isLoss: true,
      primary: 'unknown',
      codes: ['unknown'],
      paidCoversOwnCost: true,
    }
  }

  // Has paid revenue but overall profit is negative.
  const hasNonPaid = promo > 0 || admin > 0 || legacy > 0
  if (!paidCoversOwnCost && hasNonPaid) {
    const codes: LossReasonCode[] = ['mixed', 'pricing']
    if (promo > 0) codes.push('gift')
    if (admin > 0) codes.push('admin')
    if (legacy > 0) codes.push('legacy')
    return {
      isLoss: true,
      primary: 'mixed',
      codes,
      paidCoversOwnCost: false,
    }
  }
  if (!paidCoversOwnCost) {
    return {
      isLoss: true,
      primary: 'pricing',
      codes: ['pricing'],
      paidCoversOwnCost: false,
    }
  }

  // Paid traffic would cover its cost; gift/admin/legacy dragged total profit negative.
  if (promo > 0 && admin > 0) {
    return {
      isLoss: true,
      primary: 'non_paid_drag',
      codes: ['non_paid_drag', 'gift', 'admin'],
      paidCoversOwnCost: true,
    }
  }
  if (promo > 0) {
    return {
      isLoss: true,
      primary: 'gift',
      codes: ['non_paid_drag', 'gift'],
      paidCoversOwnCost: true,
    }
  }
  if (admin > 0) {
    return {
      isLoss: true,
      primary: 'admin',
      codes: ['non_paid_drag', 'admin'],
      paidCoversOwnCost: true,
    }
  }
  if (legacy > 0) {
    return {
      isLoss: true,
      primary: 'legacy',
      codes: ['non_paid_drag', 'legacy'],
      paidCoversOwnCost: true,
    }
  }
  return {
    isLoss: true,
    primary: 'unknown',
    codes: ['unknown'],
    paidCoversOwnCost: true,
  }
}

/** i18n key (English source string) for a loss reason code. */
export function lossReasonLabelKey(code: LossReasonCode): string {
  switch (code) {
    case 'no_cost':
      return 'No cost rule'
    case 'gift':
      return 'Gift quota'
    case 'admin':
      return 'Admin usage'
    case 'gift_and_admin':
      return 'Gift + admin'
    case 'legacy':
      return 'Unattributed usage'
    case 'pricing':
      return 'Paid underpricing'
    case 'non_paid_drag':
      return 'Gift/admin drag'
    case 'mixed':
      return 'Pricing + gift/admin'
    case 'unknown':
      return 'Unknown loss'
    default:
      return 'Unknown loss'
  }
}

/** Short helper text under the reason badge. */
export function lossReasonHintKey(code: LossReasonCode): string {
  switch (code) {
    case 'no_cost':
      return 'Configure purchase cost for this model'
    case 'gift':
      return 'Loss from gift balance, not paid sales'
    case 'admin':
      return 'Loss from admin/test calls, not paid sales'
    case 'gift_and_admin':
      return 'No paid revenue; gift and admin only'
    case 'legacy':
      return 'Old logs without wallet attribution'
    case 'pricing':
      return 'Paid sales earn less than purchase cost'
    case 'non_paid_drag':
      return 'Paid sales OK; gift/admin cost pulled profit down'
    case 'mixed':
      return 'Paid underpricing plus gift/admin cost'
    case 'unknown':
      return 'Could not classify this loss'
    default:
      return 'Could not classify this loss'
  }
}
