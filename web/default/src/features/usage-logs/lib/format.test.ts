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

import { getTaskBillingDisplay, getTaskBillingStage } from './format'

describe('getTaskBillingStage', () => {
  test('returns null for non-task logs', () => {
    assert.equal(getTaskBillingStage(2, null), null)
    assert.equal(getTaskBillingStage(2, { model_ratio: 1 }), null)
    assert.equal(getTaskBillingStage(6, { reason: 'failed' }), null)
  })

  test('explicit task_billing_stage marker wins', () => {
    assert.equal(
      getTaskBillingStage(2, { is_task: true, task_billing_stage: 'final' }),
      'final'
    )
    assert.equal(
      getTaskBillingStage(6, {
        is_task: true,
        task_billing_stage: 'settle',
        pre_consumed_quota: 100,
      }),
      'settle'
    )
    assert.equal(
      getTaskBillingStage(6, { is_task: true, task_billing_stage: 'refund' }),
      'refund'
    )
  })

  test('legacy settlement logs are inferred from pre_consumed_quota', () => {
    assert.equal(
      getTaskBillingStage(2, {
        is_task: true,
        pre_consumed_quota: 100,
        actual_quota: 150,
      }),
      'settle'
    )
    assert.equal(
      getTaskBillingStage(6, {
        is_task: true,
        pre_consumed_quota: 100,
        actual_quota: 50,
      }),
      'settle'
    )
  })

  test('legacy failure refund logs are inferred as refund', () => {
    assert.equal(
      getTaskBillingStage(6, { is_task: true, reason: 'upstream error' }),
      'refund'
    )
  })

  test('legacy submit-time consume logs are inferred as pre_consume', () => {
    assert.equal(
      getTaskBillingStage(2, { is_task: true, task_id: 'task_x' }),
      'pre_consume'
    )
  })

  test('other log types never resolve a stage', () => {
    assert.equal(getTaskBillingStage(1, { is_task: true }), null)
    assert.equal(getTaskBillingStage(3, { is_task: true }), null)
  })
})

describe('getTaskBillingDisplay', () => {
  const log = {
    type: 2,
    quota: 3000,
    prompt_tokens: 0,
    completion_tokens: 0,
  }

  test('keeps unfinished task final values empty', () => {
    assert.deepEqual(
      getTaskBillingDisplay(log, {
        is_task: true,
        task_billing_stage: 'pre_consume',
      }),
      {
        tokens: null,
        preConsumedQuota: 3000,
        adjustmentQuota: null,
        actualQuota: null,
        noRefund: false,
      }
    )
  })

  test('shows one settled task billing chain', () => {
    assert.deepEqual(
      getTaskBillingDisplay(log, {
        is_task: true,
        task_billing_stage: 'settle',
        pre_consumed_quota: 3000,
        actual_quota: 4200,
        billed_usage: 40594,
      }),
      {
        tokens: 40594,
        preConsumedQuota: 3000,
        adjustmentQuota: 1200,
        actualQuota: 4200,
        noRefund: false,
      }
    )
  })

  test('shows full refund and retained upstream charge distinctly', () => {
    assert.equal(
      getTaskBillingDisplay(log, {
        is_task: true,
        task_billing_stage: 'refund',
        actual_quota: 0,
      })?.adjustmentQuota,
      -3000
    )
    assert.deepEqual(
      getTaskBillingDisplay(log, {
        is_task: true,
        task_billing_stage: 'settle',
        actual_quota: 3000,
        no_refund: true,
      }),
      {
        tokens: null,
        preConsumedQuota: 3000,
        adjustmentQuota: 0,
        actualQuota: 3000,
        noRefund: true,
      }
    )
  })
})
