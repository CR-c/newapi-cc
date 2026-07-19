import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { TFunction } from 'i18next'

import {
  getRedemptionFormSchema,
  isNewRedemptionFundingSource,
  REDEMPTION_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
  transformRedemptionToFormDefaults,
} from './redemption-form'

const t = ((key: string) => key) as TFunction

describe('redemption funding source form', () => {
  test('new codes default to promotional gift balance', () => {
    assert.equal(REDEMPTION_FORM_DEFAULT_VALUES.funding_source, 'promo')
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

  test('historical unknown source is preserved when editing old codes', () => {
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
      funding_source: 'legacy_unknown',
    })

    assert.equal(values.funding_source, 'legacy_unknown')
  })

  test('schema rejects unsupported funding sources', () => {
    const result = getRedemptionFormSchema(t).safeParse({
      ...REDEMPTION_FORM_DEFAULT_VALUES,
      name: 'invalid source',
      funding_source: 'cash',
    })

    assert.equal(result.success, false)
  })

  test('new redemption requests exclude historical unknown source', () => {
    assert.equal(isNewRedemptionFundingSource('paid'), true)
    assert.equal(isNewRedemptionFundingSource('promo'), true)
    assert.equal(isNewRedemptionFundingSource('legacy_unknown'), false)
  })
})
