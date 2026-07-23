import { readFile } from 'node:fs/promises'

const source = await readFile(
  new URL('../public/docs.html', import.meta.url),
  'utf8'
)

function requireMatch(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

requireMatch(
  (source.match(/id="video-dddd"/g) || []).length === 1,
  'docs must contain exactly one video-dddd section'
)
requireMatch(
  source.includes('href="#video-dddd"'),
  'docs navigation must link to the video-dddd section'
)

for (const group of ['dddd-sd-image', 'sd-token']) {
  requireMatch(
    (source.match(new RegExp(`id="${group}"`, 'g')) || []).length === 1,
    `docs must contain exactly one ${group} section`
  )
  requireMatch(
    source.includes(`href="#${group}"`),
    `docs navigation must link to the ${group} section`
  )
}

const imageSectionMatch = source.match(
  /<section id="dddd-sd-image"[\s\S]*?<\/section>/
)
requireMatch(imageSectionMatch, 'dddd-sd-image section is missing')
const imageSection = imageSectionMatch[0]
for (const contract of [
  'dddd-sd-图',
  'dola-seedream-5-0-pro-260628',
  'seedream-4-5-251128',
  'seedream-5-0-lite-260128',
  'POST /v1/images/generations',
  '支持的参数',
  'watermark',
  'response_format',
  '1K',
  '2K',
  '3K',
  '4K',
  '0–10',
  '0–14',
  '多参考图',
  '使用须知',
]) {
  requireMatch(
    imageSection.includes(contract),
    `dddd-sd-image section is missing ${contract}`
  )
}

const sdTokenSectionMatch = source.match(
  /<section id="sd-token"[\s\S]*?<\/section>/
)
requireMatch(sdTokenSectionMatch, 'sd-token section is missing')
const sdTokenSection = sdTokenSectionMatch[0].replace(/\s+/g, ' ')
for (const contract of [
  'doubao-seedance-2-0-260128',
  'POST /v1/videos',
  'GET /v1/videos/{task_id}',
  'GET /v1/videos/{task_id}/content',
  '480p',
  '720p',
  '1080p',
  '官方基准价',
  '分组倍率 <code>0.78</code>',
  '4K 当前未启用',
  '按 Token',
  '使用须知',
]) {
  requireMatch(
    sdTokenSection.includes(contract),
    `sd-token section is missing ${contract}`
  )
}

for (const [tier, officialWithVideo, officialWithoutVideo, saleWithVideo, saleWithoutVideo] of [
  ['480p/720p', '28.00', '46.00', '21.84', '35.88'],
  ['1080p', '31.00', '51.00', '24.18', '39.78'],
]) {
  const expectedRow = `<tr> <td><code>${tier}</code></td> <td>¥${officialWithVideo}/百万 Token</td> <td>¥${officialWithoutVideo}/百万 Token</td> <td>¥${saleWithVideo}/百万 Token</td> <td>¥${saleWithoutVideo}/百万 Token</td> </tr>`
  requireMatch(
    sdTokenSection.includes(expectedRow),
    `sd-token ${tier} price mapping is incorrect`
  )
}

const sectionMatch = source.match(/<section id="video-dddd"[\s\S]*?<\/section>/)
requireMatch(sectionMatch, 'video-dddd section is missing')

const section = sectionMatch[0]
for (const contract of [
  'POST /v1/sd/assets',
  'GET /v1/sd/assets/{asset_id}',
  '/pg/assets',
  'HTTPS',
  'AssetType',
  'Image',
  'Video',
  'Audio',
  'asset://',
  'dreamina-seedance-2-0-fast-hc',
  'dreamina-seedance-2-0-hc',
  'dreamina-seedance-2-0-mini-hc',
  'POST /v1/video/generate',
  '情况 A：无图文生视频',
  '情况 B：单图参考',
  '情况 C：多图参考（最多 9 张）',
  '情况 D：图 + 视频 + 音频参考',
  '情况 E：禁止直接 URL 创建视频',
  '最多 9 张图片',
  '3 个视频',
  '3 个音频',
  'invalid_videos',
  'invalid_audios',
  '不要在 <code>/v1/videos.images</code>',
]) {
  requireMatch(
    section.includes(contract),
    `video-dddd section is missing ${contract}`
  )
}

console.log('docs contract check passed')
