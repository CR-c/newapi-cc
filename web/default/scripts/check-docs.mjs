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
]) {
  requireMatch(
    section.includes(contract),
    `video-dddd section is missing ${contract}`
  )
}

console.log('docs contract check passed')
