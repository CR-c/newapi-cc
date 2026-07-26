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
  provider: 'nano-banana' | 'gpt-image' | 'seedream' | 'generic'
  aspectRatios: string[]
  resolutions: string[]
  fixedResolution?: string
  presetSizes: Record<string, Record<string, string>>
  sizes?: string[]
  qualities: string[]
  backgrounds: string[]
  responseFormats: Array<'url' | 'b64_json'>
  maxImages: number
  /** Max reference images for image-to-image. 0 = channel does not expose refs in playground. */
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

const GENERIC_SIZES = ['1024x1024', '1024x1792', '1792x1024']

const SEEDREAM_SIZES = [
  '1K',
  '2K',
  '3K',
  '4K',
  '1024x1024',
  '1920x1920',
  '2048x2048',
  '2560x1440',
  '1440x2560',
  '1024x1792',
  '1792x1024',
]

/** Seedream 5.0 pro: 1K/2K or pixels; multi-ref up to 10. */
const SEEDREAM_PRO_SIZES = [
  '1K',
  '2K',
  '1024x1024',
  '1920x1920',
  '2048x2048',
  '2560x1440',
  '1440x2560',
  '1024x1792',
  '1792x1024',
]

/** Seedream 5.0 lite: 2K/3K or pixels; multi-ref up to 14. */
const SEEDREAM_LITE_SIZES = [
  '2K',
  '3K',
  '1920x1920',
  '2048x2048',
  '2560x1440',
  '1440x2560',
  '1024x1792',
  '1792x1024',
]

/** Seedream 4.5: 2K/4K or pixels; multi-ref up to 14. */
const SEEDREAM_45_SIZES = [
  '2K',
  '4K',
  '1920x1920',
  '2048x2048',
  '2560x1440',
  '1440x2560',
  '1024x1792',
  '1792x1024',
]

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
  sizes: GENERIC_SIZES,
  qualities: ['standard', 'hd'],
  backgrounds: [],
  responseFormats: ['url'],
  maxImages: 1,
  maxReferences: 0,
  supportsAutoSize: false,
  supportsExactSize: false,
}

function normalizeModel(model: string): string {
  return model.trim().toLowerCase()
}

function isNanoBananaExtreme(model: string): boolean {
  const m = normalizeModel(model)
  return (
    m === 'gemini-3.1-flash-image-preview' ||
    (m.includes('nano-banana') && m.includes('3.1'))
  )
}

function isNanoBananaStandard(model: string): boolean {
  const m = normalizeModel(model)
  if (isNanoBananaExtreme(model)) return false
  return (
    m === 'gemini-3-pro-image-preview' ||
    m === 'gemini-2.5-flash-image' ||
    (m.includes('gemini') && m.includes('image')) ||
    m.includes('nano-banana')
  )
}

function isGptImage2Ic(model: string, group?: string): boolean {
  const m = normalizeModel(model)
  if (m === 'gpt-image-2-ic') {
    return true
  }
  // ic生图 分组仅此生图产品线；按分组兜底，避免别名漏匹配
  return group === 'ic生图'
}

function isGptImage2(model: string): boolean {
  const m = normalizeModel(model)
  if (m === 'gpt-image-2-ic') {
    return false
  }
  return m === 'gpt-image-2' || m.startsWith('gpt-image-2-')
}

function isGptImageFamily(model: string): boolean {
  const m = normalizeModel(model)
  return m.startsWith('gpt-image') || m === 'chatgpt-image-latest'
}

function isSeedreamModel(model: string): boolean {
  const m = normalizeModel(model)
  return (
    m.includes('seedream') ||
    m.includes('dola-seedream') ||
    m.startsWith('doubao-seedream')
  )
}

