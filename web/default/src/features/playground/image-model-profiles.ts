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

export interface ImageModelProfile {
  provider: 'nano-banana' | 'gpt-image' | 'generic'
  aspectRatios: string[]
  resolutions: string[]
  fixedResolution?: string
  presetSizes: Record<string, Record<string, string>>
  qualities: string[]
  backgrounds: string[]
  responseFormats: Array<'url' | 'b64_json'>
  maxImages: number
  maxReferences: number
  supportsAutoSize: boolean
  supportsExactSize: boolean
}

const STANDARD_ASPECT_RATIOS = [
  '1:1',
  '16:9',
  '9:16',
  '4:3',
  '3:4',
  '21:9',
  '3:2',
  '2:3',
  '5:4',
  '4:5',
]

const NANO_2K_SIZES: Record<string, string> = {
  '1:1': '1024x1024',
  '16:9': '1280x720',
  '9:16': '720x1280',
  '4:3': '1024x768',
  '3:4': '768x1024',
  '21:9': '1344x576',
  '3:2': '1056x704',
  '2:3': '704x1056',
  '5:4': '960x768',
  '4:5': '768x960',
  '1:4': '256x1024',
  '1:8': '128x1024',
  '4:1': '1024x256',
  '8:1': '1024x128',
}

const GPT_IMAGE_SIZES: Record<string, Record<string, string>> = {
  '1K': {
    '1:1': '1024x1024',
    '16:9': '1280x720',
    '9:16': '720x1280',
    '4:3': '1152x864',
    '3:4': '864x1152',
    '21:9': '1536x656',
    '3:2': '1216x816',
    '2:3': '816x1216',
    '5:4': '1120x896',
    '4:5': '896x1120',
  },
  '2K': {
    '1:1': '2048x2048',
    '16:9': '2560x1440',
    '9:16': '1440x2560',
    '4:3': '2304x1728',
    '3:4': '1728x2304',
    '21:9': '3072x1312',
    '3:2': '2432x1632',
    '2:3': '1632x2432',
    '5:4': '2240x1792',
    '4:5': '1792x2240',
  },
  '4K': {
    '1:1': '2480x2480',
    '16:9': '3328x1872',
    '9:16': '1872x3328',
    '4:3': '2880x2160',
    '3:4': '2160x2880',
    '21:9': '3984x1712',
    '3:2': '3040x2032',
    '2:3': '2032x3040',
    '5:4': '2800x2240',
    '4:5': '2240x2800',
  },
}

const GENERIC_PROFILE: ImageModelProfile = {
  provider: 'generic',
  aspectRatios: [],
  resolutions: [],
  presetSizes: {
    default: {
      square: '1024x1024',
      portrait: '1024x1792',
      landscape: '1792x1024',
    },
  },
  qualities: ['standard', 'hd'],
  backgrounds: [],
  responseFormats: ['url'],
  maxImages: 1,
  maxReferences: 0,
  supportsAutoSize: false,
  supportsExactSize: false,
}

function createNanoProfile(includeExtremeRatios: boolean): ImageModelProfile {
  return {
    provider: 'nano-banana',
    aspectRatios: includeExtremeRatios
      ? [...STANDARD_ASPECT_RATIOS, '1:4', '1:8', '4:1', '8:1']
      : [...STANDARD_ASPECT_RATIOS],
    resolutions: [],
    fixedResolution: '2K',
    presetSizes: { '2K': NANO_2K_SIZES },
    qualities: [],
    backgrounds: [],
    responseFormats: ['url', 'b64_json'],
    maxImages: 4,
    maxReferences: 14,
    supportsAutoSize: false,
    supportsExactSize: false,
  }
}

export function getImageModelProfile(
  group: string,
  model: string
): ImageModelProfile {
  if (group !== '64生图') return GENERIC_PROFILE

  if (model === 'gemini-3.1-flash-image-preview') {
    return createNanoProfile(true)
  }
  if (
    model === 'gemini-3-pro-image-preview' ||
    model === 'gemini-2.5-flash-image'
  ) {
    return createNanoProfile(false)
  }
  if (model === 'gpt-image-2') {
    return {
      provider: 'gpt-image',
      aspectRatios: [...STANDARD_ASPECT_RATIOS],
      resolutions: ['1K', '2K', '4K'],
      presetSizes: GPT_IMAGE_SIZES,
      qualities: ['low', 'medium', 'high'],
      backgrounds: ['opaque', 'transparent'],
      responseFormats: ['url', 'b64_json'],
      maxImages: 4,
      maxReferences: 14,
      supportsAutoSize: true,
      supportsExactSize: true,
    }
  }
  return GENERIC_PROFILE
}

export function getImagePresetSize(
  profile: ImageModelProfile,
  resolution: string,
  aspectRatio: string
): string {
  const resolutionSizes =
    profile.presetSizes[profile.fixedResolution ?? resolution]
  if (!resolutionSizes) return '1024x1024'
  return resolutionSizes[aspectRatio] ?? Object.values(resolutionSizes)[0]
}

export function isValidExactImageSize(
  widthValue: string,
  heightValue: string
): boolean {
  const width = Number(widthValue)
  const height = Number(heightValue)
  return (
    Number.isInteger(width) &&
    Number.isInteger(height) &&
    width > 0 &&
    height > 0 &&
    width % 16 === 0 &&
    height % 16 === 0
  )
}
