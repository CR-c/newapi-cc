# 统一图片与视频 API 接入

New API 提供 OpenAI 兼容的图片与视频接口。选择模型、提交统一请求即可；鉴权、计费、任务状态与结果下载由本站处理。

> **在线文档**：`/docs.html`（统一媒体详表）· `/docs-official.html`（官方 New API 对接指南，顶部切页）

## 通用约定

| 项 | 说明 |
| -- | ---- |
| Base URL | 你的站点地址，例如 `https://your-new-api.example` |
| 鉴权 | `Authorization: Bearer <New API token>` |
| 模型列表 | `GET /v1/models` |
| 分组 | **只绑定在 token 上**；请求体不要传 `group` |
| 价格与能力 | 以控制台定价页、同一 token 的 `/v1/models` 为准 |

当前公开分组：`64生图`、`生图`、`dddd-sd-图`、`grok按次`、`grok按次-x`、`video-dddd`、`sd-token`、`sd-video`（可能调整）。

### 分组一览

| 分组 | 类型 | 端点 | 计费 | 参考素材传法 |
| ---- | ---- | ---- | ---- | ------------ |
| `dddd-sd-图` | 图片 | `POST /v1/images/generations` | 按次 | HTTPS / Base64（`image`） |
| `64生图` | 图片 | `POST /v1/images/generations` | 按次 | HTTPS / Base64（`image`） |
| `video-dddd` | 视频 | `POST /v1/videos` 等 | 按次 | 仅 `asset://`（先素材库） |
| `sd-token` | 视频 | `POST /v1/videos` 等 | Token | 公网 HTTPS URL（最多 9 图 + 3 视频 + 3 音频） |
| `sd-video` | 视频 | `POST /v1/videos` 等 | 按次 | 公网 HTTPS URL |
| `grok按次` | 视频 | `POST /v1/videos` 等 | 按次 | 公网 HTTPS URL |
| `grok按次-x` | 视频 | `POST /v1/videos` 等 | 按次 | 公网 HTTPS URL |

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
| `size` | ✓ 分辨率档位或 `宽x高` | ✓ 映射比例（固定 2K 输出） | ✓ 1K/2K/4K/精确/`auto` |
| `n` | ✓（见模型表） | ✓ | ✓ |
| `image` 参考图 | ✓ 最多 10 或 14 张（见模型） | ✓ 最多 14 张 | ✓ 最多 14 张 |
| `watermark` | ✓ | — | — |
| `quality` | — | — | ✓ `low`/`medium`/`high` |
| `background` | — | — | ✓ `opaque`/`transparent` |
| `response_format` | `url` / `b64_json` | `url` / `b64_json` | `url` / `b64_json` |

> 表中「—」表示该分组/模型不使用该参数；填了也可能无效。

---

### `dddd-sd-图`

Seedream 系列图片。统一端点 `POST /v1/images/generations`。token 绑定本分组，请求体不要传 `group`。

#### 模型

| 模型 | 当前价格 | 能力概要 |
| ---- | -------- | -------- |
| `dola-seedream-5-0-pro-260628` | ¥0.538650/次 | 文生图 / 单图与多参考图；`size`：`1K`/`2K` 或像素；参考图 ≤ 10 |
| `seedream-4-5-251128` | ¥0.239400/次 | 文生图 / 参考图 / 组图；`size`：`2K`/`4K` 或像素；参考图 ≤ 14 |
| `seedream-5-0-lite-260128` | ¥0.209475/次 | 文生图 / 参考图 / 组图；`size`：`2K`/`3K` 或像素；参考图 ≤ 14 |

价格以控制台定价页与调用日志为准。

#### 支持的参数（与 Seedream 能力一致）

| 参数 | 类型 | 说明 |
| ---- | ---- | ---- |
| `model` | string | 必填，上表模型名 |
| `prompt` | string | 必填，提示词 |
| `size` | string | **二选一**：分辨率档位，或 `宽x高` 像素；档位与像素不可混用，见下表 |
| `n` | integer | 生成张数；接入建议先用 `1`。pro 以单图为主；lite / 4.5 可按组图能力提高 |
| `image` | string / string[] | 可选参考图：公网 HTTPS URL、Base64 或 data URL |
| `input_reference` | string / string[] | `image` 的兼容别名 |
| `watermark` | boolean | 是否水印；默认随模型，显式传 `true`/`false` 可覆盖 |
| `response_format` | string | `url`（推荐）或 `b64_json` |
| `quality` / `background` | — | 本分组不使用 |

