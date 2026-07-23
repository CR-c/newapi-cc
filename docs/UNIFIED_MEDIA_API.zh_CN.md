# 统一图片与视频 API 接入

New API 提供 OpenAI 兼容的图片与视频接口。选择模型、提交统一请求即可；鉴权、计费、任务状态与结果下载由本站处理。

## 通用约定

| 项 | 说明 |
| -- | ---- |
| Base URL | 你的站点地址，例如 `https://your-new-api.example` |
| 鉴权 | `Authorization: Bearer <New API token>` |
| 模型列表 | `GET /v1/models` |
| 分组 | **只绑定在 token 上**；请求体不要传 `group` |
| 价格与能力 | 以控制台定价页、同一 token 的 `/v1/models` 为准 |

当前公开分组：`64生图`、`生图`、`dddd-sd-图`、`grok按次`、`video-dddd`、`sd-token`、`sd-video`（可能调整）。

### 分组一览

| 分组 | 类型 | 端点 | 计费 | 参考素材传法 |
| ---- | ---- | ---- | ---- | ------------ |
| `dddd-sd-图` | 图片 | `POST /v1/images/generations` | 按次 | HTTPS / Base64（`image`） |
| `64生图` | 图片 | `POST /v1/images/generations` | 按次 | HTTPS / Base64（`image`） |
| `video-dddd` | 视频 | `POST /v1/videos` 等 | 按次 | 仅 `asset://`（先素材库） |
| `sd-token` | 视频 | `POST /v1/videos` 等 | Token | 公网 HTTPS URL |
| `sd-video` | 视频 | `POST /v1/videos` 等 | 按次 | 公网 HTTPS URL |

---

## 图片生成

### 统一接口

```text
POST /v1/images/generations
```

**请求示例**

```json
{
  "model": "gemini-3-pro-image-preview",
  "prompt": "a cat on the moon",
  "size": "1024x1024",
  "n": 1,
  "response_format": "url",
  "image": ["https://example.com/reference.png"]
}
```

**统一字段**

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `model` | string | 必填，`/v1/models` 返回的模型名 |
| `prompt` | string | 必填，提示词 |
| `size` | string | 尺寸、`auto` 或比例映射尺寸，见模型表 |
| `n` | integer | 生成数量 |
| `quality` | string | 可选：`low` / `medium` / `high`（模型支持时） |
| `background` | string | 可选：`opaque` / `transparent`（模型支持时） |
| `image` | string 或 string[] | 参考图：HTTP(S)、Base64 或 data URL |
| `input_reference` | string 或 string[] | `image` 的兼容别名 |
| `response_format` | string | `url` 或 `b64_json` |

**响应**

```json
{
  "created": 1783866456,
  "data": [{ "url": "https://example.com/result.png" }]
}
```

`response_format: "b64_json"` 时，`data` 项返回 `b64_json`。

### 图片分组：参数能力对照

| 参数 | `dddd-sd-图` | `64生图` · Gemini | `64生图` · gpt-image-2 |
| ---- | ------------ | ----------------- | ---------------------- |
| `model` / `prompt` | ✓ | ✓ | ✓ |
| `size` | ✓ 预设尺寸 | ✓ 映射比例（固定 2K 输出） | ✓ 1K/2K/4K/精确/`auto` |
| `n` | ✓（建议先 `1`） | ✓ | ✓ |
| `image` 参考图 | ✓ 最多约 10 张 | ✓ 最多 14 张 | ✓ 最多 14 张 |
| `quality` | — | — | ✓ `low`/`medium`/`high` |
| `background` | — | — | ✓ `opaque`/`transparent` |
| `response_format` | `url` | `url` / `b64_json` | `url` / `b64_json` |

> 表中「—」表示该分组/模型不使用该参数；填了也可能无效。

---

### `dddd-sd-图`

| 模型 | 当前价格 | 说明 |
| ---- | -------- | ---- |
| `dola-seedream-5-0-pro-260628` | ¥0.538650/次 | 已验证 `1024x1024`、`n=1` |
| `seedream-4-5-251128` | ¥0.239400/次 | 文本生图 |
| `seedream-5-0-lite-260128` | ¥0.209475/次 | 已验证 `1920x1920`、`n=1` |

**支持的参数**

