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
import { describe, test } from 'node:test'

import { attributeLegacyWallet } from './wallet-attribution'

describe('legacy wallet attribution', () => {
  test('adds only the verified legacy portion to paid balance', () => {
    const result = attributeLegacyWallet(
      { paid_quota: 10, promo_quota: 20, legacy_quota: 70 },
      30
    )

    assert.deepEqual(result, {
      paid_quota: 40,
      promo_quota: 60,
      legacy_quota: 0,
    })
  })

  test('allows classifying all legacy balance as gift balance', () => {
    const result = attributeLegacyWallet(
      { paid_quota: 10, promo_quota: 20, legacy_quota: 70 },
      0
    )

    assert.deepEqual(result, {
      paid_quota: 10,
      promo_quota: 90,
      legacy_quota: 0,
    })
  })

  test('rejects amounts outside the legacy balance', () => {
    assert.equal(
      attributeLegacyWallet(
        { paid_quota: 10, promo_quota: 20, legacy_quota: 70 },
        71
      ),
      null
    )
  })
})
