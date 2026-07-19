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
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2)}\n`
}

const newKeys = {
  en: {
    Background: 'Background',
    Base64: 'Base64',
    'Exact size': 'Exact size',
    Height: 'Height',
    'Image count': 'Image count',
    'Response format': 'Response format',
    Width: 'Width',
    'Width and height must be multiples of 16':
      'Width and height must be multiples of 16',
    high: 'High',
    low: 'Low',
    medium: 'Medium',
    opaque: 'Opaque',
    transparent: 'Transparent',
    'Backfill missing costs': 'Backfill missing costs',
    'By channel': 'By channel',
    'By group': 'By group',
    'By model': 'By model',
    'By user': 'By user',
    'Cost coverage': 'Cost coverage',
    'Cost rule saved': 'Cost rule saved',
    'Converted cost': 'Converted cost',
    'Date range': 'Date range',
    'Failed to recalculate profit records':
      'Failed to recalculate profit records',
    'Failed to save cost rule': 'Failed to save cost rule',
    'Gross profit': 'Gross profit',
    'Last 7 days': 'Last 7 days',
    'Last 30 days': 'Last 30 days',
    'Last 90 days': 'Last 90 days',
    'No profit data': 'No profit data',
    'Profit Analysis': 'Profit Analysis',
    'Profit breakdown': 'Profit breakdown',
    'Profit margin': 'Profit margin',
    'Profit records recalculated': 'Profit records recalculated',
    'Purchase discount': 'Purchase discount',
    'Purchase exchange rate': 'Purchase exchange rate',
    Recalculate: 'Recalculate',
    'Review revenue, upstream cost, and gross margin.':
      'Review revenue, upstream cost, and gross margin.',
    'Sales revenue': 'Sales revenue',
    'Save cost rule': 'Save cost rule',
    'Upstream cost': 'Upstream cost',
    'Upstream price (USD)': 'Upstream price (USD)',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} unpriced billing records are excluded from profit margin.',
    'Group model prices': 'Group model prices',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.',
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
    Background: '背景',
    Base64: 'Base64',
    'Exact size': '精确尺寸',
    Height: '高度',
    'Image count': '生成数量',
    'Response format': '返回格式',
    Width: '宽度',
    'Width and height must be multiples of 16': '宽高必须为 16 的倍数',
    high: '高',
    low: '低',
    medium: '中',
    opaque: '不透明',
    transparent: '透明',
    'Backfill missing costs': '补算缺失成本',
    'By channel': '按渠道',
    'By group': '按分组',
    'By model': '按模型',
    'By user': '按用户',
    'Cost coverage': '成本覆盖率',
    'Cost rule saved': '进货价规则已保存',
    'Converted cost': '折算成本',
    'Date range': '日期范围',
    'Failed to recalculate profit records': '重新计算利润记录失败',
    'Failed to save cost rule': '保存进货价规则失败',
    'Gross profit': '毛利润',
    'Last 7 days': '最近 7 天',
    'Last 30 days': '最近 30 天',
    'Last 90 days': '最近 90 天',
    'No profit data': '暂无利润数据',
    'Profit Analysis': '利润分析',
    'Profit breakdown': '利润明细',
    'Profit margin': '利润率',
    'Profit records recalculated': '利润记录已重新计算',
    'Purchase discount': '进货折扣',
    'Purchase exchange rate': '进货汇率',
    Recalculate: '重新计算',
    'Review revenue, upstream cost, and gross margin.':
      '查看销售收入、上游成本和毛利率。',
    'Sales revenue': '销售收入',
    'Save cost rule': '保存进货价',
    'Upstream cost': '上游成本',
    'Upstream price (USD)': '上游标价（美元）',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} 条未配置进货价的计费记录未计入利润率。',
    'Group model prices': '分组模型价格',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      '嵌套 JSON：分组 → 模型 → 固定价格。使用该分组发起请求时，将覆盖全局模型固定价格。',
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
    Background: '背景',
    Base64: 'Base64',
    'Exact size': '精確尺寸',
    Height: '高度',
    'Image count': '產生數量',
    'Response format': '回傳格式',
    Width: '寬度',
    'Width and height must be multiples of 16': '寬高必須為 16 的倍數',
    high: '高',
    low: '低',
    medium: '中',
    opaque: '不透明',
    transparent: '透明',
    'Backfill missing costs': '補算缺失成本',
    'By channel': '按渠道',
    'By group': '按分組',
    'By model': '按模型',
    'By user': '按使用者',
    'Cost coverage': '成本覆蓋率',
    'Cost rule saved': '進貨價規則已儲存',
    'Converted cost': '換算成本',
    'Date range': '日期範圍',
    'Failed to recalculate profit records': '重新計算利潤記錄失敗',
    'Failed to save cost rule': '儲存進貨價規則失敗',
    'Gross profit': '毛利潤',
    'Last 7 days': '最近 7 天',
    'Last 30 days': '最近 30 天',
    'Last 90 days': '最近 90 天',
    'No profit data': '暫無利潤資料',
    'Profit Analysis': '利潤分析',
    'Profit breakdown': '利潤明細',
    'Profit margin': '利潤率',
    'Profit records recalculated': '利潤記錄已重新計算',
    'Purchase discount': '進貨折扣',
    'Purchase exchange rate': '進貨匯率',
    Recalculate: '重新計算',
    'Review revenue, upstream cost, and gross margin.':
      '查看銷售收入、上游成本和毛利率。',
    'Sales revenue': '銷售收入',
    'Save cost rule': '儲存進貨價',
    'Upstream cost': '上游成本',
    'Upstream price (USD)': '上游標價（美元）',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} 筆未設定進貨價的計費記錄未計入利潤率。',
    'Group model prices': '分組模型價格',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      '巢狀 JSON：分組 → 模型 → 固定價格。使用該分組發出請求時，將覆寫全域模型固定價格。',
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
    Background: 'Arrière-plan',
    Base64: 'Base64',
    'Exact size': 'Taille exacte',
    Height: 'Hauteur',
    'Image count': 'Nombre d’images',
    'Response format': 'Format de réponse',
    Width: 'Largeur',
    'Width and height must be multiples of 16':
      'La largeur et la hauteur doivent être des multiples de 16',
    high: 'Élevée',
    low: 'Faible',
    medium: 'Moyenne',
    opaque: 'Opaque',
    transparent: 'Transparent',
    'Backfill missing costs': 'Compléter les coûts manquants',
    'By channel': 'Par canal',
    'By group': 'Par groupe',
    'By model': 'Par modèle',
    'By user': 'Par utilisateur',
    'Cost coverage': 'Couverture des coûts',
    'Cost rule saved': 'Règle de coût enregistrée',
    'Converted cost': 'Coût converti',
    'Date range': 'Période',
    'Failed to recalculate profit records': 'Échec du recalcul des bénéfices',
    'Failed to save cost rule': 'Échec de l’enregistrement du coût',
    'Gross profit': 'Bénéfice brut',
    'Last 7 days': '7 derniers jours',
    'Last 30 days': '30 derniers jours',
    'Last 90 days': '90 derniers jours',
    'No profit data': 'Aucune donnée de bénéfice',
    'Profit Analysis': 'Analyse des bénéfices',
    'Profit breakdown': 'Détail des bénéfices',
    'Profit margin': 'Marge bénéficiaire',
    'Profit records recalculated': 'Bénéfices recalculés',
    'Purchase discount': 'Remise d’achat',
    'Purchase exchange rate': 'Taux de change d’achat',
    Recalculate: 'Recalculer',
    'Review revenue, upstream cost, and gross margin.':
      'Consultez les revenus, les coûts en amont et la marge brute.',
    'Sales revenue': 'Chiffre d’affaires',
    'Save cost rule': 'Enregistrer le coût',
    'Upstream cost': 'Coût en amont',
    'Upstream price (USD)': 'Prix fournisseur (USD)',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} enregistrements sans coût sont exclus de la marge.',
    'Group model prices': 'Prix des modèles par groupe',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      'JSON imbriqué : groupe → modèle → prix fixe. Remplace le prix fixe global du modèle pour les requêtes utilisant ce groupe.',
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
    Background: '背景',
    Base64: 'Base64',
    'Exact size': '正確なサイズ',
    Height: '高さ',
    'Image count': '生成枚数',
    'Response format': '応答形式',
    Width: '幅',
    'Width and height must be multiples of 16':
      '幅と高さは 16 の倍数にしてください',
    high: '高',
    low: '低',
    medium: '中',
    opaque: '不透明',
    transparent: '透明',
    'Backfill missing costs': '未計上コストを補完',
    'By channel': 'チャネル別',
    'By group': 'グループ別',
    'By model': 'モデル別',
    'By user': 'ユーザー別',
    'Cost coverage': 'コスト設定率',
    'Cost rule saved': '仕入価格ルールを保存しました',
    'Converted cost': '換算コスト',
    'Date range': '期間',
    'Failed to recalculate profit records': '利益の再計算に失敗しました',
    'Failed to save cost rule': '仕入価格の保存に失敗しました',
    'Gross profit': '粗利益',
    'Last 7 days': '過去7日間',
    'Last 30 days': '過去30日間',
    'Last 90 days': '過去90日間',
    'No profit data': '利益データがありません',
    'Profit Analysis': '利益分析',
    'Profit breakdown': '利益内訳',
    'Profit margin': '利益率',
    'Profit records recalculated': '利益を再計算しました',
    'Purchase discount': '仕入割引率',
    'Purchase exchange rate': '仕入為替レート',
    Recalculate: '再計算',
    'Review revenue, upstream cost, and gross margin.':
      '売上、上流コスト、粗利益率を確認します。',
    'Sales revenue': '売上',
    'Save cost rule': '仕入価格を保存',
    'Upstream cost': '上流コスト',
    'Upstream price (USD)': '上流価格（USD）',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '仕入価格未設定の{{count}}件は利益率から除外されています。',
    'Group model prices': 'グループ別モデル価格',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      'ネストされた JSON：グループ → モデル → 固定価格。このグループを使用するリクエストでは、グローバルなモデル固定価格を上書きします。',
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
    Background: 'Фон',
    Base64: 'Base64',
    'Exact size': 'Точный размер',
    Height: 'Высота',
    'Image count': 'Количество изображений',
    'Response format': 'Формат ответа',
    Width: 'Ширина',
    'Width and height must be multiples of 16':
      'Ширина и высота должны быть кратны 16',
    high: 'Высокое',
    low: 'Низкое',
    medium: 'Среднее',
    opaque: 'Непрозрачный',
    transparent: 'Прозрачный',
    'Backfill missing costs': 'Рассчитать недостающие затраты',
    'By channel': 'По каналам',
    'By group': 'По группам',
    'By model': 'По моделям',
    'By user': 'По пользователям',
    'Cost coverage': 'Покрытие затрат',
    'Cost rule saved': 'Правило затрат сохранено',
    'Converted cost': 'Пересчитанная стоимость',
    'Date range': 'Период',
    'Failed to recalculate profit records': 'Не удалось пересчитать прибыль',
    'Failed to save cost rule': 'Не удалось сохранить стоимость',
    'Gross profit': 'Валовая прибыль',
    'Last 7 days': 'Последние 7 дней',
    'Last 30 days': 'Последние 30 дней',
    'Last 90 days': 'Последние 90 дней',
    'No profit data': 'Нет данных о прибыли',
    'Profit Analysis': 'Анализ прибыли',
    'Profit breakdown': 'Детализация прибыли',
    'Profit margin': 'Маржа прибыли',
    'Profit records recalculated': 'Прибыль пересчитана',
    'Purchase discount': 'Скидка закупки',
    'Purchase exchange rate': 'Курс закупки',
    Recalculate: 'Пересчитать',
    'Review revenue, upstream cost, and gross margin.':
      'Просмотр выручки, затрат и валовой маржи.',
    'Sales revenue': 'Выручка',
    'Save cost rule': 'Сохранить стоимость',
    'Upstream cost': 'Затраты поставщика',
    'Upstream price (USD)': 'Цена поставщика (USD)',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} записей без стоимости исключены из маржи.',
    'Group model prices': 'Цены моделей по группам',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      'Вложенный JSON: группа → модель → фиксированная цена. Переопределяет глобальную фиксированную цену модели для запросов этой группы.',
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
    Background: 'Nền',
    Base64: 'Base64',
    'Exact size': 'Kích thước chính xác',
    Height: 'Chiều cao',
    'Image count': 'Số lượng hình ảnh',
    'Response format': 'Định dạng phản hồi',
    Width: 'Chiều rộng',
    'Width and height must be multiples of 16':
      'Chiều rộng và chiều cao phải là bội số của 16',
    high: 'Cao',
    low: 'Thấp',
    medium: 'Trung bình',
    opaque: 'Đục',
    transparent: 'Trong suốt',
    'Backfill missing costs': 'Tính bù chi phí còn thiếu',
    'By channel': 'Theo kênh',
    'By group': 'Theo nhóm',
    'By model': 'Theo mô hình',
    'By user': 'Theo người dùng',
    'Cost coverage': 'Độ phủ chi phí',
    'Cost rule saved': 'Đã lưu quy tắc chi phí',
    'Converted cost': 'Chi phí quy đổi',
    'Date range': 'Khoảng thời gian',
    'Failed to recalculate profit records': 'Không thể tính lại lợi nhuận',
    'Failed to save cost rule': 'Không thể lưu chi phí',
    'Gross profit': 'Lợi nhuận gộp',
    'Last 7 days': '7 ngày qua',
    'Last 30 days': '30 ngày qua',
    'Last 90 days': '90 ngày qua',
    'No profit data': 'Không có dữ liệu lợi nhuận',
    'Profit Analysis': 'Phân tích lợi nhuận',
    'Profit breakdown': 'Chi tiết lợi nhuận',
    'Profit margin': 'Biên lợi nhuận',
    'Profit records recalculated': 'Đã tính lại lợi nhuận',
    'Purchase discount': 'Chiết khấu mua',
    'Purchase exchange rate': 'Tỷ giá mua',
    Recalculate: 'Tính lại',
    'Review revenue, upstream cost, and gross margin.':
      'Xem doanh thu, chi phí thượng nguồn và biên lợi nhuận gộp.',
    'Sales revenue': 'Doanh thu',
    'Save cost rule': 'Lưu giá vốn',
    'Upstream cost': 'Chi phí thượng nguồn',
    'Upstream price (USD)': 'Giá thượng nguồn (USD)',
    '{{count}} unpriced billing records are excluded from profit margin.':
      '{{count}} bản ghi chưa có giá vốn bị loại khỏi biên lợi nhuận.',
    'Group model prices': 'Giá mô hình theo nhóm',
    'Nested JSON: group → model → fixed price. Overrides the global fixed model price for requests using that group.':
      'JSON lồng nhau: nhóm → mô hình → giá cố định. Ghi đè giá cố định toàn cục của mô hình cho các yêu cầu sử dụng nhóm này.',
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

newKeys.en['KYY Video'] = 'KYY Video'
newKeys.zh['KYY Video'] = '客易云视频'
newKeys['zh-TW']['KYY Video'] = '客易雲影片'
newKeys.fr['KYY Video'] = 'Vidéo KYY'
newKeys.ja['KYY Video'] = 'KYY 動画'
newKeys.ru['KYY Video'] = 'Видео KYY'
newKeys.vi['KYY Video'] = 'Video KYY'

Object.assign(newKeys.en, {
  'Failed to reset profit analysis data':
    'Failed to reset profit analysis data',
  'Profit analysis data reset': 'Profit analysis data reset',
  'Purchase cost': 'Purchase cost',
  'Purchase cost (CNY)': 'Purchase cost (CNY)',
  'Reset analysis data': 'Reset analysis data',
  'Reset profit analysis data?': 'Reset profit analysis data?',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.',
})
Object.assign(newKeys.zh, {
  'Failed to reset profit analysis data': '重置利润分析数据失败',
  'Profit analysis data reset': '利润分析数据已重置',
  'Purchase cost': '进货成本',
  'Purchase cost (CNY)': '进货成本（人民币）',
  'Reset analysis data': '重置分析数据',
  'Reset profit analysis data?': '确定重置利润分析数据？',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    '这会清空全部利润分析记录，并从现在开始重新统计。消费日志和模型进货价规则不会被删除。',
})
Object.assign(newKeys['zh-TW'], {
  'Failed to reset profit analysis data': '重設利潤分析資料失敗',
  'Profit analysis data reset': '利潤分析資料已重設',
  'Purchase cost': '進貨成本',
  'Purchase cost (CNY)': '進貨成本（人民幣）',
  'Reset analysis data': '重設分析資料',
  'Reset profit analysis data?': '確定重設利潤分析資料？',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    '這會清空全部利潤分析記錄，並從現在開始重新統計。使用記錄和模型進貨價規則不會被刪除。',
})
Object.assign(newKeys.fr, {
  'Failed to reset profit analysis data':
    'Échec de la réinitialisation des données de bénéfice',
  'Profit analysis data reset': 'Données de bénéfice réinitialisées',
  'Purchase cost': "Coût d'achat",
  'Purchase cost (CNY)': "Coût d'achat (CNY)",
  'Reset analysis data': "Réinitialiser les données d'analyse",
  'Reset profit analysis data?':
    'Réinitialiser les données d’analyse des bénéfices ?',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    "Tous les enregistrements de bénéfice seront effacés et le suivi reprendra maintenant. Les journaux d'utilisation et les règles de coût d'achat seront conservés.",
})
Object.assign(newKeys.ja, {
  'Failed to reset profit analysis data':
    '利益分析データのリセットに失敗しました',
  'Profit analysis data reset': '利益分析データをリセットしました',
  'Purchase cost': '仕入原価',
  'Purchase cost (CNY)': '仕入原価（CNY）',
  'Reset analysis data': '分析データをリセット',
  'Reset profit analysis data?': '利益分析データをリセットしますか？',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    'すべての利益分析レコードを消去し、現在から集計を再開します。利用ログと仕入価格ルールは削除されません。',
})
Object.assign(newKeys.ru, {
  'Failed to reset profit analysis data':
    'Не удалось сбросить данные анализа прибыли',
  'Profit analysis data reset': 'Данные анализа прибыли сброшены',
  'Purchase cost': 'Закупочная стоимость',
  'Purchase cost (CNY)': 'Закупочная стоимость (CNY)',
  'Reset analysis data': 'Сбросить данные анализа',
  'Reset profit analysis data?': 'Сбросить данные анализа прибыли?',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    'Все записи анализа прибыли будут удалены, а учет начнется заново с текущего момента. Журналы использования и правила закупочной стоимости сохранятся.',
})
Object.assign(newKeys.vi, {
  'Failed to reset profit analysis data':
    'Không thể đặt lại dữ liệu phân tích lợi nhuận',
  'Profit analysis data reset': 'Đã đặt lại dữ liệu phân tích lợi nhuận',
  'Purchase cost': 'Chi phí mua',
  'Purchase cost (CNY)': 'Chi phí mua (CNY)',
  'Reset analysis data': 'Đặt lại dữ liệu phân tích',
  'Reset profit analysis data?': 'Đặt lại dữ liệu phân tích lợi nhuận?',
  'This clears all profit analysis records and starts tracking from now. Usage logs and purchase cost rules are not deleted.':
    'Thao tác này xóa toàn bộ bản ghi phân tích lợi nhuận và bắt đầu theo dõi lại từ bây giờ. Nhật ký sử dụng và quy tắc chi phí mua sẽ không bị xóa.',
})

Object.assign(newKeys.en, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.',
})
Object.assign(newKeys.zh, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    '这会从现在开始新的利润分析周期，之前的分析记录将不再计入。消费日志和模型进货价规则不会被删除。',
})
Object.assign(newKeys['zh-TW'], {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    '這會從現在開始新的利潤分析週期，之前的分析記錄將不再計入。使用記錄和模型進貨價規則不會被刪除。',
})
Object.assign(newKeys.fr, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    "Une nouvelle période d'analyse commence maintenant. Les enregistrements antérieurs ne seront plus inclus. Les journaux d'utilisation et les règles de coût d'achat sont conservés.",
})
Object.assign(newKeys.ja, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    '現在から新しい利益分析期間を開始し、以前の分析レコードは集計対象外になります。利用ログと仕入価格ルールは削除されません。',
})
Object.assign(newKeys.ru, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    'С текущего момента начинается новый период анализа прибыли. Предыдущие записи больше не учитываются. Журналы использования и правила закупочной стоимости сохраняются.',
})
Object.assign(newKeys.vi, {
  'This starts a new profit analysis period from now. Earlier analysis records will no longer be included. Usage logs and purchase cost rules are not deleted.':
    'Một kỳ phân tích lợi nhuận mới sẽ bắt đầu từ bây giờ. Các bản ghi trước đó sẽ không còn được tính. Nhật ký sử dụng và quy tắc chi phí mua vẫn được giữ lại.',
})

Object.assign(newKeys.en, {
  'All time': 'All time',
  'Cost Configuration': 'Cost Configuration',
  'Filter by model': 'Filter by model',
})
Object.assign(newKeys.zh, {
  'All time': '全部时间',
  'Cost Configuration': '成本配置',
  'Filter by model': '按模型筛选',
})
Object.assign(newKeys['zh-TW'], {
  'All time': '全部時間',
  'Cost Configuration': '成本設定',
  'Filter by model': '按模型篩選',
})
Object.assign(newKeys.fr, {
  'All time': 'Toutes les périodes',
  'Cost Configuration': 'Configuration des coûts',
  'Filter by model': 'Filtrer par modèle',
})
Object.assign(newKeys.ja, {
  'All time': '全期間',
  'Cost Configuration': 'コスト設定',
  'Filter by model': 'モデルで絞り込み',
})
Object.assign(newKeys.ru, {
  'All time': 'За всё время',
  'Cost Configuration': 'Настройка стоимости',
  'Filter by model': 'Фильтр по модели',
})
Object.assign(newKeys.vi, {
  'All time': 'Toàn bộ thời gian',
  'Cost Configuration': 'Cấu hình chi phí',
  'Filter by model': 'Lọc theo mô hình',
})

Object.assign(newKeys.en, {
  'Add gift tier': 'Add gift tier',
  'Admin consumption': 'Admin consumption',
  'Admin cost': 'Admin cost',
  'Balance source': 'Balance source',
  'Balance type: Gift balance': 'Balance type: Gift balance',
  'Configure promotional gifts for exact recharge amounts':
    'Configure promotional gifts for exact recharge amounts',
  'Edit gift tier': 'Edit gift tier',
  Gift: 'Gift',
  'Gift Amount (USD)': 'Gift Amount (USD)',
  'Gift Quota': 'Gift Quota',
  'Gift balance': 'Gift balance',
  'Gift balance is promotional credit and produces no revenue.':
    'Gift balance is promotional credit and produces no revenue.',
  'Gift balance produces no recognized revenue when spent.':
    'Gift balance produces no recognized revenue when spent.',
  'Gift consumption': 'Gift consumption',
  'Gift cost': 'Gift cost',
  'Gift quota map by exact recharge amount (JSON object)':
    'Gift quota map by exact recharge amount (JSON object)',
  'Gift {{bonus}} · Total {{total}}': 'Gift {{bonus}} · Total {{total}}',
  'Legacy unattributed': 'Legacy unattributed',
  'Legacy unattributed consumption': 'Legacy unattributed consumption',
  'No gift tiers configured.': 'No gift tiers configured.',
  'Nominal consumption': 'Nominal consumption',
  'Optional promotional quota credited in addition to the paid quota.':
    'Optional promotional quota credited in addition to the paid quota.',
  'Paid balance': 'Paid balance',
  Principal: 'Principal',
  'Recharge gift': 'Recharge gift',
  'Recognized revenue': 'Recognized revenue',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.',
  'Redemption codes always create promotional gift balance and never paid balance.':
    'Redemption codes always create promotional gift balance and never paid balance.',
  'Set the gift quota credited for an exact recharge amount.':
    'Set the gift quota credited for an exact recharge amount.',
  'The gift applies only when this exact amount is recharged.':
    'The gift applies only when this exact amount is recharged.',
  'Total credited': 'Total credited',
  'Use paid balance only after confirming payment was received.':
    'Use paid balance only after confirming payment was received.',
  'e.g., 50000': 'e.g., 50000',
  gift: 'gift',
})
Object.assign(newKeys.zh, {
  'Add gift tier': '添加赠送档位',
  'Admin consumption': '管理员内部消费',
  'Admin cost': '管理员使用成本',
  'Balance source': '余额来源',
  'Balance type: Gift balance': '余额类型：赠送余额',
  'Configure promotional gifts for exact recharge amounts':
    '按精确充值金额配置赠送额度',
  'Edit gift tier': '编辑赠送档位',
  Gift: '赠送',
  'Gift Amount (USD)': '赠送金额（美元）',
  'Gift Quota': '赠送额度',
  'Gift balance': '赠送余额',
  'Gift balance is promotional credit and produces no revenue.':
    '赠送余额属于营销额度，不产生收入。',
  'Gift balance produces no recognized revenue when spent.':
    '赠送余额消费时不确认任何收入。',
  'Gift consumption': '赠送额度消费',
  'Gift cost': '赠送额度成本',
  'Gift quota map by exact recharge amount (JSON object)':
    '按精确充值金额配置赠送额度（JSON 对象）',
  'Gift {{bonus}} · Total {{total}}': '赠送 {{bonus}} · 总到账 {{total}}',
  'Legacy unattributed': '历史未归因',
  'Legacy unattributed consumption': '历史未归因消费',
  'No gift tiers configured.': '尚未配置赠送档位。',
  'Nominal consumption': '名义消费额',
  'Optional promotional quota credited in addition to the paid quota.':
    '可选。除付费额度外额外到账的营销赠送额度。',
  'Paid balance': '付费余额',
  Principal: '充值本金',
  'Recharge gift': '充值赠送',
  'Recognized revenue': '确认收入',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    '兑换码到账的是赠送余额，属于营销额度，不计为付费充值。',
  'Redemption codes always create promotional gift balance and never paid balance.':
    '兑换码固定生成营销赠送余额，不会生成付费余额。',
  'Set the gift quota credited for an exact recharge amount.':
    '设置精确充值金额对应的赠送额度。',
  'The gift applies only when this exact amount is recharged.':
    '仅充值该精确金额时发放赠送额度。',
  'Total credited': '总到账',
  'Use paid balance only after confirming payment was received.':
    '仅在确认已收到款项后选择付费余额。',
  'e.g., 50000': '例如 50000',
  gift: '赠送',
})
Object.assign(newKeys['zh-TW'], {
  'Add gift tier': '新增贈送檔位',
  'Admin consumption': '管理員內部消費',
  'Admin cost': '管理員使用成本',
  'Balance source': '餘額來源',
  'Balance type: Gift balance': '餘額類型：贈送餘額',
  'Configure promotional gifts for exact recharge amounts':
    '依精確儲值金額設定贈送額度',
  'Edit gift tier': '編輯贈送檔位',
  Gift: '贈送',
  'Gift Amount (USD)': '贈送金額（美元）',
  'Gift Quota': '贈送額度',
  'Gift balance': '贈送餘額',
  'Gift balance is promotional credit and produces no revenue.':
    '贈送餘額屬於行銷額度，不產生收入。',
  'Gift balance produces no recognized revenue when spent.':
    '贈送餘額消費時不確認任何收入。',
  'Gift consumption': '贈送額度消費',
  'Gift cost': '贈送額度成本',
  'Gift quota map by exact recharge amount (JSON object)':
    '依精確儲值金額設定贈送額度（JSON 物件）',
  'Gift {{bonus}} · Total {{total}}': '贈送 {{bonus}} · 總入帳 {{total}}',
  'Legacy unattributed': '歷史未歸因',
  'Legacy unattributed consumption': '歷史未歸因消費',
  'No gift tiers configured.': '尚未設定贈送檔位。',
  'Nominal consumption': '名義消費額',
  'Optional promotional quota credited in addition to the paid quota.':
    '選填。除付費額度外額外入帳的行銷贈送額度。',
  'Paid balance': '付費餘額',
  Principal: '儲值本金',
  'Recharge gift': '儲值贈送',
  'Recognized revenue': '確認收入',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    '兌換碼入帳的是贈送餘額，屬於行銷額度，不計為付費儲值。',
  'Redemption codes always create promotional gift balance and never paid balance.':
    '兌換碼固定產生行銷贈送餘額，不會產生付費餘額。',
  'Set the gift quota credited for an exact recharge amount.':
    '設定精確儲值金額對應的贈送額度。',
  'The gift applies only when this exact amount is recharged.':
    '僅儲值此精確金額時發放贈送額度。',
  'Total credited': '總入帳',
  'Use paid balance only after confirming payment was received.':
    '僅在確認已收到款項後選擇付費餘額。',
  'e.g., 50000': '例如 50000',
  gift: '贈送',
})
Object.assign(newKeys.fr, {
  'Add gift tier': 'Ajouter un palier bonus',
  'Admin consumption': 'Consommation administrateur',
  'Admin cost': 'Coût administrateur',
  'Balance source': 'Origine du solde',
  'Balance type: Gift balance': 'Type de solde : solde offert',
  'Configure promotional gifts for exact recharge amounts':
    'Configurer les bonus pour des montants de recharge exacts',
  'Edit gift tier': 'Modifier le palier bonus',
  Gift: 'Bonus',
  'Gift Amount (USD)': 'Montant offert (USD)',
  'Gift Quota': 'Quota offert',
  'Gift balance': 'Solde offert',
  'Gift balance is promotional credit and produces no revenue.':
    'Le solde offert est un crédit promotionnel et ne génère aucun revenu.',
  'Gift balance produces no recognized revenue when spent.':
    'La dépense du solde offert ne génère aucun revenu comptabilisé.',
  'Gift consumption': 'Consommation offerte',
  'Gift cost': 'Coût du solde offert',
  'Gift quota map by exact recharge amount (JSON object)':
    'Quotas offerts par montant exact de recharge (objet JSON)',
  'Gift {{bonus}} · Total {{total}}': 'Bonus {{bonus}} · Total {{total}}',
  'Legacy unattributed': 'Historique non attribué',
  'Legacy unattributed consumption': 'Consommation historique non attribuée',
  'No gift tiers configured.': 'Aucun palier bonus configuré.',
  'Nominal consumption': 'Consommation nominale',
  'Optional promotional quota credited in addition to the paid quota.':
    'Quota promotionnel facultatif crédité en plus du quota payé.',
  'Paid balance': 'Solde payé',
  Principal: 'Montant principal',
  'Recharge gift': 'Bonus de recharge',
  'Recognized revenue': 'Revenu comptabilisé',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    "Les codes ajoutent un solde offert promotionnel qui n'est pas compté comme recharge payée.",
  'Redemption codes always create promotional gift balance and never paid balance.':
    'Les codes créent toujours un solde promotionnel offert, jamais un solde payé.',
  'Set the gift quota credited for an exact recharge amount.':
    'Définissez le quota offert pour un montant de recharge exact.',
  'The gift applies only when this exact amount is recharged.':
    "Le bonus s'applique uniquement à ce montant de recharge exact.",
  'Total credited': 'Total crédité',
  'Use paid balance only after confirming payment was received.':
    'Utilisez le solde payé uniquement après confirmation de la réception du paiement.',
  'e.g., 50000': 'p. ex. 50000',
  gift: 'offert',
})
Object.assign(newKeys.ja, {
  'Add gift tier': '特典枠を追加',
  'Admin consumption': '管理者の内部消費',
  'Admin cost': '管理者利用コスト',
  'Balance source': '残高の種別',
  'Balance type: Gift balance': '残高タイプ：特典残高',
  'Configure promotional gifts for exact recharge amounts':
    '指定したチャージ金額ごとに特典を設定',
  'Edit gift tier': '特典枠を編集',
  Gift: '特典',
  'Gift Amount (USD)': '特典額（USD）',
  'Gift Quota': '特典クォータ',
  'Gift balance': '特典残高',
  'Gift balance is promotional credit and produces no revenue.':
    '特典残高は販促クレジットであり、売上にはなりません。',
  'Gift balance produces no recognized revenue when spent.':
    '特典残高の利用は売上として認識されません。',
  'Gift consumption': '特典残高の消費',
  'Gift cost': '特典利用コスト',
  'Gift quota map by exact recharge amount (JSON object)':
    '指定チャージ金額ごとの特典クォータ（JSON オブジェクト）',
  'Gift {{bonus}} · Total {{total}}': '特典 {{bonus}} · 合計 {{total}}',
  'Legacy unattributed': '過去分・未帰属',
  'Legacy unattributed consumption': '過去の未帰属消費',
  'No gift tiers configured.': '特典枠は設定されていません。',
  'Nominal consumption': '名目消費額',
  'Optional promotional quota credited in addition to the paid quota.':
    '任意。支払済みクォータに加えて付与する販促クォータです。',
  'Paid balance': '支払済み残高',
  Principal: 'チャージ元金',
  'Recharge gift': 'チャージ特典',
  'Recognized revenue': '認識売上',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    '引換コードは特典残高を追加します。販促クレジットのため、有料チャージには含まれません。',
  'Redemption codes always create promotional gift balance and never paid balance.':
    '引換コードは常に販促用の特典残高を作成し、支払済み残高は作成しません。',
  'Set the gift quota credited for an exact recharge amount.':
    '指定したチャージ金額に付与する特典クォータを設定します。',
  'The gift applies only when this exact amount is recharged.':
    'この指定金額をチャージした場合にのみ特典が適用されます。',
  'Total credited': '合計付与額',
  'Use paid balance only after confirming payment was received.':
    '入金確認後にのみ支払済み残高を選択してください。',
  'e.g., 50000': '例：50000',
  gift: '特典',
})
Object.assign(newKeys.ru, {
  'Add gift tier': 'Добавить уровень бонуса',
  'Admin consumption': 'Внутреннее потребление администратора',
  'Admin cost': 'Затраты администратора',
  'Balance source': 'Источник баланса',
  'Balance type: Gift balance': 'Тип баланса: подарочный',
  'Configure promotional gifts for exact recharge amounts':
    'Настройте бонусы для точных сумм пополнения',
  'Edit gift tier': 'Изменить уровень бонуса',
  Gift: 'Бонус',
  'Gift Amount (USD)': 'Сумма бонуса (USD)',
  'Gift Quota': 'Подарочная квота',
  'Gift balance': 'Подарочный баланс',
  'Gift balance is promotional credit and produces no revenue.':
    'Подарочный баланс является рекламным кредитом и не приносит дохода.',
  'Gift balance produces no recognized revenue when spent.':
    'Расход подарочного баланса не создаёт признанного дохода.',
  'Gift consumption': 'Расход подарочного баланса',
  'Gift cost': 'Стоимость подарочного баланса',
  'Gift quota map by exact recharge amount (JSON object)':
    'Подарочные квоты по точной сумме пополнения (объект JSON)',
  'Gift {{bonus}} · Total {{total}}': 'Бонус {{bonus}} · Итого {{total}}',
  'Legacy unattributed': 'Историческое без атрибуции',
  'Legacy unattributed consumption': 'Историческое потребление без атрибуции',
  'No gift tiers configured.': 'Уровни бонусов не настроены.',
  'Nominal consumption': 'Номинальное потребление',
  'Optional promotional quota credited in addition to the paid quota.':
    'Необязательная рекламная квота сверх оплаченной квоты.',
  'Paid balance': 'Оплаченный баланс',
  Principal: 'Основная сумма',
  'Recharge gift': 'Бонус за пополнение',
  'Recognized revenue': 'Признанный доход',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    'Коды пополняют подарочный рекламный баланс, который не считается платным пополнением.',
  'Redemption codes always create promotional gift balance and never paid balance.':
    'Коды всегда создают подарочный рекламный баланс, а не оплаченный.',
  'Set the gift quota credited for an exact recharge amount.':
    'Задайте подарочную квоту для точной суммы пополнения.',
  'The gift applies only when this exact amount is recharged.':
    'Бонус применяется только при пополнении на эту точную сумму.',
  'Total credited': 'Всего зачислено',
  'Use paid balance only after confirming payment was received.':
    'Выбирайте оплаченный баланс только после подтверждения получения платежа.',
  'e.g., 50000': 'например, 50000',
  gift: 'бонус',
})
Object.assign(newKeys.vi, {
  'Add gift tier': 'Thêm mức tặng',
  'Admin consumption': 'Mức sử dụng nội bộ của quản trị viên',
  'Admin cost': 'Chi phí quản trị viên',
  'Balance source': 'Nguồn số dư',
  'Balance type: Gift balance': 'Loại số dư: số dư tặng',
  'Configure promotional gifts for exact recharge amounts':
    'Cấu hình quà tặng theo số tiền nạp chính xác',
  'Edit gift tier': 'Sửa mức tặng',
  Gift: 'Tặng',
  'Gift Amount (USD)': 'Số tiền tặng (USD)',
  'Gift Quota': 'Hạn mức tặng',
  'Gift balance': 'Số dư tặng',
  'Gift balance is promotional credit and produces no revenue.':
    'Số dư tặng là tín dụng khuyến mại và không tạo doanh thu.',
  'Gift balance produces no recognized revenue when spent.':
    'Chi tiêu số dư tặng không tạo doanh thu được ghi nhận.',
  'Gift consumption': 'Mức sử dụng số dư tặng',
  'Gift cost': 'Chi phí số dư tặng',
  'Gift quota map by exact recharge amount (JSON object)':
    'Hạn mức tặng theo số tiền nạp chính xác (đối tượng JSON)',
  'Gift {{bonus}} · Total {{total}}': 'Tặng {{bonus}} · Tổng {{total}}',
  'Legacy unattributed': 'Dữ liệu cũ chưa phân bổ',
  'Legacy unattributed consumption': 'Mức sử dụng cũ chưa phân bổ',
  'No gift tiers configured.': 'Chưa cấu hình mức tặng.',
  'Nominal consumption': 'Mức tiêu dùng danh nghĩa',
  'Optional promotional quota credited in addition to the paid quota.':
    'Hạn mức khuyến mại tùy chọn được cộng thêm ngoài hạn mức đã thanh toán.',
  'Paid balance': 'Số dư đã thanh toán',
  Principal: 'Tiền nạp gốc',
  'Recharge gift': 'Quà tặng nạp tiền',
  'Recognized revenue': 'Doanh thu được ghi nhận',
  'Redemption codes add gift balance. Gift balance is promotional credit and does not count as paid recharge.':
    'Mã đổi thưởng cộng số dư tặng khuyến mại và không được tính là khoản nạp đã thanh toán.',
  'Redemption codes always create promotional gift balance and never paid balance.':
    'Mã đổi thưởng luôn tạo số dư tặng khuyến mại, không tạo số dư đã thanh toán.',
  'Set the gift quota credited for an exact recharge amount.':
    'Đặt hạn mức tặng cho một số tiền nạp chính xác.',
  'The gift applies only when this exact amount is recharged.':
    'Quà tặng chỉ áp dụng khi nạp đúng số tiền này.',
  'Total credited': 'Tổng cộng vào',
  'Use paid balance only after confirming payment was received.':
    'Chỉ dùng số dư đã thanh toán sau khi xác nhận đã nhận tiền.',
  'e.g., 50000': 'ví dụ: 50000',
  gift: 'tặng',
})

Object.assign(newKeys.en, {
  'Amount gift must be a JSON object': 'Amount gift must be a JSON object',
  'Gift quota cannot be negative': 'Gift quota cannot be negative',
})
Object.assign(newKeys.zh, {
  'Amount gift must be a JSON object': '充值赠送必须是 JSON 对象',
  'Gift quota cannot be negative': '赠送额度不能为负数',
})
Object.assign(newKeys['zh-TW'], {
  'Amount gift must be a JSON object': '儲值贈送必須是 JSON 物件',
  'Gift quota cannot be negative': '贈送額度不能為負數',
})
Object.assign(newKeys.fr, {
  'Amount gift must be a JSON object':
    'Les bonus par montant doivent être un objet JSON',
  'Gift quota cannot be negative': 'Le quota offert ne peut pas être négatif',
})
Object.assign(newKeys.ja, {
  'Amount gift must be a JSON object':
    'チャージ特典は JSON オブジェクトで指定してください',
  'Gift quota cannot be negative': '特典クォータを負の値にはできません',
})
Object.assign(newKeys.ru, {
  'Amount gift must be a JSON object':
    'Бонусы по суммам должны быть объектом JSON',
  'Gift quota cannot be negative':
    'Подарочная квота не может быть отрицательной',
})
Object.assign(newKeys.vi, {
  'Amount gift must be a JSON object':
    'Quà tặng theo số tiền phải là một đối tượng JSON',
  'Gift quota cannot be negative': 'Hạn mức tặng không được âm',
})

Object.assign(newKeys.en, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    'The redemption card type determines whether quota is added to paid or gift balance.',
})
Object.assign(newKeys.zh, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    '兑换卡类型决定额度计入付费余额还是赠送余额。',
})
Object.assign(newKeys['zh-TW'], {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    '兌換卡類型決定額度計入付費餘額或贈送餘額。',
})
Object.assign(newKeys.fr, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    'Le type de carte détermine si le quota est ajouté au solde payé ou offert.',
})
Object.assign(newKeys.ja, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    '引換カードの種類により、クォータが支払済み残高または特典残高のどちらに追加されるかが決まります。',
})
Object.assign(newKeys.ru, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    'Тип карты определяет, будет ли квота зачислена на оплаченный или подарочный баланс.',
})
Object.assign(newKeys.vi, {
  'The redemption card type determines whether quota is added to paid or gift balance.':
    'Loại thẻ đổi thưởng quyết định hạn mức được cộng vào số dư đã thanh toán hay số dư tặng.',
})

Object.assign(newKeys.en, {
  'Attribute balance': 'Attribute balance',
  'Attribution reason': 'Attribution reason',
  'Balance attribution updated': 'Balance attribution updated',
  'Enter attribution reason': 'Enter attribution reason',
  'Paid balance after attribution': 'Paid balance after attribution',
  'Reclassification keeps total quota unchanged.':
    'Reclassification keeps total quota unchanged.',
})
Object.assign(newKeys.zh, {
  'Attribute balance': '归纳余额',
  'Attribution reason': '归因原因',
  'Balance attribution updated': '余额归因已更新',
  'Enter attribution reason': '请输入归因原因',
  'Paid balance after attribution': '归因后的付费余额',
  'Reclassification keeps total quota unchanged.': '重新归因不会改变总额度。',
})
Object.assign(newKeys['zh-TW'], {
  'Attribute balance': '歸納餘額',
  'Attribution reason': '歸因原因',
  'Balance attribution updated': '餘額歸因已更新',
  'Enter attribution reason': '請輸入歸因原因',
  'Paid balance after attribution': '歸因後的付費餘額',
  'Reclassification keeps total quota unchanged.': '重新歸因不會改變總額度。',
})
Object.assign(newKeys.fr, {
  'Attribute balance': 'Attribuer le solde',
  'Attribution reason': "Motif de l'attribution",
  'Balance attribution updated': "L'attribution du solde a été mise à jour",
  'Enter attribution reason': "Saisissez le motif de l'attribution",
  'Paid balance after attribution': 'Solde payé après attribution',
  'Reclassification keeps total quota unchanged.':
    'La réattribution ne modifie pas le quota total.',
})
Object.assign(newKeys.ja, {
  'Attribute balance': '残高を帰属',
  'Attribution reason': '帰属理由',
  'Balance attribution updated': '残高の帰属を更新しました',
  'Enter attribution reason': '帰属理由を入力',
  'Paid balance after attribution': '帰属後の支払済み残高',
  'Reclassification keeps total quota unchanged.':
    '再帰属しても合計クォータは変わりません。',
})
Object.assign(newKeys.ru, {
  'Attribute balance': 'Распределить баланс',
  'Attribution reason': 'Причина распределения',
  'Balance attribution updated': 'Распределение баланса обновлено',
  'Enter attribution reason': 'Укажите причину распределения',
  'Paid balance after attribution': 'Оплаченный баланс после распределения',
  'Reclassification keeps total quota unchanged.':
    'Перераспределение не изменяет общий лимит.',
})
Object.assign(newKeys.vi, {
  'Attribute balance': 'Phân loại số dư',
  'Attribution reason': 'Lý do phân loại',
  'Balance attribution updated': 'Đã cập nhật phân loại số dư',
  'Enter attribution reason': 'Nhập lý do phân loại',
  'Paid balance after attribution': 'Số dư đã thanh toán sau khi phân loại',
  'Reclassification keeps total quota unchanged.':
    'Việc phân loại lại không làm thay đổi tổng hạn mức.',
})

Object.assign(newKeys.en, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.',
  'Enter the paid portion of the legacy balance.':
    'Enter the paid portion of the legacy balance.',
  'Paid portion must be between 0 and {{amount}}.':
    'Paid portion must be between 0 and {{amount}}.',
  'Paid portion of legacy balance': 'Paid portion of legacy balance',
  'Reason must be between 3 and 200 characters.':
    'Reason must be between 3 and 200 characters.',
  'Select paid or gift balance.': 'Select paid or gift balance.',
})
Object.assign(newKeys.zh, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    '请选择历史未归因余额中已确认收款的部分，剩余额度将归为赠送余额。',
  'Enter the paid portion of the legacy balance.':
    '请输入历史未归因余额中的付费部分。',
  'Paid portion must be between 0 and {{amount}}.':
    '付费部分必须在 0 到 {{amount}} 之间。',
  'Paid portion of legacy balance': '历史未归因余额中的付费部分',
  'Reason must be between 3 and 200 characters.':
    '归因原因必须为 3 到 200 个字符。',
  'Select paid or gift balance.': '请选择付费余额或赠送余额。',
})
Object.assign(newKeys['zh-TW'], {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    '請選擇歷史未歸因餘額中已確認收款的部分，剩餘額度將歸為贈送餘額。',
  'Enter the paid portion of the legacy balance.':
    '請輸入歷史未歸因餘額中的付費部分。',
  'Paid portion must be between 0 and {{amount}}.':
    '付費部分必須在 0 到 {{amount}} 之間。',
  'Paid portion of legacy balance': '歷史未歸因餘額中的付費部分',
  'Reason must be between 3 and 200 characters.':
    '歸因原因必須為 3 到 200 個字元。',
  'Select paid or gift balance.': '請選擇付費餘額或贈送餘額。',
})
Object.assign(newKeys.fr, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    'Choisissez la part du solde historique non attribué provenant de paiements vérifiés. Le reste devient un solde offert.',
  'Enter the paid portion of the legacy balance.':
    'Saisissez la part payée du solde historique.',
  'Paid portion must be between 0 and {{amount}}.':
    'La part payée doit être comprise entre 0 et {{amount}}.',
  'Paid portion of legacy balance': 'Part payée du solde historique',
  'Reason must be between 3 and 200 characters.':
    'Le motif doit contenir entre 3 et 200 caractères.',
  'Select paid or gift balance.': 'Sélectionnez le solde payé ou offert.',
})
Object.assign(newKeys.ja, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    '過去の未帰属残高のうち、入金確認済みの金額を選択してください。残りは特典残高になります。',
  'Enter the paid portion of the legacy balance.':
    '過去の残高のうち支払済みの金額を入力してください。',
  'Paid portion must be between 0 and {{amount}}.':
    '支払済み部分は 0 から {{amount}} の範囲で入力してください。',
  'Paid portion of legacy balance': '過去の残高の支払済み部分',
  'Reason must be between 3 and 200 characters.':
    '帰属理由は3文字以上200文字以内で入力してください。',
  'Select paid or gift balance.':
    '支払済み残高または特典残高を選択してください。',
})
Object.assign(newKeys.ru, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    'Укажите часть прежнего нераспределенного баланса, подтвержденную оплатой. Остаток станет подарочным балансом.',
  'Enter the paid portion of the legacy balance.':
    'Введите оплаченную часть прежнего баланса.',
  'Paid portion must be between 0 and {{amount}}.':
    'Оплаченная часть должна быть от 0 до {{amount}}.',
  'Paid portion of legacy balance': 'Оплаченная часть прежнего баланса',
  'Reason must be between 3 and 200 characters.':
    'Причина должна содержать от 3 до 200 символов.',
  'Select paid or gift balance.': 'Выберите оплаченный или подарочный баланс.',
})
Object.assign(newKeys.vi, {
  'Choose how much of the legacy unattributed balance came from verified payments. The remainder becomes gift balance.':
    'Chọn phần số dư cũ chưa phân loại đã được xác nhận thanh toán. Phần còn lại sẽ thành số dư tặng.',
  'Enter the paid portion of the legacy balance.':
    'Nhập phần đã thanh toán của số dư cũ.',
  'Paid portion must be between 0 and {{amount}}.':
    'Phần đã thanh toán phải nằm trong khoảng từ 0 đến {{amount}}.',
  'Paid portion of legacy balance': 'Phần đã thanh toán của số dư cũ',
  'Reason must be between 3 and 200 characters.':
    'Lý do phải có từ 3 đến 200 ký tự.',
  'Select paid or gift balance.': 'Chọn số dư đã thanh toán hoặc số dư tặng.',
})

Object.assign(newKeys.en, {
  'All redemption codes add paid balance.':
    'All redemption codes add paid balance.',
  'Enter the final paid balance.': 'Enter the final paid balance.',
  'Final paid balance': 'Final paid balance',
  'Final paid balance must be between 0 and {{amount}}.':
    'Final paid balance must be between 0 and {{amount}}.',
  'Redemption codes add paid balance.': 'Redemption codes add paid balance.',
  'Set paid balance': 'Set paid balance',
  'The balance allocation is unchanged.':
    'The balance allocation is unchanged.',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    'The remaining balance becomes gift balance and legacy attribution is cleared.',
})
Object.assign(newKeys.zh, {
  'All redemption codes add paid balance.': '所有兑换码均计入付费余额。',
  'Enter the final paid balance.': '请输入最终付费余额。',
  'Final paid balance': '最终付费余额',
  'Final paid balance must be between 0 and {{amount}}.':
    '最终付费余额必须在 0 到 {{amount}} 之间。',
  'Redemption codes add paid balance.': '兑换码将计入付费余额。',
  'Set paid balance': '设置付费余额',
  'The balance allocation is unchanged.': '余额分配未发生变化。',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    '剩余额度将转为赠送余额，历史未归因余额将清零。',
})
Object.assign(newKeys['zh-TW'], {
  'All redemption codes add paid balance.': '所有兌換碼均計入付費餘額。',
  'Enter the final paid balance.': '請輸入最終付費餘額。',
  'Final paid balance': '最終付費餘額',
  'Final paid balance must be between 0 and {{amount}}.':
    '最終付費餘額必須在 0 到 {{amount}} 之間。',
  'Redemption codes add paid balance.': '兌換碼將計入付費餘額。',
  'Set paid balance': '設定付費餘額',
  'The balance allocation is unchanged.': '餘額分配未發生變化。',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    '剩餘額度將轉為贈送餘額，歷史未歸因餘額將清零。',
})
Object.assign(newKeys.fr, {
  'All redemption codes add paid balance.':
    'Tous les codes alimentent le solde payé.',
  'Enter the final paid balance.': 'Saisissez le solde payé final.',
  'Final paid balance': 'Solde payé final',
  'Final paid balance must be between 0 and {{amount}}.':
    'Le solde payé final doit être compris entre 0 et {{amount}}.',
  'Redemption codes add paid balance.': 'Les codes alimentent le solde payé.',
  'Set paid balance': 'Définir le solde payé',
  'The balance allocation is unchanged.':
    "La répartition du solde n'a pas changé.",
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    "Le solde restant devient un solde offert et l'ancienne attribution est effacée.",
})
Object.assign(newKeys.ja, {
  'All redemption codes add paid balance.':
    'すべての引換コードは支払済み残高に追加されます。',
  'Enter the final paid balance.': '最終的な支払済み残高を入力してください。',
  'Final paid balance': '最終支払済み残高',
  'Final paid balance must be between 0 and {{amount}}.':
    '最終支払済み残高は 0 から {{amount}} の範囲で入力してください。',
  'Redemption codes add paid balance.':
    '引換コードは支払済み残高に追加されます。',
  'Set paid balance': '支払済み残高を設定',
  'The balance allocation is unchanged.': '残高の配分は変更されていません。',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    '残りは特典残高となり、過去の未帰属残高は消去されます。',
})
Object.assign(newKeys.ru, {
  'All redemption codes add paid balance.':
    'Все коды пополняют оплаченный баланс.',
  'Enter the final paid balance.': 'Введите итоговый оплаченный баланс.',
  'Final paid balance': 'Итоговый оплаченный баланс',
  'Final paid balance must be between 0 and {{amount}}.':
    'Итоговый оплаченный баланс должен быть от 0 до {{amount}}.',
  'Redemption codes add paid balance.': 'Коды пополняют оплаченный баланс.',
  'Set paid balance': 'Задать оплаченный баланс',
  'The balance allocation is unchanged.':
    'Распределение баланса не изменилось.',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    'Остаток станет подарочным балансом, прежнее распределение будет очищено.',
})
Object.assign(newKeys.vi, {
  'All redemption codes add paid balance.':
    'Mọi mã đổi thưởng đều cộng vào số dư đã thanh toán.',
  'Enter the final paid balance.': 'Nhập số dư đã thanh toán cuối cùng.',
  'Final paid balance': 'Số dư đã thanh toán cuối cùng',
  'Final paid balance must be between 0 and {{amount}}.':
    'Số dư đã thanh toán cuối cùng phải từ 0 đến {{amount}}.',
  'Redemption codes add paid balance.':
    'Mã đổi thưởng cộng vào số dư đã thanh toán.',
  'Set paid balance': 'Đặt số dư đã thanh toán',
  'The balance allocation is unchanged.': 'Phân bổ số dư không thay đổi.',
  'The remaining balance becomes gift balance and legacy attribution is cleared.':
    'Phần còn lại thành số dư tặng và phân loại cũ sẽ được xóa.',
})

Object.assign(newKeys.en, {
  'Allocate balance': 'Allocate balance',
  'Set all to gift balance': 'Set all to gift balance',
  'Set all to paid balance': 'Set all to paid balance',
})
Object.assign(newKeys.zh, {
  'Allocate balance': '分配余额',
  'Set all to gift balance': '全部设为赠送余额',
  'Set all to paid balance': '全部设为付费余额',
})
Object.assign(newKeys['zh-TW'], {
  'Allocate balance': '分配餘額',
  'Set all to gift balance': '全部設為贈送餘額',
  'Set all to paid balance': '全部設為付費餘額',
})
Object.assign(newKeys.fr, {
  'Allocate balance': 'Répartir le solde',
  'Set all to gift balance': 'Tout définir comme solde offert',
  'Set all to paid balance': 'Tout définir comme solde payé',
})
Object.assign(newKeys.ja, {
  'Allocate balance': '残高を配分',
  'Set all to gift balance': 'すべて特典残高に設定',
  'Set all to paid balance': 'すべて支払済み残高に設定',
})
Object.assign(newKeys.ru, {
  'Allocate balance': 'Распределить баланс',
  'Set all to gift balance': 'Всё в подарочный баланс',
  'Set all to paid balance': 'Всё в оплаченный баланс',
})
Object.assign(newKeys.vi, {
  'Allocate balance': 'Phân bổ số dư',
  'Set all to gift balance': 'Đặt tất cả thành số dư tặng',
  'Set all to paid balance': 'Đặt tất cả thành số dư đã thanh toán',
})

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