| 参数 | 取值 / 说明 |
| ---- | ----------- |
| `model` | 上表模型名 |
| `prompt` | 必填 |
| `size` | 常用：`1024x1024`、`1920x1920`、`2048x2048`、`2560x1440`、`1440x2560`、`1024x1792`、`1792x1024` |
| `n` | 生成数量，接入先用 `1` |
| `image` | 可选参考图（URL / Base64） |
| `response_format` | 建议 `url` |

**示例**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dola-seedream-5-0-pro-260628",
    "prompt": "极简产品摄影，蓝色玻璃瓶放在白色台面上，柔和棚拍光",
    "size": "1024x1024",
    "n": 1,
    "response_format": "url"
  }'
```

**提示**

- 同步长请求，读取超时建议 ≥ 300 秒；处理中不要重复提交。
- 收到 `data[].url` 后尽快下载并自行持久化。
- token 绑定 `dddd-sd-图`，请求体不要传 `group`。

---

### `64生图`

| 模型 | 价格 | 分辨率 | 比例 | 其他参数 |
| ---- | ---- | ------ | ---- | -------- |
| `gemini-3-pro-image-preview` | `$0.15/次` | 固定 2K | 标准 10 比例 | `n`、参考图、`response_format` |
| `gemini-3.1-flash-image-preview` | `$0.10/次` | 固定 2K | 14 比例（含超长图） | 同上 |
| `gpt-image-2` | `$0.10/次` | 1K / 2K / 4K / 自动 / 精确 | 标准 10 比例 | `quality`、`background`、`n`、参考图、`response_format` |

**标准比例（10）**

```text
1:1  16:9  9:16  4:3  3:4  21:9  3:2  2:3  5:4  4:5
```

**`gemini-3.1-flash-image-preview` 额外比例**

```text
1:4  1:8  4:1  8:1
```

**Gemini 说明**

- 统一图片端点下固定输出 **2K**；`size` 只用于选最接近的宽高比，不能切换 1K/2K/4K。
- 参考图最多 **14** 张。

**`gpt-image-2` 常用预设**

| 比例 | 1K | 2K | 4K |
| ---- | -- | -- | -- |
| 1:1 | `1024x1024` | `2048x2048` | `2480x2480` |
| 16:9 | `1280x720` | `2560x1440` | `3328x1872` |
| 9:16 | `720x1280` | `1440x2560` | `1872x3328` |

也支持精确尺寸（宽高均为 16 的倍数，如 `2160x3840`），以及 `size: "auto"`。

**示例**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "a red apple on a white table",
    "size": "2048x2048",
    "quality": "high",
    "background": "opaque",
    "response_format": "url"
  }'
```

---

## 视频生成

### 统一接口（所有视频分组相同）

| 步骤 | 接口 | 说明 |
| ---- | ---- | ---- |
| 创建 | `POST /v1/videos` | 提交任务，返回任务 ID |
| 查询 | `GET /v1/videos/{task_id}` | `queued` / `in_progress` / `completed` / `failed` |
| 下载 | `GET /v1/videos/{task_id}/content` | MP4 |

**请求示例**

```json
{
  "model": "dreamina-seedance-2-0-mini-hc",
  "prompt": "a drone shot over mountains",
  "seconds": 8,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "images": ["asset://asset-image-1"]
}
```

**统一字段（按模型能力选用）**

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `model` | string | 必填 |
| `prompt` | string | 必填 |
| `seconds` | integer / string | 推荐时长字段 |
| `duration` | integer / string | `seconds` 别名；勿与 `seconds` 传不同值 |
| `size` | string | 如 `1920x1080`（部分模型用 `resolution`） |
| `aspect_ratio` | string | 如 `16:9`、`9:16` |
| `resolution` | string | 如 `480p`、`720p`、`1080p` |
| `image` | string | 单图兼容字段 → 等价于单元素 `images` |
| `images` | string[] | 参考图 |
| `videos` | string[] | 参考视频 |
| `audios` | string[] | 参考音频（通常不可单独使用） |
| `generate_audio` | boolean | 是否生成有声视频 |
| `watermark` | boolean | 是否水印 |
| `first_image` | string | 首帧，须与 `last_image` 成对 |
| `last_image` | string | 尾帧；不可与 `images`/`videos`/`audios` 混用 |
| `auto_face` | boolean | 自动人脸处理（仅部分模型） |

**任务创建响应**

```json
{
  "id": "video_xxx",
  "status": "queued",
  "model": "dreamina-seedance-2-0-mini-hc",
  "created_at": 1783866456
}
```

