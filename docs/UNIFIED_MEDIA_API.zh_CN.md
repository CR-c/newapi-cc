# 统一图片与视频 API 接入

New API 统一提供 OpenAI 兼容的图片与视频接口。调用方只需要选择模型并提交统一请求；分组路由、请求鉴权、参数转换、任务 ID 映射和结果代理均由 New API 自动处理。

## 通用约定

- Base URL：你的 New API 站点地址，例如 `https://your-new-api.example`
- 鉴权：`Authorization: Bearer <New API token>`
- 模型列表：`GET /v1/models`
- API token 所属分组决定可见模型、价格和平台路由
- 分组只绑定在 API token 上；生产 API 请求体不要传 `group`

当前控制台公开分组为 `64生图`、`生图`、`grok按次`、`video-dddd` 和 `sd-video`。分组、模型和价格可能由管理员调整，应以控制台定价页及同一 token 调用 `GET /v1/models` 的实际返回为准。

## 图片生成

统一端点：

```text
POST /v1/images/generations
```

统一请求：

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

| 字段              | 类型               | 说明                                             |
| ----------------- | ------------------ | ------------------------------------------------ |
| `model`           | string             | 必填，使用 `/v1/models` 返回的模型名             |
| `prompt`          | string             | 必填，生成提示词                                 |
| `size`            | string             | 尺寸、比例映射尺寸或 `auto`，具体能力见模型表    |
| `n`               | integer            | 生成数量，演练场按模型限制可选范围               |
| `quality`         | string             | 支持时可选 `low`、`medium`、`high`               |
| `background`      | string             | 支持时可选 `opaque`、`transparent`               |
| `image`           | string 或 string[] | 可选参考图，支持 HTTP(S) URL、Base64 或 data URL |
| `input_reference` | string 或 string[] | `image` 的兼容别名                               |
| `response_format` | string             | `url` 或 `b64_json`                              |

统一响应：

```json
{
  "created": 1783866456,
  "data": [{ "url": "https://example.com/result.png" }]
}
```

当 `response_format` 为 `b64_json` 时，`data` 项返回 `b64_json`。

### `64生图` 分组

| 模型                             | 价格       | 分辨率                     | 比例                    | 其他参数                                                |
| -------------------------------- | ---------- | -------------------------- | ----------------------- | ------------------------------------------------------- |
| `gemini-3-pro-image-preview`     | `$0.15/次` | OpenAI 兼容端点固定 2K     | 标准 10 比例            | `n`、参考图、`response_format`                          |
| `gemini-3.1-flash-image-preview` | `$0.10/次` | OpenAI 兼容端点固定 2K     | 14 比例，额外支持超长图 | `n`、参考图、`response_format`                          |
| `gpt-image-2`                    | `$0.10/次` | 1K、2K、4K、自动、精确尺寸 | 标准 10 比例            | `quality`、`background`、`n`、参考图、`response_format` |

`64生图` 标准比例：

```text
1:1 16:9 9:16 4:3 3:4 21:9 3:2 2:3 5:4 4:5
```

`gemini-3.1-flash-image-preview` 额外支持：

```text
1:4 1:8 4:1 8:1
```

`64生图` 的两个 preview 图片模型经统一图片端点调用时固定输出 2K。`size` 用于选择最接近的宽高比，不用于切换 1K/2K/4K。参考图最多 14 张。

`gpt-image-2` 常用预设：

| 比例 | 1K          | 2K          | 4K          |
| ---- | ----------- | ----------- | ----------- |
| 1:1  | `1024x1024` | `2048x2048` | `2480x2480` |
| 16:9 | `1280x720`  | `2560x1440` | `3328x1872` |
| 9:16 | `720x1280`  | `1440x2560` | `1872x3328` |

`gpt-image-2` 也支持精确尺寸。宽高均为 16 的倍数时可直接使用，例如 `2160x3840`；`size: "auto"` 由平台自动选择。

示例：

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

## 视频生成

所有视频生成统一使用异步任务流程。

### 1. 提交任务

```text
POST /v1/videos
```

```json
{
  "model": "veo-3.1",
  "prompt": "a drone shot over mountains",
  "size": "1920x1080",
  "seconds": 8,
  "images": ["https://example.com/first-frame.png"]
}
```

统一字段：

