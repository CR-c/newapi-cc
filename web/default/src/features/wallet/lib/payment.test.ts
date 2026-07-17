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

import { getTopupCredit, mergePresetAmounts } from './payment'

test('mergePresetAmounts attaches an exact-match gift amount to each preset', () => {
  assert.deepEqual(
    mergePresetAmounts([50, 100, 200], { 100: 0.95 }, { 100: 10, 200: 30 }),
    [
      { value: 50, discount: 1, bonus: 0 },
      { value: 100, discount: 0.95, bonus: 10 },
      { value: 200, discount: 1, bonus: 30 },
    ]
  )
})

test('getTopupCredit keeps principal, gift, and total credit separate', () => {
  assert.deepEqual(getTopupCredit(100, { 100: 15 }), {
    principal: 100,
    bonus: 15,
    total: 115,
  })
  assert.deepEqual(getTopupCredit(101, { 100: 15 }), {
    principal: 101,
    bonus: 0,
    total: 101,
  })
})
