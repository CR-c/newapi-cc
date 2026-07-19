import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { TFunction } from 'i18next'

import {
  getRedemptionFormSchema,
  REDEMPTION_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
  transformRedemptionToFormDefaults,
} from './redemption-form'

const t = ((key: string) => key) as TFunction

describe('redemption funding source form', () => {
  test('new codes default to paid balance', () => {
    assert.equal(REDEMPTION_FORM_DEFAULT_VALUES.funding_source, 'paid')
  })

  test('paid card selection is preserved in the API payload', () => {
    const payload = transformFormDataToPayload({
      name: 'paid card',
      quota_dollars: 10,
      count: 1,
      funding_source: 'paid',
    })

    assert.equal(payload.funding_source, 'paid')
  })

  test('editing keeps the only supported paid source', () => {
    const values = transformRedemptionToFormDefaults({
      id: 1,
      user_id: 1,
      name: 'legacy card',
      key: '00000000000000000000000000000001',
      status: 1,
      quota: 100,
      created_time: 1,
      redeemed_time: 0,
      expired_time: 0,
      used_user_id: 0,
      funding_source: 'paid',
    })

    assert.equal(values.funding_source, 'paid')
  })

  test('schema rejects unsupported funding sources', () => {
    const result = getRedemptionFormSchema(t).safeParse({
      ...REDEMPTION_FORM_DEFAULT_VALUES,
      name: 'invalid source',
      funding_source: 'cash',
    })

    assert.equal(result.success, false)
  })

  test('schema rejects gift and historical sources', () => {
    for (const funding_source of ['promo', 'legacy_unknown']) {
      const result = getRedemptionFormSchema(t).safeParse({
        ...REDEMPTION_FORM_DEFAULT_VALUES,
        name: 'paid source only',
        funding_source,
      })
      assert.equal(result.success, false)
    }
  })
})
