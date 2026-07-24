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
import type { ImageModelProfile } from '../../image-model-profiles'
import type { PlaygroundAssetKind } from '../../types'
import type { VideoModelProfile } from '../../video-model-profiles'

export type MediaCapabilityKind = PlaygroundAssetKind

export type CapabilityMessage = {
  key: string
  values?: Record<string, string | number>
}

const KIND_LABEL_KEYS: Record<MediaCapabilityKind, string> = {
  image: 'images',
  video: 'videos',
  audio: 'audios',
}

export function getReferenceLimitHintKey(): string {
  return 'This model allows up to {{count}} {{kind}}.'
}

export function getReferenceLimitReachedKey(): string {
  return 'This model allows up to {{count}} {{kind}}. Remove some before adding more.'
}

export function getReferenceKindLabelKey(kind: MediaCapabilityKind): string {
  return KIND_LABEL_KEYS[kind]
}

export function getImageReferenceCapabilityHint(
  profile: ImageModelProfile
): CapabilityMessage {
  if (profile.maxReferences <= 0) {
    return {
      key: 'This model does not support reference images.',
    }
  }

  return {
    key: getReferenceLimitHintKey(),
    values: {
      count: profile.maxReferences,
      kind: getReferenceKindLabelKey('image'),
    },
  }
}

export function getImageOutputCountHint(
  profile: ImageModelProfile
): CapabilityMessage | null {
  if (profile.maxImages <= 1) return null
  return {
    key: 'This model can generate up to {{count}} images per request.',
    values: { count: profile.maxImages },
  }
}

export type VideoReferenceCapabilityLine = {
  kind: MediaCapabilityKind
  max: number
}

export function getVideoReferenceCapabilityLines(
  profile: VideoModelProfile
): VideoReferenceCapabilityLine[] {
  const lines: VideoReferenceCapabilityLine[] = []
  if (profile.maxImages > 0) {
    lines.push({ kind: 'image', max: profile.maxImages })
  }
  if (profile.maxVideos > 0) {
    lines.push({ kind: 'video', max: profile.maxVideos })
  }
  if (profile.maxAudios > 0) {
    lines.push({ kind: 'audio', max: profile.maxAudios })
  }
  return lines
}

export function getVideoReferenceCapabilitySummary(
  profile: VideoModelProfile
): CapabilityMessage {
  const lines = getVideoReferenceCapabilityLines(profile)
  if (lines.length === 0) {
    return {
      key: 'This model does not support reference media.',
    }
  }

  if (
    profile.maxImages > 0 &&
    profile.maxVideos > 0 &&
    profile.maxAudios > 0
  ) {
    return {
      key: 'This model allows up to {{images}} images, {{videos}} videos, and {{audios}} audios.',
      values: {
        images: profile.maxImages,
        videos: profile.maxVideos,
        audios: profile.maxAudios,
      },
    }
  }

  if (profile.maxImages > 0 && profile.maxVideos > 0) {
    return {
      key: 'This model allows up to {{images}} images and {{videos}} videos.',
      values: {
        images: profile.maxImages,
        videos: profile.maxVideos,
      },
    }
  }

  if (profile.maxImages > 0 && profile.maxAudios > 0) {
    return {
      key: 'This model allows up to {{images}} images and {{audios}} audios.',
      values: {
        images: profile.maxImages,
        audios: profile.maxAudios,
      },
    }
  }

  const only = lines[0]
  return {
    key: getReferenceLimitHintKey(),
    values: {
      count: only.max,
      kind: getReferenceKindLabelKey(only.kind),
    },
  }
}

export function getVideoReferenceRequirementHints(
  profile: VideoModelProfile
): CapabilityMessage[] {
  const hints: CapabilityMessage[] = []
  if (profile.requiresImage) {
    hints.push({
      key: 'This model requires exactly one reference image.',
    })
  }
  if (profile.requiresImageWithAudio) {
    hints.push({
      key: 'When reference audio is provided, at least one reference image is required.',
    })
  } else if (profile.maxAudios > 0) {
    hints.push({
      key: 'Reference audio cannot be used alone; add at least one image or video.',
    })
  }
  return hints
}

export type VideoReferenceCounts = {
  images: number
  videos: number
  audios: number
}

export function getVideoReferenceValidationIssue(
  profile: VideoModelProfile,
  counts: VideoReferenceCounts
): CapabilityMessage | null {
  if (profile.requiresImage && counts.images !== 1) {
    return {
      key: 'This model requires exactly one reference image.',
    }
  }

  if (
    profile.requiresImageWithAudio &&
    counts.audios > 0 &&
    counts.images < 1
  ) {
    return {
      key: 'When reference audio is provided, at least one reference image is required.',
    }
  }

  // Multimodal providers generally reject audio-only reference sets.
  if (
    profile.maxAudios > 0 &&
    counts.audios > 0 &&
    counts.images === 0 &&
    counts.videos === 0
  ) {
    return {
      key: 'Reference audio cannot be used alone; add at least one image or video.',
    }
  }

  if (counts.images > profile.maxImages) {
    return {
      key: getReferenceLimitReachedKey(),
      values: {
        count: profile.maxImages,
        kind: getReferenceKindLabelKey('image'),
      },
    }
  }
  if (counts.videos > profile.maxVideos) {
    return {
      key: getReferenceLimitReachedKey(),
      values: {
        count: profile.maxVideos,
        kind: getReferenceKindLabelKey('video'),
      },
    }
  }
  if (counts.audios > profile.maxAudios) {
    return {
      key: getReferenceLimitReachedKey(),
      values: {
        count: profile.maxAudios,
        kind: getReferenceKindLabelKey('audio'),
      },
    }
  }

  return null
}

export function isVideoReferenceSubmissionAllowed(
  profile: VideoModelProfile,
  counts: VideoReferenceCounts
): boolean {
  return getVideoReferenceValidationIssue(profile, counts) === null
}
