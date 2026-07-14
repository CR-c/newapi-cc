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
  provider: 'grok' | 'sub2api' | 'service-inference' | 'generic'
  durations: number[]
  aspectRatios: string[]
  resolutions: string[]
  maxImages: number
  maxVideos: number
  maxAudios: number
  requiresImage: boolean
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
  supportsGenerateAudio: false,
  supportsWatermark: false,
}

export function getVideoModelProfile(model: string): VideoModelProfile {
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
      supportsGenerateAudio: false,
      supportsWatermark: false,
    }
  }
  if (model.startsWith('dreamina-seedance-2-0-')) {
    return {
      provider: 'service-inference',
      durations: [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      aspectRatios: ['1:1', '16:9', '9:16'],
      resolutions: ['480p', '720p', '1080p', '4k'],
      maxImages: 4,
      maxVideos: 0,
      maxAudios: 0,
      requiresImage: false,
      supportsGenerateAudio: true,
      supportsWatermark: true,
    }
  }
  return GENERIC_PROFILE
}
