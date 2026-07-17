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
import assert from 'node:assert/strict'
import test from 'node:test'

import { formatProfitMoney, normalizeProfitAggregate } from './lib'

test('formatProfitMoney displays positive and negative amounts as CNY', () => {
  assert.match(formatProfitMoney(1_000_000), /^¥\s*1/)
  assert.match(formatProfitMoney(-1_000_000), /^-¥\s*1/)
})

test('normalizeProfitAggregate exposes wallet attribution metrics', () => {
  const normalized = normalizeProfitAggregate({
    revenue_micros: 1_000_000,
    known_revenue_micros: 900_000,
    cost_micros: 600_000,
    profit_micros: 300_000,
    record_count: 3,
    unpriced_record_count: 0,
    profit_margin: 1 / 3,
    cost_coverage: 1,
    nominal_consumption_micros: 1_500_000,
    recognized_revenue_micros: 900_000,
    promo_consumption_micros: 300_000,
    promo_cost_micros: 120_000,
    admin_consumption_micros: 200_000,
    admin_cost_micros: 80_000,
    legacy_unknown_consumption_micros: 100_000,
  })

  assert.equal(normalized.nominalConsumptionMicros, 1_500_000)
  assert.equal(normalized.recognizedRevenueMicros, 900_000)
  assert.equal(normalized.promoConsumptionMicros, 300_000)
  assert.equal(normalized.promoCostMicros, 120_000)
  assert.equal(normalized.adminConsumptionMicros, 200_000)
  assert.equal(normalized.adminCostMicros, 80_000)
  assert.equal(normalized.legacyUnknownConsumptionMicros, 100_000)
})

test('normalizeProfitAggregate falls back to legacy revenue fields', () => {
  const normalized = normalizeProfitAggregate({
    revenue_micros: 1_000_000,
    known_revenue_micros: 800_000,
    cost_micros: 400_000,
    profit_micros: 400_000,
    record_count: 2,
    unpriced_record_count: 0,
    profit_margin: 0.5,
    cost_coverage: 1,
  })

  assert.equal(normalized.nominalConsumptionMicros, 1_000_000)
  assert.equal(normalized.recognizedRevenueMicros, 1_000_000)
  assert.equal(normalized.promoConsumptionMicros, 0)
  assert.equal(normalized.adminConsumptionMicros, 0)
  assert.equal(normalized.legacyUnknownConsumptionMicros, 0)
})
