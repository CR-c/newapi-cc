import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'All media': 'All media',
    'Describe the image you want to create': 'Describe the image you want to create',
    'Describe the video you want to create': 'Describe the video you want to create',
    'Download video': 'Download video',
    'Duration in seconds': 'Duration in seconds',
    'Failed to poll video task': 'Failed to poll video task',
    Generate: 'Generate',
    Generating: 'Generating',
    'Generated images will appear here': 'Generated images will appear here',
    'Generated video will appear here': 'Generated video will appear here',
    'Media generation failed': 'Media generation failed',
    Processing: 'Processing',
    Quality: 'Quality',
    Size: 'Size',
    hd: 'HD',
    standard: 'Standard',
    'Video generation failed': 'Video generation failed',
  },
  zh: {
    'All media': '全部媒体',
    'Describe the image you want to create': '描述你想生成的图片',
    'Describe the video you want to create': '描述你想生成的视频',
    'Download video': '下载视频',
    'Duration in seconds': '时长（秒）',
    'Failed to poll video task': '获取视频任务进度失败',
    Generate: '生成',
    Generating: '生成中',
    'Generated images will appear here': '生成的图片会显示在这里',
    'Generated video will appear here': '生成的视频会显示在这里',
    'Media generation failed': '媒体生成失败',
    Processing: '处理中',
    Quality: '品质',
    Size: '尺寸',
    hd: '高清',
    standard: '标准',
    'Video generation failed': '视频生成失败',
  },
  'zh-TW': {
    'All media': '全部媒體',
    'Describe the image you want to create': '描述你想產生的圖片',
    'Describe the video you want to create': '描述你想產生的影片',
    'Download video': '下載影片',
    'Duration in seconds': '時長（秒）',
    'Failed to poll video task': '取得影片任務進度失敗',
    Generate: '產生',
    Generating: '產生中',
    'Generated images will appear here': '產生的圖片會顯示在這裡',
    'Generated video will appear here': '產生的影片會顯示在這裡',
    'Media generation failed': '媒體產生失敗',
    Processing: '處理中',
    Quality: '品質',
    Size: '尺寸',
    hd: '高畫質',
    standard: '標準',
    'Video generation failed': '影片產生失敗',
  },
  fr: {
    'All media': 'Tous les médias',
    'Describe the image you want to create': "Décrivez l'image que vous souhaitez créer",
    'Describe the video you want to create': 'Décrivez la vidéo que vous souhaitez créer',
    'Download video': 'Télécharger la vidéo',
    'Duration in seconds': 'Durée en secondes',
    'Failed to poll video task': "Impossible d'obtenir la progression de la vidéo",
    Generate: 'Générer',
    Generating: 'Génération en cours',
    'Generated images will appear here': 'Les images générées apparaîtront ici',
    'Generated video will appear here': 'La vidéo générée apparaîtra ici',
    'Media generation failed': 'Échec de la génération du média',
    Processing: 'Traitement en cours',
    Quality: 'Qualité',
    Size: 'Dimensions',
    hd: 'HD',
    standard: 'Standard',
    'Video generation failed': 'Échec de la génération de la vidéo',
  },
  ja: {
    'All media': 'すべてのメディア',
    'Describe the image you want to create': '生成したい画像を説明してください',
    'Describe the video you want to create': '生成したい動画を説明してください',
    'Download video': '動画をダウンロード',
    'Duration in seconds': '長さ（秒）',
    'Failed to poll video task': '動画タスクの進行状況を取得できませんでした',
    Generate: '生成',
    Generating: '生成中',
    'Generated images will appear here': '生成された画像がここに表示されます',
    'Generated video will appear here': '生成された動画がここに表示されます',
    'Media generation failed': 'メディアの生成に失敗しました',
    Processing: '処理中',
    Quality: '品質',
    Size: 'サイズ',
    hd: '高画質',
    standard: '標準',
    'Video generation failed': '動画の生成に失敗しました',
  },
  ru: {
    'All media': 'Все медиа',
    'Describe the image you want to create': 'Опишите изображение, которое хотите создать',
    'Describe the video you want to create': 'Опишите видео, которое хотите создать',
    'Download video': 'Скачать видео',
    'Duration in seconds': 'Длительность в секундах',
    'Failed to poll video task': 'Не удалось получить статус задачи видео',
    Generate: 'Создать',
    Generating: 'Создание',
    'Generated images will appear here': 'Созданные изображения появятся здесь',
    'Generated video will appear here': 'Созданное видео появится здесь',
    'Media generation failed': 'Не удалось создать медиа',
    Processing: 'Обработка',
    Quality: 'Качество',
    Size: 'Размер',
    hd: 'Высокое качество',
    standard: 'Стандартное',
    'Video generation failed': 'Не удалось создать видео',
  },
  vi: {
    'All media': 'Tất cả nội dung đa phương tiện',
    'Describe the image you want to create': 'Mô tả hình ảnh bạn muốn tạo',
    'Describe the video you want to create': 'Mô tả video bạn muốn tạo',
    'Download video': 'Tải video xuống',
    'Duration in seconds': 'Thời lượng tính bằng giây',
    'Failed to poll video task': 'Không thể lấy tiến độ tác vụ video',
    Generate: 'Tạo',
    Generating: 'Đang tạo',
    'Generated images will appear here': 'Hình ảnh đã tạo sẽ xuất hiện tại đây',
    'Generated video will appear here': 'Video đã tạo sẽ xuất hiện tại đây',
    'Media generation failed': 'Tạo nội dung đa phương tiện thất bại',
    Processing: 'Đang xử lý',
    Quality: 'Chất lượng',
    Size: 'Kích thước',
    hd: 'Chất lượng cao',
    standard: 'Tiêu chuẩn',
    'Video generation failed': 'Tạo video thất bại',
  },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