**查询响应示例**

```json
{
  "id": "video_xxx",
  "status": "in_progress",
  "model": "dreamina-seedance-2-0-mini-hc",
  "progress": 35
}
```

**通用提示**

- 轮询间隔建议 5–10 秒；客户端超时 ≥ 600 秒。
- 任务未结束前不要重复提交。
- 换分组换 token；请求体不要传 `group`。

### 视频分组：参数能力对照

| 参数 | `video-dddd` | `sd-token` | `sd-video` |
| ---- | ------------ | ---------- | ---------- |
| `model` / `prompt` | ✓ | ✓ | ✓ |
| `seconds` / `duration` | ✓ 4–15 | ✓（常用 4–15） | ✓ 见模型表 |
| `aspect_ratio` | ✓ `1:1` `16:9` `9:16` | ✓ `16:9` `9:16` `1:1` | ✓ 见模型表 |
| `resolution` | ✓ 见模型表 | ✓ `480p`/`720p`/`1080p` | ✓ 仅 `720p` |
| `images` | ✓ 最多 9 · **仅 `asset://`** | ✓ 最多 9 · HTTPS | ✓ 见模型表 · HTTPS |
| `videos` | ✓ 最多 3 · **仅 `asset://`** | ✓ 最多 3 · HTTPS | ✓ 部分模型 · HTTPS |
| `audios` | ✓ 最多 3 · **仅 `asset://`** | ✓ 最多 3 · HTTPS | ✓ 部分模型 · HTTPS |
| `generate_audio` | ✓ | ✓ | — |
| `watermark` | ✓ | ✓ | — |
| `first_image` + `last_image` | — | — | ✓（与多模态参考互斥） |
| `auto_face` | — | — | ✓（仅 `sd2.0-933`） |

> `audios` 一般需至少再带 1 张图或 1 段视频；具体以各分组规则为准。

---

### `video-dddd`

多模态参考视频；创建 / 查询 / 下载用统一视频接口。

#### 模型

| 模型 | 时长 | 清晰度 | 参考素材 |
| ---- | ---- | ------ | -------- |
| `dreamina-seedance-2-0-mini-hc` | 4–15 秒 | `480p`、`720p` | 最多 9 图 + 3 视频 + 3 音频 |
| `dreamina-seedance-2-0-fast-hc` | 4–15 秒 | `480p`、`720p` | 最多 9 图 + 3 视频 + 3 音频 |
| `dreamina-seedance-2-0-hc` | 4–15 秒 | `480p`、`720p`、`1080p`、`4k` | 最多 9 图 + 3 视频 + 3 音频 |

#### 支持的参数

| 参数 | 是否支持 | 取值 / 说明 |
| ---- | -------- | ----------- |
| `model` | ✓ | 上表模型名 |
| `prompt` | ✓ | 必填 |
| `seconds` | ✓ | 整数 4–15；也可用 `duration` |
| `aspect_ratio` | ✓ | `1:1`、`16:9`、`9:16` |
| `resolution` | ✓ | mini/fast：`480p`/`720p`；hc 另含 `1080p`/`4k` |
| `images` | ✓ | 0–9；**仅** `asset://<素材ID>` |
| `videos` | ✓ | 0–3；**仅** `asset://` |
| `audios` | ✓ | 0–3；**仅** `asset://`；不可单独只传音频 |
| `generate_audio` | ✓ | `true` / `false` |
| `watermark` | ✓ | `true` / `false` |
| `first_image` / `last_image` | — | 不支持 |
| `auto_face` | — | 不支持 |

#### 参考素材上限

| 字段 | 上限 | 超限 | 补充 |
| ---- | ---- | ---- | ---- |
| `images` | ≤ 9 | `400 invalid_images` | — |
| `videos` | ≤ 3 | `400 invalid_videos` | 建议单段 2–15s，总时长 ≤ 15s |
| `audios` | ≤ 3 | `400 invalid_audios` | 建议单段 2–15s，总时长 ≤ 15s |

#### 素材库（本分组必用）

| 接口 | 说明 |
| ---- | ---- |
| `POST /pg/assets` | 上传本地文件，得到临时 URL |
| `POST /v1/sd/assets` | 注册素材，得到 `data.Id` |
| `GET /v1/sd/assets/{asset_id}` | 查询至 `Status: Active` |