#### 按模型：`size` 与参考图

| 模型 | 分辨率档位 | 像素 `宽x高`（示例） | 参考图 `image` |
| ---- | ---------- | -------------------- | -------------- |
| `dola-seedream-5-0-pro-260628` | `1K`、`2K`（默认倾向 `2K`） | 如 `1024x1024`、`2048x2048`、`2560x1440` | 0–10 张 |
| `seedream-4-5-251128` | `2K`、`4K` | 如 `2048x2048`、`2560x1440`、`4096x4096` 量级 | 0–14 张 |
| `seedream-5-0-lite-260128` | `2K`、`3K` | 如 `1920x1920`、`2048x2048`、`2560x1440` | 0–14 张 |

常用像素示例：`1024x1024`、`1920x1920`、`2048x2048`、`2560x1440`、`1440x2560`、`1024x1792`、`1792x1024`。宽高比建议在约 `1:16`～`16:1` 内；单张参考图建议 ≤ 10MB，格式 jpeg / png / webp 等。

#### 示例

**文生图（档位）**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dola-seedream-5-0-pro-260628",
    "prompt": "极简产品摄影，蓝色玻璃瓶放在白色台面上，柔和棚拍光",
    "size": "2K",
    "n": 1,
    "watermark": false,
    "response_format": "url"
  }'
```

**文生图（像素）**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "seedream-5-0-lite-260128",
    "prompt": "清晨山间薄雾，写实风景",
    "size": "1920x1920",
    "n": 1,
    "response_format": "url"
  }'
```

**多参考图**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dola-seedream-5-0-pro-260628",
    "prompt": "保持人物外形一致，换成咖啡馆窗边光线",
    "size": "2K",
    "n": 1,
    "image": [
      "https://example.com/ref-1.png",
      "https://example.com/ref-2.png"
    ],
    "watermark": false,
    "response_format": "url"
  }'
```

#### 提示

- 同步长请求，读取超时建议 ≥ 300 秒；处理中不要重复提交。
- 响应 `data[].url` 通常有时效，请尽快下载并自行持久化。
- 档位（如 `2K`）与像素（如 `2048x2048`）二选一，不要混用。
- 参考图张数不要超过上表上限；pro 最多 10 张，lite / 4.5 最多 14 张。
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
| `first_image` | string | 首帧；`sd-video` 须与 `last_image` 成对且不可与参考素材混用；`sd-token` 可单独/成对，计入 9 图上限 |
| `last_image` | string | 尾帧；规则同上 |
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
| `seconds` / `duration` | ✓ 4–15 | ✓ 4–15，或 `-1` 智能时长 | ✓ 见模型表 |
| `aspect_ratio` | ✓ `1:1` `16:9` `9:16` | ✓ `adaptive` / `16:9` / `9:16` / `1:1` / `4:3` / `3:4` / `21:9` | ✓ 见模型表 |
| `resolution` | ✓ 见模型表 | ✓ `480p`/`720p`/`1080p`（`4k` 可传，当前价档未启用） | ✓ 仅 `720p` |
| `images` | ✓ 最多 9 · **仅 `asset://`** | ✓ 最多 9 · HTTPS（`reference_image`） | ✓ 见模型表 · HTTPS |
| `videos` | ✓ 最多 3 · **仅 `asset://`** | ✓ 最多 3 · HTTPS（`reference_video`） | ✓ 部分模型 · HTTPS |
| `audios` | ✓ 最多 3 · **仅 `asset://`** | ✓ 最多 3 · HTTPS（`reference_audio`）；须再带图或视频 | ✓ 部分模型 · HTTPS |
| `generate_audio` | ✓ | ✓（默认 `true`） | — |
| `watermark` | ✓ | ✓（默认 `false`） | — |
| `first_image` / `last_image` | — | ✓ 首帧/尾帧（计入 9 图上限，可与 `images` 同用） | ✓（与多模态参考互斥） |
| `metadata.return_last_frame` 等 | — | ✓ 见 `sd-token` 高级参数 | — |
| `auto_face` | — | — | ✓（仅 `sd2.0-933`） |

