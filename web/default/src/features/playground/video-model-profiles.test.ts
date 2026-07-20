import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getVideoModelProfile } from './video-model-profiles'

describe('KYY video model profiles', () => {
  test('sd-video uses the KYY model capabilities', () => {
    const profile = getVideoModelProfile('sd2.0-933', 'sd-video')

    assert.equal(profile.provider, 'kyy')
    assert.equal(profile.maxImages, 9)
    assert.equal(profile.maxVideos, 3)
    assert.equal(profile.maxAudios, 3)
  })

  test('933 alias exposes nine images three videos and three audios', () => {
    const profile = getVideoModelProfile('sd2.0-933', 'sd-video')

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
    const stable = getVideoModelProfile('sd2.0-431', 'sd-video')
    const fast = getVideoModelProfile('sd2.0fast-431', 'sd-video')

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
    for (const model of ['sd2.0-903', 'sd2.0fast-903']) {
      const profile = getVideoModelProfile(model, 'sd-video')
      assert.deepEqual(profile.durations, [10, 15])
      assert.equal(profile.maxImages, 9)
      assert.equal(profile.maxVideos, 0)
      assert.equal(profile.maxAudios, 3)
      assert.equal(profile.requiresImage, false)
      assert.equal(profile.requiresImageWithAudio, true)
    }
  })

  test('same model alias outside sd-video keeps the generic profile', () => {
    const profile = getVideoModelProfile('sd2.0-933', 'default')

    assert.equal(profile.provider, 'generic')
    assert.equal(profile.maxImages, 0)
    assert.equal(profile.maxVideos, 0)
    assert.equal(profile.maxAudios, 0)
  })
})

describe('service inference video model profiles', () => {
  test('only the full model exposes 1080p and 4k', () => {
    const full = getVideoModelProfile('dreamina-seedance-2-0-hc', 'video-dddd')
    const fast = getVideoModelProfile(
      'dreamina-seedance-2-0-fast-hc',
      'video-dddd'
    )
    const mini = getVideoModelProfile(
      'dreamina-seedance-2-0-mini-hc',
      'video-dddd'
    )

    assert.deepEqual(full.resolutions, ['480p', '720p', '1080p', '4k'])
    assert.deepEqual(fast.resolutions, ['480p', '720p'])
    assert.deepEqual(mini.resolutions, ['480p', '720p'])
  })
})
