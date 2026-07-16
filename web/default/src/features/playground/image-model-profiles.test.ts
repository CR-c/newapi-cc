import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getImageModelProfile,
  getImagePresetSize,
  isValidExactImageSize,
} from './image-model-profiles'

describe('64 image model profiles', () => {
  test('nano-banana-3 exposes fourteen aspect ratios at fixed 2K', () => {
    const profile = getImageModelProfile(
      '64生图',
      'gemini-3.1-flash-image-preview'
    )

    assert.equal(profile.provider, 'nano-banana')
    assert.equal(profile.fixedResolution, '2K')
    assert.equal(profile.aspectRatios.length, 14)
    for (const ratio of ['1:4', '1:8', '4:1', '8:1']) {
      assert.ok(profile.aspectRatios.includes(ratio))
    }
    assert.deepEqual(profile.qualities, [])
    assert.equal(profile.maxReferences, 14)
  })

  test('nano-banana-2 exposes the standard ten aspect ratios', () => {
    const profile = getImageModelProfile('64生图', 'gemini-3-pro-image-preview')

    assert.equal(profile.provider, 'nano-banana')
    assert.equal(profile.fixedResolution, '2K')
    assert.equal(profile.aspectRatios.length, 10)
    assert.equal(profile.aspectRatios.includes('1:4'), false)
  })

  test('gpt-image-2 exposes documented resolution and parameter controls', () => {
    const profile = getImageModelProfile('64生图', 'gpt-image-2')

    assert.equal(profile.provider, 'gpt-image')
    assert.deepEqual(profile.resolutions, ['1K', '2K', '4K'])
    assert.deepEqual(profile.qualities, ['low', 'medium', 'high'])
    assert.deepEqual(profile.backgrounds, ['opaque', 'transparent'])
    assert.equal(profile.supportsAutoSize, true)
    assert.equal(profile.supportsExactSize, true)
    assert.equal(getImagePresetSize(profile, '1K', '1:1'), '1024x1024')
    assert.equal(getImagePresetSize(profile, '2K', '16:9'), '2560x1440')
    assert.equal(getImagePresetSize(profile, '4K', '9:16'), '1872x3328')
  })

  test('exact gpt-image-2 sizes require positive multiples of sixteen', () => {
    assert.equal(isValidExactImageSize('2160', '3840'), true)
    assert.equal(isValidExactImageSize('2159', '3840'), false)
    assert.equal(isValidExactImageSize('0', '3840'), false)
  })
})
