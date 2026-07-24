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
import { describe, expect, test } from 'bun:test'

import { getImageModelProfile } from '../../image-model-profiles'
import { getVideoModelProfile } from '../../video-model-profiles'
import {
  getImageReferenceCapabilityHint,
  getVideoReferenceCapabilitySummary,
  getVideoReferenceRequirementHints,
  getVideoReferenceValidationIssue,
  isVideoReferenceSubmissionAllowed,
} from './media-capability-utils'

describe('media capability hints', () => {
  test('seedream pro reports a 10-image reference limit', () => {
    const profile = getImageModelProfile(
      'dddd-sd-图',
      'dola-seedream-5-0-pro-260628'
    )
    expect(getImageReferenceCapabilityHint(profile)).toEqual({
      key: 'This model allows up to {{count}} {{kind}}.',
      values: { count: 10, kind: 'images' },
    })
  })

  test('video-dddd summarizes 9/3/3 multimodal limits', () => {
    const profile = getVideoModelProfile(
      'dreamina-seedance-2-0-mini-hc',
      'video-dddd'
    )
    expect(getVideoReferenceCapabilitySummary(profile)).toEqual({
      key: 'This model allows up to {{images}} images, {{videos}} videos, and {{audios}} audios.',
      values: { images: 9, videos: 3, audios: 3 },
    })
    expect(getVideoReferenceRequirementHints(profile)).toContainEqual({
      key: 'Reference audio cannot be used alone; add at least one image or video.',
    })
  })

  test('rejects audio-only multimodal references', () => {
    const profile = getVideoModelProfile(
      'dreamina-seedance-2-0-mini-hc',
      'video-dddd'
    )
    expect(
      getVideoReferenceValidationIssue(profile, {
        images: 0,
        videos: 0,
        audios: 1,
      })
    ).toEqual({
      key: 'Reference audio cannot be used alone; add at least one image or video.',
    })
    expect(
      isVideoReferenceSubmissionAllowed(profile, {
        images: 1,
        videos: 0,
        audios: 1,
      })
    ).toBe(true)
  })

  test('kyy pro requires an image when audio is supplied', () => {
    const profile = getVideoModelProfile('sd2.0-903', 'sd-video')
    expect(getVideoReferenceCapabilitySummary(profile)).toEqual({
      key: 'This model allows up to {{images}} images and {{audios}} audios.',
      values: { images: 9, audios: 3 },
    })
    expect(
      getVideoReferenceValidationIssue(profile, {
        images: 0,
        videos: 0,
        audios: 1,
      })
    ).toEqual({
      key: 'When reference audio is provided, at least one reference image is required.',
    })
  })

  test('grok-video-1.5 requires exactly one reference image', () => {
    const profile = getVideoModelProfile('grok-video-1.5')
    expect(getVideoReferenceCapabilitySummary(profile)).toEqual({
      key: 'This model allows up to {{count}} {{kind}}.',
      values: { count: 1, kind: 'images' },
    })
    expect(
      getVideoReferenceValidationIssue(profile, {
        images: 0,
        videos: 0,
        audios: 0,
      })
    ).toEqual({
      key: 'This model requires exactly one reference image.',
    })
  })
})
