import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getVideoModelProfile } from './video-model-profiles'

describe('KYY video model profiles', () => {
  test('videos exposes nine images three videos and three audios', () => {
    const profile = getVideoModelProfile('videos', 'kyy-sd')

    assert.equal(profile.provider, 'kyy')
    assert.deepEqual(
      profile.durations,
      [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]
    )
    assert.deepEqual(profile.aspectRatios, [
      '21:9',
      '16:9',
      '4:3',
      '1:1',
      '3:4',
      '9:16',
    ])
    assert.deepEqual(profile.resolutions, ['720p'])
    assert.equal(profile.maxImages, 9)
    assert.equal(profile.maxVideos, 3)
    assert.equal(profile.maxAudios, 3)
  })

  test('stable models expose their documented limits', () => {
    const stable = getVideoModelProfile('videos_stable', 'kyy-sd')
    const fast = getVideoModelProfile('videos_stable_fast', 'kyy-sd')

    assert.deepEqual(
      stable.durations,
      [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]
    )
    assert.deepEqual(fast.durations, [10, 15])
    for (const profile of [stable, fast]) {
      assert.equal(profile.maxImages, 4)
      assert.equal(profile.maxVideos, 3)
      assert.equal(profile.maxAudios, 1)
    }
  })

  test('pro models require an image only when audio is supplied', () => {
    for (const model of ['videos_pro', 'videos_pro_fast']) {
      const profile = getVideoModelProfile(model, 'kyy-sd')
      assert.deepEqual(profile.durations, [10, 15])
      assert.equal(profile.maxImages, 9)
      assert.equal(profile.maxVideos, 0)
      assert.equal(profile.maxAudios, 3)
      assert.equal(profile.requiresImage, false)
      assert.equal(profile.requiresImageWithAudio, true)
    }
  })

  test('same model name outside kyy-sd keeps the generic profile', () => {
    const profile = getVideoModelProfile('videos', 'default')

    assert.equal(profile.provider, 'generic')
    assert.equal(profile.maxImages, 0)
    assert.equal(profile.maxVideos, 0)
    assert.equal(profile.maxAudios, 0)
  })
})
