import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getImageModelProfile,
  getImagePresetSize,
  isValidExactImageSize,
} from './image-model-profiles'

describe('image model profiles', () => {
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
    assert.equal(profile.maxReferences, 14)
  })

  test('gemini image models keep reference support outside 64生图 group', () => {
    const profile = getImageModelProfile(
      'default',
      'gemini-2.5-flash-image'
    )
    assert.equal(profile.provider, 'nano-banana')
    assert.equal(profile.maxReferences, 14)
  })

  test('gpt-image-2 exposes documented resolution and parameter controls', () => {
    const profile = getImageModelProfile('64生图', 'gpt-image-2')

    assert.equal(profile.provider, 'gpt-image')
    assert.deepEqual(profile.resolutions, ['1K', '2K', '4K'])
    assert.deepEqual(profile.qualities, ['low', 'medium', 'high'])
    assert.deepEqual(profile.backgrounds, ['opaque', 'transparent'])
    assert.equal(profile.supportsAutoSize, true)
    assert.equal(profile.supportsExactSize, true)
    assert.equal(profile.maxReferences, 14)
    assert.equal(getImagePresetSize(profile, '1K', '1:1'), '1024x1024')
    assert.equal(getImagePresetSize(profile, '2K', '16:9'), '2560x1440')
    assert.equal(getImagePresetSize(profile, '4K', '9:16'), '1872x3328')
  })

  test('ic生图 gpt-image-2-ic exposes 1k and super-res 4k controls', () => {
    const byModel = getImageModelProfile('ic生图', 'gpt-image-2-ic')
    assert.equal(byModel.provider, 'gpt-image')
    assert.deepEqual(byModel.resolutions, ['1K', '4K'])
    assert.deepEqual(byModel.qualities, [])
    assert.deepEqual(byModel.backgrounds, [])
    assert.equal(byModel.maxImages, 1)
    assert.equal(byModel.maxReferences, 14)
    assert.equal(byModel.supportsAutoSize, false)
    assert.equal(byModel.supportsExactSize, true)
    assert.equal(getImagePresetSize(byModel, '1K', '1:1'), '1024x1024')
    assert.equal(getImagePresetSize(byModel, '4K', '16:9'), '3328x1872')
    // 无 2K 档，避免与 64 的 gpt-image-2 能力混淆
    assert.equal(byModel.resolutions.includes('2K'), false)

    // 分组兜底：ic生图 下任意模型名也走 IC profile
    const byGroup = getImageModelProfile('ic生图', 'some-alias')
    assert.deepEqual(byGroup.resolutions, ['1K', '4K'])
    assert.equal(byGroup.maxImages, 1)
  })

  test('gpt-image-1 family supports multi-image references', () => {
    const profile = getImageModelProfile('default', 'gpt-image-1')
    assert.equal(profile.provider, 'gpt-image')
    assert.equal(profile.maxReferences, 14)
  })

  test('seedream models enable image-to-image references', () => {
    const pro = getImageModelProfile(
      'dddd-sd-图',
      'dola-seedream-5-0-pro-260628'
    )
    assert.equal(pro.provider, 'seedream')
    assert.equal(pro.maxReferences, 10)
    assert.ok(pro.sizes?.includes('1K'))
    assert.ok(pro.sizes?.includes('2K'))
    assert.ok(pro.sizes?.includes('2048x2048'))
    assert.ok(pro.responseFormats.includes('b64_json'))

    const lite = getImageModelProfile(
      'dddd-sd-图',
      'seedream-5-0-lite-260128'
    )
    assert.equal(lite.maxReferences, 14)
    assert.ok(lite.sizes?.includes('2K'))
    assert.ok(lite.sizes?.includes('3K'))
    assert.ok(!lite.sizes?.includes('1K'))

    const v45 = getImageModelProfile('dddd-sd-图', 'seedream-4-5-251128')
    assert.equal(v45.maxReferences, 14)
    assert.ok(v45.sizes?.includes('2K'))
    assert.ok(v45.sizes?.includes('4K'))

    const byGroup = getImageModelProfile('dddd-sd-图', 'custom-seedream-alias')
    assert.equal(byGroup.provider, 'seedream')
    // Unknown seedream alias keeps the broader size list and 14-ref default.
    assert.equal(byGroup.maxReferences, 14)
  })

  test('image-edit models expose reference uploads', () => {
    const profile = getImageModelProfile('default', 'qwen-image-edit-plus')
    assert.equal(profile.provider, 'generic')
    assert.equal(profile.maxReferences, 16)
  })

  test('plain text-to-image models hide reference uploads', () => {
    const profile = getImageModelProfile('default', 'dall-e-3')
    assert.equal(profile.maxReferences, 0)
  })

  test('exact gpt-image-2 sizes require positive multiples of sixteen', () => {
    assert.equal(isValidExactImageSize('2160', '3840'), true)
    assert.equal(isValidExactImageSize('2159', '3840'), false)
    assert.equal(isValidExactImageSize('0', '3840'), false)
  })
})