function isImageEditModel(model: string): boolean {
  const m = normalizeModel(model)
  return (
    m.includes('image-edit') ||
    m.includes('image_edit') ||
    (m.endsWith('-edit') && m.includes('image')) ||
    m.includes('qwen-image-edit') ||
    (m.includes('flux') && (m.includes('kontext') || m.includes('edit')))
  )
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

function createGptImage2Profile(): ImageModelProfile {
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

/** ic生图 · gpt-image-2-ic：1k 出图，支持超分 4k；按次 1 张 */
function createGptImage2IcProfile(): ImageModelProfile {
  return {
    provider: 'gpt-image',
    aspectRatios: [...STANDARD_ASPECT_RATIOS],
    resolutions: ['1K', '4K'],
    presetSizes: {
      '1K': GPT_IMAGE_SIZES['1K'],
      '4K': GPT_IMAGE_SIZES['4K'],
    },
    // 上游 quality/background 兼容字段暂不影响结果，演练场不展示
    qualities: [],
    backgrounds: [],
    responseFormats: ['url', 'b64_json'],
    maxImages: 1,
    maxReferences: 14,
    supportsAutoSize: false,
    supportsExactSize: true,
  }
}

function createGptImageBasicProfile(): ImageModelProfile {
  return {
    provider: 'gpt-image',
    aspectRatios: [...STANDARD_ASPECT_RATIOS],
    resolutions: ['1K', '2K'],
    presetSizes: {
      '1K': GPT_IMAGE_SIZES['1K'],
      '2K': GPT_IMAGE_SIZES['2K'],
    },
    qualities: ['low', 'medium', 'high', 'auto'],
    backgrounds: ['opaque', 'transparent'],
    responseFormats: ['url', 'b64_json'],
    maxImages: 4,
    maxReferences: 14,
    supportsAutoSize: true,
    supportsExactSize: false,
  }
}

function createSeedreamProfile(model?: string): ImageModelProfile {
  const m = normalizeModel(model ?? '')
  const isPro =
    m.includes('seedream-5-0-pro') ||
    m.includes('seedream-5.0-pro') ||
    m.includes('dola-seedream-5-0-pro')
  const isLite =
    m.includes('seedream-5-0-lite') ||
    m.includes('seedream-5.0-lite') ||
    (m.includes('seedream-5-0') && m.includes('lite'))
  const is45 = m.includes('seedream-4-5') || m.includes('seedream-4.5')

  let sizes = SEEDREAM_SIZES
  let maxReferences = 14
  if (isPro) {
    sizes = SEEDREAM_PRO_SIZES
    maxReferences = 10
  } else if (isLite) {
    sizes = SEEDREAM_LITE_SIZES
    maxReferences = 14
  } else if (is45) {
    sizes = SEEDREAM_45_SIZES
    maxReferences = 14
  }

  return {
    provider: 'seedream',
    aspectRatios: [],
    resolutions: [],
    presetSizes: GENERIC_PROFILE.presetSizes,
    sizes,
    qualities: [],
    backgrounds: [],
    // Official Seedream supports url and b64_json.
    responseFormats: ['url', 'b64_json'],
    maxImages: 1,
    // Multi-image reference via unified `image` field (pro ≤10, lite/4.5 ≤14).
    maxReferences,
    supportsAutoSize: false,
    supportsExactSize: true,
  }
}

function createImageEditProfile(): ImageModelProfile {
  return {
    ...GENERIC_PROFILE,
    maxReferences: 16,
    responseFormats: ['url', 'b64_json'],
  }
}

/**
 * Resolve playground image controls and image-to-image reference limits.
 * Different channels/models support different reference counts; 0 hides the upload UI.
 */
export function getImageModelProfile(
  group: string,
  model: string
): ImageModelProfile {
  if (isNanoBananaExtreme(model)) {
    return createNanoProfile(true)
  }
  if (isNanoBananaStandard(model)) {
    return createNanoProfile(false)
  }
  if (isGptImage2Ic(model, group)) {
    return createGptImage2IcProfile()
  }
  if (isGptImage2(model)) {
    return createGptImage2Profile()
  }
  if (isGptImageFamily(model)) {
    return createGptImageBasicProfile()
  }
  if (isSeedreamModel(model) || group === 'dddd-sd-图') {
    return createSeedreamProfile(model)
  }
  if (isImageEditModel(model)) {
    return createImageEditProfile()
  }

  // Legacy group gate: 64生图 models that are not name-matched still get no
  // references unless they match the families above.
  if (group === '64生图') {
    return GENERIC_PROFILE
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
