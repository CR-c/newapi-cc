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
export interface VideoModelProfile {
  provider:
    | 'grok'
    | 'sub2api'
    | 'service-inference'
    | 'doubao'
    | 'kyy'
    | 'generic'
  durations: number[]
  aspectRatios: string[]
  resolutions: string[]
  maxImages: number
  maxVideos: number
  maxAudios: number
  requiresImage: boolean
  requiresImageWithAudio: boolean
  supportsGenerateAudio: boolean
  supportsWatermark: boolean
}

const GENERIC_PROFILE: VideoModelProfile = {
  provider: 'generic',
  durations: [],
  aspectRatios: ['16:9', '9:16', '1:1'],
  resolutions: ['720p'],
  maxImages: 0,
  maxVideos: 0,
  maxAudios: 0,
  requiresImage: false,
  requiresImageWithAudio: false,
  supportsGenerateAudio: false,
  supportsWatermark: false,
}

const KYY_MODEL_CAPABILITY_ALIASES: Readonly<Record<string, string>> = {
  'sd2.0-933': 'videos',
  'sd2.0-903': 'videos_pro',
  'sd2.0fast-903': 'videos_pro_fast',
  'sd2.0-431': 'videos_stable',
  'sd2.0fast-431': 'videos_stable_fast',
}

export function getVideoModelProfile(
  model: string,
  group?: string
): VideoModelProfile {
  const isKyyVideoGroup = group === 'sd-video' || group === 'kyy-sd'
  const kyyModel = KYY_MODEL_CAPABILITY_ALIASES[model] ?? model

  if (isKyyVideoGroup && kyyModel === 'videos') {
    return {
      provider: 'kyy',
      durations: [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['21:9', '16:9', '4:3', '1:1', '3:4', '9:16'],
      resolutions: ['720p'],
      maxImages: 9,
      maxVideos: 3,
      maxAudios: 3,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (
    isKyyVideoGroup &&
    (kyyModel === 'videos_stable' || kyyModel === 'videos_stable_fast')
  ) {
    return {
      provider: 'kyy',
      durations:
        kyyModel === 'videos_stable_fast'
          ? [10, 15]
          : [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['16:9', '9:16', '1:1'],
      resolutions: ['720p'],
      maxImages: 4,
      maxVideos: 3,
      maxAudios: 1,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (
    isKyyVideoGroup &&
    (kyyModel === 'videos_pro' || kyyModel === 'videos_pro_fast')
  ) {
    return {
      provider: 'kyy',
      durations: [10, 15],
      aspectRatios: ['16:9', '9:16', '1:1'],
      resolutions: ['720p'],
      maxImages: 9,
      maxVideos: 0,
      maxAudios: 3,
      requiresImage: false,
      requiresImageWithAudio: true,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model === 'grok-video-1.5') {
    return {
      provider: 'grok',
      durations: [4, 6, 8, 10, 12, 15],
      aspectRatios: ['16:9', '9:16'],
      resolutions: ['720p', '480p'],
      maxImages: 1,
      maxVideos: 0,
      maxAudios: 0,
      requiresImage: true,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model === 'grok-imagine-video-1.5-preview') {
    return {
      provider: 'wxart',
      durations: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['16:9', '9:16', '1:1', '4:3', '3:4', '3:2', '2:3'],
      resolutions: ['720p', '480p'],
      maxImages: 1,
      maxVideos: 0,
      maxAudios: 0,
      requiresImage: true,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model === 'grok-video-3') {
    return {
      provider: 'wxart',
      durations: [6, 10, 12, 16, 20],
      aspectRatios: ['16:9', '9:16', '1:1'],
      resolutions: ['720p', '480p'],
      maxImages: 7,
      maxVideos: 0,
      maxAudios: 0,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model === 'grok-image-video') {
    return {
      provider: 'grok',
      durations: [4, 6, 8, 10, 12, 15],
      aspectRatios: ['1:1', '16:9', '9:16', '4:3', '3:4', '3:2', '2:3'],
      resolutions: ['720p', '480p'],
      maxImages: 7,
      maxVideos: 0,
      maxAudios: 0,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model.startsWith('video-ds-2.0') || model === 'as-sd2.0-fast') {
    return {
      provider: 'sub2api',
      durations: [5, 10, 15],
      aspectRatios: ['16:9', '9:16', '1:1'],
      resolutions: [],
      maxImages: 4,
      maxVideos: 3,
      maxAudios: 1,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (group === 'sd-token' && model === 'doubao-seedance-2-0-260128') {
    return {
      provider: 'doubao',
      // Official V3: 4–15 seconds, or omit / use -1 for smart duration at the API layer.
      durations: [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['adaptive', '16:9', '9:16', '1:1', '4:3', '3:4', '21:9'],
      resolutions: ['480p', '720p', '1080p'],
      maxImages: 9,
      maxVideos: 3,
      maxAudios: 3,
      requiresImage: false,
      requiresImageWithAudio: true,
      supportsGenerateAudio: true,
      supportsWatermark: true,
    }
  }
  if (model.startsWith('dreamina-seedance-2-0-')) {
    return {
      provider: 'service-inference',
      durations: [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['1:1', '16:9', '9:16'],
      resolutions:
        model === 'dreamina-seedance-2-0-hc'
          ? ['480p', '720p', '1080p', '4k']
          : ['480p', '720p'],
      maxImages: 9,
      maxVideos: 3,
      maxAudios: 3,
      requiresImage: false,
      requiresImageWithAudio: false,
      supportsGenerateAudio: true,
      supportsWatermark: true,
    }
  }
  return GENERIC_PROFILE
}