| 素材类型 | 上传 `kind` | 注册 `AssetType` | 放入字段 |
| -------- | ----------- | ---------------- | -------- |
| 图片 | `image` | `Image` | `images` |
| 视频 | `video` | `Video` | `videos` |
| 音频 | `audio` | `Audio` | `audios` |

**推荐流程**

1. 上传本地文件 → 得到 URL  
2. `POST /v1/sd/assets` 注册 → 保存 `data.Id`  
3. 轮询至 `Active`  
4. `POST /v1/videos`，字段中写 `asset://<Id>`  

多个素材 = 多次 1–3，最后一次创建里放多个 `asset://`。

```bash
# 1) 上传
curl "$NEWAPI_BASE_URL/pg/assets" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -F "kind=image" \
  -F "file=@./reference-1.png"

# 2) 注册
curl "$NEWAPI_BASE_URL/v1/sd/assets" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "URL": "https://your-new-api.example/pg/assets/<upload_id>/reference-1.png",
    "Name": "reference_1",
    "AssetType": "Image"
  }'

# 3) 查询
curl "$NEWAPI_BASE_URL/v1/sd/assets/asset-N3w8cK7pQ2mR5tV9xY4zA6bD" \
  -H "Authorization: Bearer $NEWAPI_API_KEY"
```

**接入提示**

- 本分组 **禁止** 在 `images`/`videos`/`audios` 里直接塞 HTTPS URL。
- 类型必须匹配（图不能塞进 `videos` 等）。
- 音频不能单独使用：至少再带 1 图或 1 视频。
- 先确认素材 `Active`，再创建视频，便于区分「素材失败」与「视频任务失败」。

#### 示例

**A · 文生视频（无参考）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-mini-hc",
    "prompt": "纸飞机在清晨天空中缓慢飞行",
    "seconds": 4,
    "aspect_ratio": "16:9",
    "resolution": "480p",
    "generate_audio": false,
    "watermark": false
  }'
```

**B · 单图参考**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-mini-hc",
    "prompt": "让人物自然转身并看向镜头",
    "seconds": 4,
    "aspect_ratio": "9:16",
    "resolution": "480p",
    "images": ["asset://asset-image-1"],
    "generate_audio": false,
    "watermark": false
  }'
```

**C · 多图（最多 9）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-hc",
    "prompt": "参考多张人物设定图，生成同一角色在城市夜景中缓慢走动",
    "seconds": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "images": [
      "asset://asset-image-1",
      "asset://asset-image-2",
      "asset://asset-image-3",
      "asset://asset-image-4",
      "asset://asset-image-5",
      "asset://asset-image-6",
      "asset://asset-image-7",
      "asset://asset-image-8",
      "asset://asset-image-9"
    ],
    "generate_audio": false,
    "watermark": false
  }'
```

**D · 图 + 视频 + 音频**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-hc",
    "prompt": "参考人物图、动作视频和背景音乐，生成带节奏感的短视频",
    "seconds": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "images": ["asset://asset-image-1"],
    "videos": ["asset://asset-video-1"],
    "audios": ["asset://asset-audio-1"],
    "generate_audio": false,
    "watermark": false
  }'
```

---

### `sd-token`

Token 计费视频；接口与上文统一三件套相同。

#### 模型与价格

模型：`doubao-seedance-2-0-260128`

| 输出档位 | 基准价（含视频参考） | 基准价（不含视频参考） | 本分组价（×0.78，含视频参考） | 本分组价（×0.78，不含视频参考） |
| -------- | -------------------- | ---------------------- | ----------------------------- | ------------------------------- |
| `480p`/`720p` | ¥28.00/百万 Token | ¥46.00/百万 Token | ¥21.84/百万 Token | ¥35.88/百万 Token |
| `1080p` | ¥31.00/百万 Token | ¥51.00/百万 Token | ¥24.18/百万 Token | ¥39.78/百万 Token |

- 「含视频参考」= 请求带了 `videos`；仅图/音频不算该档。
- 4K 当前未启用。

#### 支持的参数

| 参数 | 是否支持 | 取值 / 说明 |
| ---- | -------- | ----------- |
| `model` | ✓ | `doubao-seedance-2-0-260128` |
| `prompt` | ✓ | 必填 |
| `seconds` | ✓ | 常用 4–15 |
| `aspect_ratio` | ✓ | `16:9`、`9:16`、`1:1` |
| `resolution` | ✓ | `480p`、`720p`、`1080p` |
| `images` | ✓ | 最多 9；**公网 HTTPS URL** |
| `videos` | ✓ | 最多 3；公网 HTTPS URL |
| `audios` | ✓ | 最多 3；公网 HTTPS URL；不宜单独只传音频 |
| `generate_audio` | ✓ | `true` / `false` |
| `watermark` | ✓ | `true` / `false` |
| `first_image` / `last_image` | — | 不支持 |
| `auto_face` | — | 不支持 |