> `audios` 一般需至少再带 1 张图或 1 段视频；`sd-token` 本站会校验此规则。

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

Token 计费视频。对外仍是统一三件套（`POST/GET /v1/videos` + content 下载）；上游协议为 Seedance Official V3（`POST /api/v3/contents/generations/tasks`），本站将统一字段映射为 V3 `content[]`。

#### 模型与价格

模型：`doubao-seedance-2-0-260128`

| 输出档位 | 基准价（含视频参考） | 基准价（不含视频参考） | 本分组价（×0.78，含视频参考） | 本分组价（×0.78，不含视频参考） |
| -------- | -------------------- | ---------------------- | ----------------------------- | ------------------------------- |
| `480p`/`720p` | ¥28.00/百万 Token | ¥46.00/百万 Token | ¥21.84/百万 Token | ¥35.88/百万 Token |
| `1080p` | ¥31.00/百万 Token | ¥51.00/百万 Token | ¥24.18/百万 Token | ¥39.78/百万 Token |

- 「含视频参考」= 请求带了 `videos`；仅图/音频不算该档。
- `4k` 可通过参数校验，但当前分组 4K 价档未启用，Quote/创建可能失败。

#### 支持的参数（对齐 Official V3）

| 参数 | 是否支持 | 取值 / 说明 |
| ---- | -------- | ----------- |
| `model` | ✓ | 固定 `doubao-seedance-2-0-260128` |
| `prompt` | ✓ | 必填；映射为 `content` 中的 `type=text` |
| `seconds` / `duration` | ✓ | `4`–`15`；或 `-1` 智能时长（省略时上游默认 `5`） |
| `aspect_ratio` | ✓ | `adaptive`（默认）、`16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9` → 上游 `ratio` |
| `resolution` | ✓ | `480p`、`720p`、`1080p`；`4k` 见上 |
| `images` | ✓ | 最多 **9**；公网 **HTTPS** URL → `image_url` + `role=reference_image` |
| `videos` | ✓ | 最多 **3**；HTTPS → `video_url` + `role=reference_video` |
| `audios` | ✓ | 最多 **3**；HTTPS → `audio_url` + `role=reference_audio`；**不可单独只传音频**（须再带图或视频） |
| `first_image` | ✓ | 首帧 HTTPS → `role=first_frame`；计入 9 图上限 |
| `last_image` | ✓ | 尾帧 HTTPS → `role=last_frame`；计入 9 图上限；可与 `images`/`videos`/`audios` 同用 |
| `generate_audio` | ✓ | `true` / `false`（上游默认 `true`） |
| `watermark` | ✓ | `true` / `false`（上游默认 `false`） |
| `auto_face` | — | 不支持 |

**素材数量合计规则（本站 400 校验）**

| 字段 | 上限 | 错误码 |
| ---- | ---: | ------ |
| `first_image` + `last_image` + `images` | ≤ 9 | `invalid_images` |
| `videos` | ≤ 3 | `invalid_videos` |
| `audios` | ≤ 3 | `invalid_audios` |
| 仅 `audios` 无图无视频 | 禁止 | `invalid_audios` |

**高级参数（经 `metadata` 透传 Official V3 顶层字段）**

| `metadata` 键 | 说明 |
| ------------- | ---- |
| `return_last_frame` | `true` 时成功任务可保留末帧（本站任务详情字段见查询响应） |
| `service_tier` | 当前仅 `default` |
| `execution_expires_after` | `3600`–`259200` 秒，默认 `172800` |
| `safety_identifier` | 非空且 ≤ 64 字符 |
| `priority` | `0`–`9`，默认 `0` |

以下字段上游明确拒绝，本站创建前也会 `400`：`callback_url`、`draft`、`frames`、`seed`、`camera_fixed`、`tools`。

**示例：文生视频**

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

**示例：多模态参考（9 图 / 3 视频 / 3 音频上限内）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "沿用参考视频运镜，保持参考图人物与产品外观，动作与音频重拍同步",
    "seconds": 10,
    "aspect_ratio": "9:16",
    "resolution": "720p",
    "images": [
      "https://cdn.example.com/character.jpg",
      "https://cdn.example.com/product.jpg"
    ],
    "videos": ["https://cdn.example.com/camera-reference.mp4"],
    "audios": ["https://cdn.example.com/rhythm-reference.mp3"],
    "generate_audio": true,
    "watermark": false
  }'
