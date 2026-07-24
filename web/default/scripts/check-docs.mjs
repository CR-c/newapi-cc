import { readFile } from 'node:fs/promises'

const docsSource = await readFile(
  new URL('../public/docs.html', import.meta.url),
  'utf8'
)
const officialSource = await readFile(
  new URL('../public/docs-official.html', import.meta.url),
  'utf8'
)
const cssSource = await readFile(
  new URL('../public/docs.css', import.meta.url),
  'utf8'
)
const jsSource = await readFile(
  new URL('../public/docs.js', import.meta.url),
  'utf8'
)

function requireMatch(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

// Shared chrome: page tabs + stylesheet
for (const [name, source] of [
  ['docs.html', docsSource],
  ['docs-official.html', officialSource],
]) {
  requireMatch(
    source.includes('href="/docs.css"') || source.includes("href='/docs.css'"),
    `${name} must link shared docs.css`
  )
  requireMatch(
    source.includes('src="/docs.js"') || source.includes("src='/docs.js'"),
    `${name} must load docs.js for section highlight`
  )
  requireMatch(
    source.includes('class="page-tabs"') || source.includes("class='page-tabs'"),
    `${name} must include page-tabs nav`
  )
  requireMatch(
    source.includes('href="/docs.html"'),
    `${name} must link unified docs route /docs.html`
  )
  requireMatch(
    source.includes('href="/docs-official.html"'),
    `${name} must link official guide route /docs-official.html`
  )
  requireMatch(
    source.includes('class="docs-sticky"') ||
      source.includes("class='docs-sticky'"),
    `${name} must use sticky docs chrome shell`
  )
}

requireMatch(
  docsSource.includes('aria-current="page"') &&
    docsSource.includes('统一图片与视频 API'),
  'docs.html must mark unified page as current'
)
requireMatch(
  officialSource.includes('aria-current="page"') &&
    officialSource.includes('官方 New API 对接'),
  'docs-official.html must mark official page as current'
)

// Official page must be its own document (not merged into unified long page)
requireMatch(
  !docsSource.includes('官方 New API 对接本站图片与视频'),
  'docs.html must not embed the full official integration guide title'
)
requireMatch(
  officialSource.includes('官方 New API 对接本站图片与视频'),
  'docs-official.html must contain official guide title'
)

// Official guide contracts
for (const contract of [
  'POST /v1/images/generations',
  'POST /v1/videos',
  '/v1/videos/{task_id}',
  '/v1/videos/{task_id}/content',
  'POST /v1/video/generations',
  'POST /v1/sd/assets',
  '/v1/sd/assets',
  '/pg/assets',
  'asset://',
  'Authorization: Bearer',
  'video-dddd',
  'sd-token',
  'sd-video',
  'dddd-sd-图',
  'github.com/QuantumNous/new-api',
  'id="start"',
  'id="images"',
  'id="videos"',
  'id="assets"',
  'id="compat"',
  'id="clients"',
  'id="faq"',
  'id="checklist"',
]) {
  requireMatch(
    officialSource.includes(contract),
    `docs-official.html is missing ${contract}`
  )
}

requireMatch(cssSource.includes('.page-tabs'), 'docs.css must style page tabs')
requireMatch(
  cssSource.includes('.page-tabs a.is-active'),
  'docs.css must highlight active page tab'
)
requireMatch(
  cssSource.includes('.section-nav a.is-active'),
  'docs.css must highlight active section link'
)
requireMatch(
  cssSource.includes('.docs-sticky'),
  'docs.css must style sticky docs shell'
)
requireMatch(
  jsSource.includes('IntersectionObserver') || jsSource.includes('pickFromScroll'),
  'docs.js must update section highlight on scroll'
)
requireMatch(
  jsSource.includes("classList.toggle('is-active'") ||
    jsSource.includes('classList.toggle("is-active"'),
  'docs.js must toggle is-active on section links'
)

// --- Existing unified docs contracts ---

requireMatch(
  (docsSource.match(/id="video-dddd"/g) || []).length === 1,
  'docs must contain exactly one video-dddd section'
)
requireMatch(
  docsSource.includes('href="#video-dddd"'),
  'docs navigation must link to the video-dddd section'
)

for (const group of ['dddd-sd-image', 'sd-token']) {
  requireMatch(
    (docsSource.match(new RegExp(`id="${group}"`, 'g')) || []).length === 1,
    `docs must contain exactly one ${group} section`
  )
  requireMatch(
    docsSource.includes(`href="#${group}"`),
    `docs navigation must link to the ${group} section`
  )
}

const imageSectionMatch = docsSource.match(
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

const sdTokenSectionMatch = docsSource.match(
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
  '本分组没有素材库',
  '/v1/sd/assets',
  'asset://',
  'HTTPS URL',
  'images',
  'videos',
  'audios',
  '使用须知',
]) {
  requireMatch(
    sdTokenSection.includes(contract),
    `sd-token section is missing ${contract}`
  )
}
requireMatch(
  sdTokenSection.includes('不要调用') &&
    sdTokenSection.includes('/v1/sd/assets') &&
    sdTokenSection.includes('直接'),
  'sd-token section must tell users to pass HTTPS params directly without the asset library'
)

for (const [
  tier,
  officialWithVideo,
  officialWithoutVideo,
  saleWithVideo,
  saleWithoutVideo,
] of [
  ['480p/720p', '28.00', '46.00', '21.84', '35.88'],
  ['1080p', '31.00', '51.00', '24.18', '39.78'],
]) {
  const expectedRow = `<tr> <td><code>${tier}</code></td> <td>¥${officialWithVideo}/百万 Token</td> <td>¥${officialWithoutVideo}/百万 Token</td> <td>¥${saleWithVideo}/百万 Token</td> <td>¥${saleWithoutVideo}/百万 Token</td> </tr>`
  requireMatch(
    sdTokenSection.includes(expectedRow),
    `sd-token ${tier} price mapping is incorrect`
  )
}

const sectionMatch = docsSource.match(
  /<section id="video-dddd"[\s\S]*?<\/section>/
)
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