| 字段             | 类型              | 说明                                                 |
| ---------------- | ----------------- | ---------------------------------------------------- |
| `model`          | string            | 必填，视频模型名                                     |
| `prompt`         | string            | 必填，视频提示词                                     |
| `seconds`        | integer 或 string | 推荐的视频时长字段，数字和数字字符串均兼容           |
| `duration`       | integer 或 string | `seconds` 的旧版兼容别名；不要同时传入不同值         |
| `size`           | string            | OpenAI 风格尺寸，例如 `1920x1080`                    |
| `aspect_ratio`   | string            | 模型支持时使用，例如 `16:9`                          |
| `resolution`     | string            | 模型支持时使用，例如 `720p`、`1080p`                 |
| `image`          | string            | 单张参考图兼容字段                                   |
| `images`         | string[]          | 参考图片                                             |
| `videos`         | string[]          | 支持时传入参考视频                                   |
| `audios`         | string[]          | 支持时传入参考音频                                   |
| `generate_audio` | boolean           | 支持时控制是否生成音频                               |
| `watermark`      | boolean           | 支持时控制水印                                       |
| `first_image`    | string            | 支持时传入首帧图片 URL，必须与 `last_image` 成对使用 |
| `last_image`     | string            | 支持时传入尾帧图片 URL，不能与参考素材数组混用       |
| `auto_face`      | boolean           | 支持时控制自动人脸处理                               |

提交成功后返回统一任务 ID：

```json
{
  "id": "video_xxx",
  "status": "queued",
  "model": "veo-3.1",
  "created_at": 1783866456
}
```

### 2. 查询状态

```text
GET /v1/videos/{id}
```

```json
{
  "id": "video_xxx",
  "status": "in_progress",
  "model": "veo-3.1",
  "progress": 35
}
```

状态统一为 `queued`、`in_progress`、`completed` 或 `failed`。建议每 5 至 10 秒轮询一次，视频任务超时设置不低于 600 秒。

任务完成后，可使用响应中的结果 URL，或通过统一内容代理下载：

```text
GET /v1/videos/{id}/content
```

### `sd-video` 分组

该分组仍使用上述统一创建、查询和内容接口。平台自动完成参数转换、任务 ID 映射和最终视频代理。

| 模型            | 价格      | 时长        | 比例                                        | 参考素材上限         |
| --------------- | --------- | ----------- | ------------------------------------------- | -------------------- |
| `sd2.0-933`     | 7 元/次   | 4–15 秒     | `21:9`、`16:9`、`4:3`、`1:1`、`3:4`、`9:16` | 9 图、3 视频、3 音频 |
| `sd2.0-431`     | 6 元/次   | 4–15 秒     | `16:9`、`9:16`、`1:1`                       | 4 图、3 视频、1 音频 |
| `sd2.0fast-431` | 5 元/次   | 10 或 15 秒 | `16:9`、`9:16`、`1:1`                       | 4 图、3 视频、1 音频 |
| `sd2.0-903`     | 6.5 元/次 | 10 或 15 秒 | `16:9`、`9:16`、`1:1`                       | 9 图、0 视频、3 音频 |
| `sd2.0fast-903` | 5.5 元/次 | 10 或 15 秒 | `16:9`、`9:16`、`1:1`                       | 9 图、0 视频、3 音频 |

所有模型当前仅支持 `720p`。`sd2.0-903` 和 `sd2.0fast-903` 使用音频参考时必须同时提供至少一张参考图。除 `sd2.0-933` 外，提示词最多 5000 个字符。

多媒体参考示例：

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

首尾帧模式使用 `first_image` 和 `last_image`，必须成对传入，且不能同时传 `images`、`videos` 或 `audios`。

## 统一处理规则

调用方看到的接口保持不变，New API 按以下顺序处理：

1. 根据 New API token 确定可用分组。
2. 根据分组、模型和 abilities 选择渠道。
3. 根据模型能力规范化统一字段。
4. 为异步视频生成统一任务 ID。
5. 统一记录额度、价格、日志、任务状态和结果地址。

演练场遵循同一逻辑：图片使用 `/pg/images/generations`，视频使用 `/pg/videos`。界面只根据当前分组和模型展示可用参数，提交仍进入同一套网关路由与计费流程。

## 兼容性说明

- 不同分组可以为同名模型配置不同价格。
- 未被当前模型配置支持的字段不会由演练场发送。
- API 调用方仍需按所选模型的能力填写字段；未支持的参数不会生效。
- 模型与价格以同一 token 调用 `/v1/models` 的返回、站点定价页和管理员配置为准。
- 错误响应可能使用 OpenAI 风格的 `error.message`，也可能使用视频任务的顶层 `code` 与 `message`；客户端应兼容两种结构。