```

**示例：首帧 + 尾帧**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "从清晨安静街道自然过渡到夜晚霓虹，保持主体一致",
    "seconds": 8,
    "aspect_ratio": "16:9",
    "resolution": "1080p",
    "first_image": "https://cdn.example.com/first-frame.jpg",
    "last_image": "https://cdn.example.com/last-frame.jpg",
    "generate_audio": true,
    "watermark": false
  }'
```

**示例：智能时长 + 返回末帧**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "一杯咖啡从研磨到拉花完成的连贯近景",
    "seconds": -1,
    "aspect_ratio": "adaptive",
    "resolution": "720p",
    "metadata": { "return_last_frame": true }
  }'
```

**提示**

- 提交预扣，完成后按 `usage.total_tokens` 多退少补；失败退回预扣。
- 素材须为服务端与上游可访问的 HTTPS URL；勿使用即将过期的一次性签名链。
- 接入建议：`4 秒 + 480p + 无参考`；超时 ≥ 600 秒；勿高频轮询。
- 本站对外状态仍为 OpenAI 视频风格（`queued` / `in_progress` / `completed` / `failed`）；结果经 `/v1/videos/{id}/content` 代理下载。

---

### `grok按次-x`

按次计费；接口与上文统一三件套相同（`POST /v1/videos` / 查询 / 内容下载）；参考图用公网 HTTPS URL。

#### 模型

| 模型 | 售价 | 上游 | 能力概要 |
| ---- | ---- | ---- | -------- |
| `grok-imagine-video-1.5-preview` | ¥0.70/次 | Grok Imagine Video 1.5 | 仅图生视频：必须且只能 1 张首帧图；时长 **1–15** 秒任意整数；7 种画幅；`480p`/`720p` |
| `grok-video-3` | ¥0.55/次 | 实际上游为 Grok Imagine Video 1.0 | `mode=text` 文生视频、`mode=frame` 单图首帧、`mode=ref` 概念参考（1–7 张图）；时长 **6/10/12/16/20**；画幅 `16:9`/`9:16`/`1:1`；`480p`/`720p` |

价格以控制台定价页与调用日志为准。

#### 支持的参数

| 参数 | 是否支持 | 取值 / 说明 |
| ---- | -------- | ----------- |
| `model` | ✓ | 上表模型名 |
| `prompt` | ✓ | 必填 |
| `seconds` / `duration` | ✓ | 见上表；也可用 `duration` 整数 |
| `aspect_ratio` | ✓ | 见上表；默认 `16:9` |
| `resolution` | ✓ | 仅 `480p`、`720p`；默认 `720p` |
| `images` | 视模型 | 1.5 必须 1 张；video-3 按 `mode`（text=0，frame=1，ref=1–7） |
| `mode` | 仅 `grok-video-3` | `text` / `frame` / `ref`；省略时按 `images` 数量推断 |
| `videos` / `audios` | — | 不支持 |
| `first_image` / `last_image` | — | 不支持（可用 `images` 传首帧） |
| `generate_audio` / `watermark` | — | 不支持 |

#### 示例

**图生视频（Imagine 1.5）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video-1.5-preview",
    "prompt": "让画面中的人物轻微转头，背景保持稳定，电影感光线",
    "seconds": 6,
    "aspect_ratio": "9:16",
    "resolution": "720p",
    "images": ["https://example.com/start.png"]
  }'
```

**文生视频（Video 3）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-video-3",
    "prompt": "一只猫在月球表面缓慢行走，电影感光线",
    "mode": "text",
    "seconds": 6,
    "aspect_ratio": "16:9",
    "resolution": "480p"
  }'
```

**概念参考 R2V（Video 3，1–7 张参考图）**

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-video-3",
    "prompt": "参考多张概念图生成同一角色的短镜头",
    "mode": "ref",
    "seconds": 10,
    "aspect_ratio": "1:1",
    "resolution": "720p",
    "images": [
      "https://example.com/ref1.png",
      "https://example.com/ref2.png"
    ]
  }'
```

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