**示例**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "蓝色渐变背景缓慢流动，画面干净，无文字",
    "seconds": 4,
    "aspect_ratio": "16:9",
    "resolution": "480p",
    "generate_audio": false,
    "watermark": false
  }'
```

**提示**

- 提交预扣，完成后按 `usage.total_tokens` 多退少补；失败退回预扣。
- 接入建议：`4 秒 + 480p + 无参考`；超时 ≥ 600 秒；勿高频轮询。

---

### `sd-video`

按次计费；接口与上文统一三件套相同；参考素材用公网 HTTPS URL。

#### 模型

| 模型 | 价格 | 时长 | 比例 | 图 | 视频 | 音频 |
| ---- | ---- | ---- | ---- | -- | ---- | ---- |
| `sd2.0-933` | 7 元/次 | 4–15 秒 | `21:9` `16:9` `4:3` `1:1` `3:4` `9:16` | 0–9 | 0–3 | 0–3 |
| `sd2.0-431` | 6 元/次 | 4–15 秒 | `16:9` `9:16` `1:1` | 0–4 | 0–3 | 0–1 |
| `sd2.0fast-431` | 5 元/次 | 10 或 15 秒 | `16:9` `9:16` `1:1` | 0–4 | 0–3 | 0–1 |
| `sd2.0-903` | 6.5 元/次 | 10 或 15 秒 | `16:9` `9:16` `1:1` | 0–9 | — | 0–3 |
| `sd2.0fast-903` | 5.5 元/次 | 10 或 15 秒 | `16:9` `9:16` `1:1` | 0–9 | — | 0–3 |

全部模型当前仅 **`720p`**。

#### 支持的参数

| 参数 | 是否支持 | 取值 / 说明 |
| ---- | -------- | ----------- |
| `model` | ✓ | 上表模型名 |
| `prompt` | ✓ | 必填；除 `sd2.0-933` 外最多 5000 字 |
| `seconds` | ✓ | 见上表；也可用 `duration` |
| `aspect_ratio` | ✓ | 见上表 |
| `resolution` | ✓ | 固定 `720p` |
| `images` | ✓ | 上限见上表；HTTPS URL |
| `videos` | 部分 | `933`/`431` 系列支持；`903` 系列不支持 |
| `audios` | ✓ | 上限见上表；`903` 系列带音频时须至少 1 张图 |
| `first_image` + `last_image` | ✓ | 成对使用；不可与 `images`/`videos`/`audios` 同时使用 |
| `auto_face` | 仅 `sd2.0-933` | `true` / `false` |
| `generate_audio` | — | 不支持 |
| `watermark` | — | 不支持 |

#### 示例

**多模态参考**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2.0-933",
    "prompt": "参考人物、动作和音乐生成自然流畅的短片",
    "seconds": 10,
    "aspect_ratio": "9:16",
    "resolution": "720p",
    "images": ["https://example.com/person.jpg"],
    "videos": ["https://example.com/motion.mp4"],
    "audios": ["https://example.com/music.mp3"]
  }'
```

**首尾帧（与多模态参考二选一）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2.0-933",
    "prompt": "从第一帧自然过渡到最后一帧",
    "seconds": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "first_image": "https://example.com/first.png",
    "last_image": "https://example.com/last.png"
  }'
```

---

## 使用说明

| 项 | 说明 |
| -- | ---- |
| 端点 | 图片与视频路径全站统一；分组差异在模型、参数能力、价格、素材传法 |
| 换分组 | 换对应 token，不要在 body 里传 `group` |
| 演练场 | 图片 `/pg/images/generations`，视频 `/pg/videos` |
| 能力与价格 | 以 token 的 `/v1/models` 与定价页为准 |
| 多余字段 | 可能被忽略或返回 400 |
| 错误格式 | 可能是 `error.message`，也可能是顶层 `code` + `message`，客户端宜两种都兼容 |
| 素材传法 | `asset://` 与 HTTPS 只影响字段取值，不改变端点 |
