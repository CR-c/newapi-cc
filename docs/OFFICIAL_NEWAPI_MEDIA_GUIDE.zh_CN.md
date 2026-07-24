# 官方 New API 对接本站图片与视频

本文说明如何用 [官方 New API](https://github.com/QuantumNous/new-api) 兼容协议，对接本站的 **图片生成** 与 **视频生成** 能力。

协议与官方 New API / OpenAI 兼容接口一致；模型名、分组、素材传法以本站控制台与 token 绑定的 `/v1/models` 为准。

> **在线文档（推荐）**  
> - 统一媒体详表：`/docs.html`  
> - 本对接指南：`/docs-official.html`  
> - 同一入口，顶部切换分页，不整页连在一起展示  
>
> 更细的分组模型表、参数能力与价格见：[统一图片与视频 API 接入](./UNIFIED_MEDIA_API.zh_CN.md)  
> 官方文档入口：[Images](https://docs.newapi.pro/zh/docs/api/ai-model/images/openai/post-v1-images-generations) · [Videos](https://docs.newapi.pro/zh/docs/api/ai-model/videos/sora/createvideo)

---

## 1. 快速开始

| 项 | 说明 |
| -- | ---- |
| Base URL | 本站地址，例如 `https://your-new-api.example`（不要带末尾 `/`） |
| API Key | 控制台创建的 token，格式一般为 `sk-...` |
| 鉴权头 | `Authorization: Bearer <API Key>` |
| 模型列表 | `GET /v1/models`（返回当前 token 可用模型） |
| 分组 | **只绑定在 token 上**；请求体 **不要** 传 `group` |
| Content-Type | JSON 请求用 `application/json` |

```bash
export NEWAPI_BASE_URL="https://your-new-api.example"
export NEWAPI_API_KEY="sk-xxxxxxxx"
```

### 与官方 New API 的关系

| 能力 | 官方 New API 兼容路径 | 本站是否支持 | 说明 |
| ---- | --------------------- | ------------ | ---- |
| 图片生成 | `POST /v1/images/generations` | ✅ | OpenAI Images 格式；可带参考图 `image` |
| 视频（OpenAI Videos） | `POST /v1/videos` · `GET /v1/videos/{id}` · `GET /v1/videos/{id}/content` | ✅ **推荐** | 异步任务：创建 → 轮询 → 下载 |
| 视频（New API 通用） | `POST /v1/video/generations` · `GET /v1/video/generations/{id}` | ✅ | 与官方 New API 视频通用接口一致 |
| 可灵格式 | `/kling/v1/videos/...` | ✅ | 需上游/渠道支持 |
| 即梦官方格式 | `POST /jimeng/` | ✅ | 需上游/渠道支持 |
| 素材库 | `POST /pg/assets` · `POST /v1/sd/assets` · `GET /v1/sd/assets/{id}` | ✅（部分视频分组） | `video-dddd` 等要求 `asset://` |

把任意 OpenAI / New API 兼容客户端的 **Base URL** 改成本站地址、**API Key** 改成本站 token，即可调用。

---

## 2. 图片生成

### 2.1 接口

```text
POST {BASE_URL}/v1/images/generations
```

同步接口：请求返回前会等待生成完成。客户端读取超时建议 **≥ 300 秒**。

### 2.2 请求体（核心字段）

| 字段 | 类型 | 必填 | 说明 |
| ---- | ---- | ---- | ---- |
| `model` | string | 是 | 模型名，以 `GET /v1/models` 为准 |
| `prompt` | string | 是 | 提示词 |
| `size` | string | 否 | 如 `1024x1024`、`2K`、`auto`（按模型） |
| `n` | integer | 否 | 生成张数；接入建议先用 `1` |
| `quality` | string | 否 | 如 `low` / `medium` / `high`（模型支持时） |
| `background` | string | 否 | 如 `opaque` / `transparent`（模型支持时） |
| `image` | string 或 string[] | 否 | 参考图：公网 HTTPS、Base64 或 data URL |
| `input_reference` | string 或 string[] | 否 | `image` 的兼容别名 |
| `watermark` | boolean | 否 | 部分模型支持 |
| `response_format` | string | 否 | `url`（推荐）或 `b64_json` |

### 2.3 响应

```json
{
  "created": 1783866456,
  "data": [
    { "url": "https://example.com/result.png" }
  ]
}
```

`response_format: "b64_json"` 时，`data[]` 中为 `b64_json` 字段。  
`url` 通常有时效，请尽快下载并自行持久化。

### 2.4 curl 示例

**文生图**

```bash
curl "$NEWAPI_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3-pro-image-preview",
    "prompt": "a cat on the moon, cinematic lighting",
    "size": "1024x1024",
    "n": 1,
    "response_format": "url"
  }'
```

**参考图（图生图 / 多参考）**

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

### 2.5 OpenAI Python SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-xxxxxxxx",
    base_url="https://your-new-api.example/v1",
)

resp = client.images.generate(
    model="gpt-image-2",
    prompt="a red apple on a white table",
    size="2048x2048",
    n=1,
    response_format="url",
)
print(resp.data[0].url)
```

### 2.6 图片分组速查（本站）

| 分组（绑在 token） | 典型模型 | 参考图传法 |
| ------------------ | -------- | ---------- |
| `dddd-sd-图` | Seedream 系列 | HTTPS / Base64 |
| `64生图` | Gemini 图像、`gpt-image-2` 等 | HTTPS / Base64 |

完整 `size` 档位、参考图上限与价格见 [UNIFIED_MEDIA_API.zh_CN.md](./UNIFIED_MEDIA_API.zh_CN.md)。

---

## 3. 视频生成（推荐：OpenAI Videos 格式）

官方 New API 与 OpenAI Videos 对齐。本站推荐使用下列三件套：

| 步骤 | 方法 | 路径 | 说明 |
| ---- | ---- | ---- | ---- |
| 创建任务 | `POST` | `/v1/videos` | 立即返回任务 ID，异步生成 |
| 查询状态 | `GET` | `/v1/videos/{task_id}` | `queued` / `in_progress` / `completed` / `failed` |
| 下载成片 | `GET` | `/v1/videos/{task_id}/content` | 成功后拉取 MP4 |

轮询间隔建议 **5–10 秒**；整条链路客户端超时建议 **≥ 600 秒**。任务未结束前不要重复提交同一业务请求。

### 3.1 创建任务

```text
POST {BASE_URL}/v1/videos
Content-Type: application/json
```

**核心字段**

| 字段 | 类型 | 必填 | 说明 |
| ---- | ---- | ---- | ---- |
| `model` | string | 是 | 视频模型名 |
| `prompt` | string | 是 | 提示词 |
| `seconds` | integer / string | 否 | 推荐时长字段（秒） |
| `duration` | integer / string | 否 | `seconds` 别名；勿与 `seconds` 传不同值 |
| `aspect_ratio` | string | 否 | 如 `16:9`、`9:16`、`1:1` |
| `resolution` | string | 否 | 如 `480p`、`720p`、`1080p` |
| `size` | string | 否 | 如 `1920x1080`（部分模型） |
| `images` | string[] | 否 | 参考图列表 |
| `image` | string | 否 | 单图兼容字段，等价于单元素 `images` |
| `videos` | string[] | 否 | 参考视频 |
| `audios` | string[] | 否 | 参考音频（通常不能单独使用） |
| `generate_audio` | boolean | 否 | 是否生成有声视频 |
| `watermark` | boolean | 否 | 是否水印 |
| `first_image` + `last_image` | string | 否 | 首尾帧（部分模型；与多模态参考互斥） |

**创建响应示例**

```json
{
  "id": "video_xxx",
  "object": "video",
  "status": "queued",
  "model": "dreamina-seedance-2-0-mini-hc",
  "progress": 0,
  "created_at": 1783866456
}
```

### 3.2 查询与下载

```bash
# 查询
curl "$NEWAPI_BASE_URL/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $NEWAPI_API_KEY"

# 下载（status=completed 后）
curl -L "$NEWAPI_BASE_URL/v1/videos/$TASK_ID/content" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -o output.mp4
```

查询中进度字段为 `progress`（0–100）。失败时可能带 `error.message` / `error.code`。

### 3.3 curl：文生视频

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

### 3.4 curl：图生视频（HTTPS 参考图）

适用于 `sd-token`、`sd-video` 等允许公网 URL 的分组：

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "让画面中的人物自然转身并看向镜头",
    "seconds": 4,
    "aspect_ratio": "9:16",
    "resolution": "480p",
    "images": ["https://example.com/person.jpg"],
    "generate_audio": false,
    "watermark": false
  }'
```

### 3.5 完整轮询脚本（bash）

```bash
#!/usr/bin/env bash
set -euo pipefail

CREATE=$(curl -sS "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-mini-hc",
    "prompt": "a drone shot over mountains",
    "seconds": 4,
    "aspect_ratio": "16:9",
    "resolution": "480p"
  }')

TASK_ID=$(echo "$CREATE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "task_id=$TASK_ID"

for i in $(seq 1 120); do
  STATUS_JSON=$(curl -sS "$NEWAPI_BASE_URL/v1/videos/$TASK_ID" \
    -H "Authorization: Bearer $NEWAPI_API_KEY")
  STATUS=$(echo "$STATUS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")
  PROGRESS=$(echo "$STATUS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('progress',0))")
  echo "[$i] status=$STATUS progress=$PROGRESS"
  case "$STATUS" in
    completed)
      curl -sSL "$NEWAPI_BASE_URL/v1/videos/$TASK_ID/content" \
        -H "Authorization: Bearer $NEWAPI_API_KEY" \
        -o "video_${TASK_ID}.mp4"
      echo "saved video_${TASK_ID}.mp4"
      exit 0
      ;;
    failed)
      echo "$STATUS_JSON"
      exit 1
      ;;
  esac
  sleep 5
done

echo "timeout waiting for task"
exit 1
```

### 3.6 Python 示例（requests）

```python
import time
import requests

BASE = "https://your-new-api.example"
KEY = "sk-xxxxxxxx"
H = {"Authorization": f"Bearer {KEY}", "Content-Type": "application/json"}

r = requests.post(
    f"{BASE}/v1/videos",
    headers=H,
    json={
        "model": "dreamina-seedance-2-0-mini-hc",
        "prompt": "a drone shot over mountains",
        "seconds": 4,
        "aspect_ratio": "16:9",
        "resolution": "480p",
    },
    timeout=60,
)
r.raise_for_status()
task_id = r.json()["id"]

while True:
    s = requests.get(f"{BASE}/v1/videos/{task_id}", headers=H, timeout=60)
    s.raise_for_status()
    body = s.json()
    status = body.get("status")
    print(status, body.get("progress"))
    if status == "completed":
        content = requests.get(
            f"{BASE}/v1/videos/{task_id}/content",
            headers={"Authorization": f"Bearer {KEY}"},
            timeout=300,
        )
        content.raise_for_status()
        with open(f"{task_id}.mp4", "wb") as f:
            f.write(content.content)
        break
    if status == "failed":
        raise RuntimeError(body)
    time.sleep(5)
```

### 3.7 视频分组速查（本站）

| 分组 | 计费 | 参考素材传法 | 备注 |
| ---- | ---- | ------------ | ---- |
| `video-dddd` | 按次 | **仅** `asset://素材ID`（须先走素材库） | 多模态参考能力强 |
| `sd-token` | Token | 公网 HTTPS URL（最多 9 图 + 3 视频 + 3 音频；支持首尾帧） | 提交预扣，完成后按用量结算；Official V3 映射 |
| `sd-video` | 按次 | 公网 HTTPS URL | 部分模型支持首尾帧 |
| `grok按次` | 按次 | 公网 HTTPS URL | Grok 图/文生视频（既有） |
| `grok按次-x` | 按次 | 公网 HTTPS URL | Grok Imagine 1.5 / Grok Video 3 |

素材字段上限、分辨率与价格见 [UNIFIED_MEDIA_API.zh_CN.md](./UNIFIED_MEDIA_API.zh_CN.md)。

---

## 4. 素材库（`video-dddd` 等分组必用）

当 token 绑定的分组要求参考素材为 `asset://...` 时，**不能**在 `images` / `videos` / `audios` 里直接塞 HTTPS 链接，请按下列流程：

```text
本地文件
  → POST /pg/assets          上传，得到临时 URL
  → POST /v1/sd/assets       注册素材，得到 data.Id
  → GET  /v1/sd/assets/{id}  轮询至 Status = Active
  → POST /v1/videos          images/videos/audios 填 asset://{Id}
```

| 素材类型 | 上传 `kind` | 注册 `AssetType` | 写入字段 |
| -------- | ----------- | ---------------- | -------- |
| 图片 | `image` | `Image` | `images` |
| 视频 | `video` | `Video` | `videos` |
| 音频 | `audio` | `Audio` | `audios` |

```bash
# 1) 上传
curl "$NEWAPI_BASE_URL/pg/assets" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -F "kind=image" \
  -F "file=@./reference-1.png"

# 2) 注册（URL 换成上一步返回的可访问地址）
curl "$NEWAPI_BASE_URL/v1/sd/assets" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "URL": "https://your-new-api.example/pg/assets/<upload_id>/reference-1.png",
    "Name": "reference_1",
    "AssetType": "Image"
  }'

# 3) 查询至 Active
curl "$NEWAPI_BASE_URL/v1/sd/assets/<asset_id>" \
  -H "Authorization: Bearer $NEWAPI_API_KEY"

# 4) 创建视频
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dreamina-seedance-2-0-mini-hc",
    "prompt": "让人物自然转身并看向镜头",
    "seconds": 4,
    "aspect_ratio": "9:16",
    "resolution": "480p",
    "images": ["asset://<asset_id>"],
    "generate_audio": false,
    "watermark": false
  }'
```

注意：

- 类型必须匹配（图不能塞进 `videos`）。
- 音频一般不能单独使用：至少再带 1 张图或 1 段视频。
- 先确认素材 `Active`，再创建视频，便于区分「素材失败」与「视频任务失败」。

---

## 5. 兼容路径：`/v1/video/generations`

与官方 New API 通用视频接口一致，适合从旧文档 / Kling·Jimeng 示例迁移的客户端：

| 步骤 | 方法 | 路径 |
| ---- | ---- | ---- |
| 创建 | `POST` | `/v1/video/generations` |
| 查询 | `GET` | `/v1/video/generations/{task_id}` |

```bash
curl "$NEWAPI_BASE_URL/v1/video/generations" \
  -H "Authorization: Bearer $NEWAPI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-video-model",
    "prompt": "宇航员在月球上行走，电影级画质",
    "duration": 5,
    "image": "https://example.com/ref.jpg"
  }'
```

**新接入优先使用 `/v1/videos` 三件套**（状态语义与下载路径更清晰）。  
若业务已对接 `/v1/video/generations`，本站同样兼容，无需强行改路径。

---

## 6. 客户端 / 工具对接清单

### 6.1 通用 HTTP 客户端

1. Base URL = 本站根地址（或带 `/v1`，视 SDK 是否自动拼路径）。
2. Header：`Authorization: Bearer sk-...`
3. 图片：`POST /v1/images/generations`
4. 视频：`POST /v1/videos` → 轮询 `GET /v1/videos/{id}` → `GET /v1/videos/{id}/content`

### 6.2 OpenAI 官方 SDK

- `base_url` 指向本站 `.../v1`
- `api_key` 使用本站 token
- 图片可用 `client.images.generate(...)`
- 视频部分 SDK 版本尚未完整封装 Videos API：请直接 HTTP 调用 `/v1/videos` 三件套

### 6.3 Cherry Studio / 其他桌面客户端

1. 提供商类型选 **OpenAI 兼容**
2. API 地址填本站 Base URL
3. 密钥填本站 token
4. 在模型列表中启用本站图片 / 视频模型名（与 `/v1/models` 一致）

### 6.4 自建后端转发

- 不要在 body 里写 `group`；换能力请换 token。
- 图片长请求与视频轮询要分别配置超时。
- 对上游返回的 `url` / 成片 content 做本地落盘，勿依赖临时链接长期可用。
- 错误体可能是 `error.message`，也可能是顶层 `code` + `message`，建议两种都解析。

---

## 7. 常见问题

| 现象 | 可能原因 | 处理 |
| ---- | -------- | ---- |
| 401 / 鉴权失败 | Key 错误或未带 Bearer | 检查 `Authorization: Bearer sk-...` |
| 模型不存在 / 无可用渠道 | token 分组与模型不匹配 | `GET /v1/models` 核对；换绑定对应分组的 token |
| 图片超时 | 同步生成较慢 | 超时 ≥ 300s；勿重复连打 |
| 视频一直 `in_progress` | 上游排队或生成中 | 5–10s 轮询；总等待 ≥ 10 分钟再判定失败 |
| `invalid_images` / 素材错误 | 分组要求 `asset://` 却传了 HTTPS，或类型不匹配 | 走素材库；确认 `Active` |
| 参考图无效 | 非公网可达 URL、过大或格式不支持 | 用公网 HTTPS / Base64；控制张数与大小 |
| 扣费异常 | 预扣未退或任务失败 | 失败应自动退预扣；以控制台日志为准 |
| body 里传了 `group` | 本站分组只认 token | 去掉 `group`，换 token |

---

## 8. 接入检查清单

- [ ] 已拿到本站 Base URL 与 token（`sk-...`）
- [ ] `GET /v1/models` 能看到目标图片 / 视频模型
- [ ] 图片：`POST /v1/images/generations` 成功返回 `data[].url` 或 `b64_json`
- [ ] 视频：`POST /v1/videos` 返回 `id`，轮询至 `completed`，能下载 `content`
- [ ] 若使用 `video-dddd`：素材上传 → 注册 → `Active` → `asset://` 创建成功
- [ ] 客户端超时：图片 ≥ 300s，视频轮询链路 ≥ 600s
- [ ] 未在请求体中传递 `group`
- [ ] 结果 URL / 成片已自行持久化

---

## 9. 相关链接

| 资源 | 链接 |
| ---- | ---- |
| 官方 New API 仓库 | https://github.com/QuantumNous/new-api |
| 本站统一媒体能力详表 | [UNIFIED_MEDIA_API.zh_CN.md](./UNIFIED_MEDIA_API.zh_CN.md) |
| Sub2API 视频渠道预置（运维） | [SUB2API_VIDEO_PRESET.zh_CN.md](./SUB2API_VIDEO_PRESET.zh_CN.md) |
| 官方图片文档 | https://docs.newapi.pro/zh/docs/api/ai-model/images/openai/post-v1-images-generations |
| 官方视频文档 | https://docs.newapi.pro/zh/docs/api/ai-model/videos/sora/createvideo |
| OpenAI Images 参考 | https://platform.openai.com/docs/api-reference/images |
| OpenAI Videos 参考 | https://platform.openai.com/docs/api-reference/videos |

模型、分组与价格可能随运营调整；**以当前 token 的 `/v1/models` 与控制台定价页为准**。
