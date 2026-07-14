import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Add audio': 'Add audio',
    'Add images': 'Add images',
    'Add videos': 'Add videos',
    'All media': 'All media',
    'Aspect ratio': 'Aspect ratio',
    'Describe the image you want to create':
      'Describe the image you want to create',
    'Describe the video you want to create':
      'Describe the video you want to create',
    'Download video': 'Download video',
    'Duration in seconds': 'Duration in seconds',
    'Failed to load media history': 'Failed to load media history',
    'Failed to poll video task': 'Failed to poll video task',
    Generate: 'Generate',
    'Generate audio': 'Generate audio',
    Generating: 'Generating',
    'Generated images will appear here': 'Generated images will appear here',
    'Generated video will appear here': 'Generated video will appear here',
    'Media generation failed': 'Media generation failed',
    'No media generated in the last 24 hours':
      'No media generated in the last 24 hours',
    Processing: 'Processing',
    Quality: 'Quality',
    'Reference audio': 'Reference audio',
    'Reference images': 'Reference images',
    'Reference limit reached': 'Reference limit reached',
    'Reference videos': 'Reference videos',
    'Recent 24 hours': 'Recent 24 hours',
    'Refresh history': 'Refresh history',
    'Remove reference': 'Remove reference',
    Resolution: 'Resolution',
    Size: 'Size',
    hd: 'HD',
    standard: 'Standard',
    'Video generation failed': 'Video generation failed',
    Watermark: 'Watermark',
  },
  zh: {
    'Add audio': '添加音频',
    'Add images': '添加图片',
    'Add videos': '添加视频',
    'All media': '全部媒体',
    'Aspect ratio': '画面比例',
    'Describe the image you want to create': '描述你想生成的图片',
    'Describe the video you want to create': '描述你想生成的视频',
    'Download video': '下载视频',
    'Duration in seconds': '时长（秒）',
    'Failed to load media history': '加载媒体历史记录失败',
    'Failed to poll video task': '获取视频任务进度失败',
    Generate: '生成',
    'Generate audio': '生成音频',
    Generating: '生成中',
    'Generated images will appear here': '生成的图片会显示在这里',
    'Generated video will appear here': '生成的视频会显示在这里',
    'Media generation failed': '媒体生成失败',
    'No media generated in the last 24 hours': '最近24小时内没有生成记录',
    Processing: '处理中',
    Quality: '品质',
    'Reference audio': '参考音频',
    'Reference images': '参考图片',
    'Reference limit reached': '已达到参考素材数量上限',
    'Reference videos': '参考视频',
    'Recent 24 hours': '最近24小时',
    'Refresh history': '刷新历史记录',
    'Remove reference': '移除参考素材',
    Resolution: '分辨率',
    Size: '尺寸',
    hd: '高清',
    standard: '标准',
    'Video generation failed': '视频生成失败',
    Watermark: '水印',
  },
  'zh-TW': {
    'Add audio': '新增音訊',
    'Add images': '新增圖片',
    'Add videos': '新增影片',
    'All media': '全部媒體',
    'Aspect ratio': '畫面比例',
    'Describe the image you want to create': '描述你想產生的圖片',
    'Describe the video you want to create': '描述你想產生的影片',
    'Download video': '下載影片',
    'Duration in seconds': '時長（秒）',
    'Failed to load media history': '載入媒體歷史記錄失敗',
    'Failed to poll video task': '取得影片任務進度失敗',
    Generate: '產生',
    'Generate audio': '產生音訊',
    Generating: '產生中',
    'Generated images will appear here': '產生的圖片會顯示在這裡',
    'Generated video will appear here': '產生的影片會顯示在這裡',
    'Media generation failed': '媒體產生失敗',
    'No media generated in the last 24 hours': '最近24小時內沒有產生記錄',
    Processing: '處理中',
    Quality: '品質',
    'Reference audio': '參考音訊',
    'Reference images': '參考圖片',
    'Reference limit reached': '已達到參考素材數量上限',
    'Reference videos': '參考影片',
    'Recent 24 hours': '最近24小時',
    'Refresh history': '重新整理歷史記錄',
    'Remove reference': '移除參考素材',
    Resolution: '解析度',
    Size: '尺寸',
    hd: '高畫質',
    standard: '標準',
    'Video generation failed': '影片產生失敗',
    Watermark: '浮水印',
  },
  fr: {
    'Add audio': "Ajouter de l'audio",
    'Add images': 'Ajouter des images',
    'Add videos': 'Ajouter des vidéos',
    'All media': 'Tous les médias',
    'Aspect ratio': "Format d'image",
    'Describe the image you want to create':
      "Décrivez l'image que vous souhaitez créer",
    'Describe the video you want to create':
      'Décrivez la vidéo que vous souhaitez créer',
    'Download video': 'Télécharger la vidéo',
    'Duration in seconds': 'Durée en secondes',
    'Failed to load media history':
      "Impossible de charger l'historique des médias",
    'Failed to poll video task':
      "Impossible d'obtenir la progression de la vidéo",
    Generate: 'Générer',
    'Generate audio': "Générer l'audio",
    Generating: 'Génération en cours',
    'Generated images will appear here': 'Les images générées apparaîtront ici',
    'Generated video will appear here': 'La vidéo générée apparaîtra ici',
    'Media generation failed': 'Échec de la génération du média',
    'No media generated in the last 24 hours':
      'Aucun média généré au cours des dernières 24 heures',
    Processing: 'Traitement en cours',
    Quality: 'Qualité',
    'Reference audio': 'Audio de référence',
    'Reference images': 'Images de référence',
    'Reference limit reached': 'Limite de références atteinte',
    'Reference videos': 'Vidéos de référence',
    'Recent 24 hours': '24 dernières heures',
    'Refresh history': "Actualiser l'historique",
    'Remove reference': 'Supprimer la référence',
    Resolution: 'Résolution',
    Size: 'Dimensions',
    hd: 'HD',
    standard: 'Standard',
    'Video generation failed': 'Échec de la génération de la vidéo',
    Watermark: 'Filigrane',
  },
  ja: {
    'Add audio': '音声を追加',
    'Add images': '画像を追加',
    'Add videos': '動画を追加',
    'All media': 'すべてのメディア',
    'Aspect ratio': 'アスペクト比',
    'Describe the image you want to create': '生成したい画像を説明してください',
    'Describe the video you want to create': '生成したい動画を説明してください',
    'Download video': '動画をダウンロード',
    'Duration in seconds': '長さ（秒）',
    'Failed to load media history': 'メディア履歴を読み込めませんでした',
    'Failed to poll video task': '動画タスクの進行状況を取得できませんでした',
    Generate: '生成',
    'Generate audio': '音声を生成',
    Generating: '生成中',
    'Generated images will appear here': '生成された画像がここに表示されます',
    'Generated video will appear here': '生成された動画がここに表示されます',
    'Media generation failed': 'メディアの生成に失敗しました',
    'No media generated in the last 24 hours':
      '過去24時間に生成されたメディアはありません',
    Processing: '処理中',
    Quality: '品質',
    'Reference audio': '参照音声',
    'Reference images': '参照画像',
    'Reference limit reached': '参照素材の上限に達しました',
    'Reference videos': '参照動画',
    'Recent 24 hours': '過去24時間',
    'Refresh history': '履歴を更新',
    'Remove reference': '参照素材を削除',
    Resolution: '解像度',
    Size: 'サイズ',
    hd: '高画質',
    standard: '標準',
    'Video generation failed': '動画の生成に失敗しました',
    Watermark: '透かし',
  },
  ru: {
    'Add audio': 'Добавить аудио',
    'Add images': 'Добавить изображения',
    'Add videos': 'Добавить видео',
    'All media': 'Все медиа',
    'Aspect ratio': 'Соотношение сторон',
    'Describe the image you want to create':
      'Опишите изображение, которое хотите создать',
    'Describe the video you want to create':
      'Опишите видео, которое хотите создать',
    'Download video': 'Скачать видео',
    'Duration in seconds': 'Длительность в секундах',
    'Failed to load media history': 'Не удалось загрузить историю медиа',
    'Failed to poll video task': 'Не удалось получить статус задачи видео',
    Generate: 'Создать',
    'Generate audio': 'Создать аудио',
    Generating: 'Создание',
    'Generated images will appear here': 'Созданные изображения появятся здесь',
    'Generated video will appear here': 'Созданное видео появится здесь',
    'Media generation failed': 'Не удалось создать медиа',
    'No media generated in the last 24 hours':
      'За последние 24 часа ничего не создано',
    Processing: 'Обработка',
    Quality: 'Качество',
    'Reference audio': 'Референсное аудио',
    'Reference images': 'Референсные изображения',
    'Reference limit reached': 'Достигнут лимит референсов',
    'Reference videos': 'Референсные видео',
    'Recent 24 hours': 'Последние 24 часа',
    'Refresh history': 'Обновить историю',
    'Remove reference': 'Удалить референс',
    Resolution: 'Разрешение',
    Size: 'Размер',
    hd: 'Высокое качество',
    standard: 'Стандартное',
    'Video generation failed': 'Не удалось создать видео',
    Watermark: 'Водяной знак',
  },
  vi: {
    'Add audio': 'Thêm âm thanh',
    'Add images': 'Thêm hình ảnh',
    'Add videos': 'Thêm video',
    'All media': 'Tất cả nội dung đa phương tiện',
    'Aspect ratio': 'Tỷ lệ khung hình',
    'Describe the image you want to create': 'Mô tả hình ảnh bạn muốn tạo',
    'Describe the video you want to create': 'Mô tả video bạn muốn tạo',
    'Download video': 'Tải video xuống',
    'Duration in seconds': 'Thời lượng tính bằng giây',
    'Failed to load media history':
      'Không thể tải lịch sử nội dung đa phương tiện',
    'Failed to poll video task': 'Không thể lấy tiến độ tác vụ video',
    Generate: 'Tạo',
    'Generate audio': 'Tạo âm thanh',
    Generating: 'Đang tạo',
    'Generated images will appear here': 'Hình ảnh đã tạo sẽ xuất hiện tại đây',
    'Generated video will appear here': 'Video đã tạo sẽ xuất hiện tại đây',
    'Media generation failed': 'Tạo nội dung đa phương tiện thất bại',
    'No media generated in the last 24 hours':
      'Không có nội dung nào được tạo trong 24 giờ qua',
    Processing: 'Đang xử lý',
    Quality: 'Chất lượng',
    'Reference audio': 'Âm thanh tham chiếu',
    'Reference images': 'Hình ảnh tham chiếu',
    'Reference limit reached': 'Đã đạt giới hạn nội dung tham chiếu',
    'Reference videos': 'Video tham chiếu',
    'Recent 24 hours': '24 giờ gần đây',
    'Refresh history': 'Làm mới lịch sử',
    'Remove reference': 'Xóa nội dung tham chiếu',
    Resolution: 'Độ phân giải',
    Size: 'Kích thước',
    hd: 'Chất lượng cao',
    standard: 'Tiêu chuẩn',
    'Video generation failed': 'Tạo video thất bại',
    Watermark: 'Hình mờ',
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
