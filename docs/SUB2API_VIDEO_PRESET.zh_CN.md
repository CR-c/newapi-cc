# Sub2API 视频模型快速接入

这个仓库不改 New API 的启动流程，也不要求额外环境变量。最快接入方式是：先按 New API 原方式部署，然后运行一次导入脚本，把 Sub2API 作为上游视频渠道写入 New API 数据库。

## 一条命令导入

```bash
tools/sub2api-video-preset.sh \
  --container postgres \
  --dsn "postgresql://root:123456@localhost:5432/new-api?sslmode=disable" \
  --base-url "https://your-sub2api.example" \
  --api-key "sk-your-sub2api-key"
```

如果 PostgreSQL 已经暴露到宿主机，或在有 `psql` 的运维机器上执行，也可以不走容器：

```bash
tools/sub2api-video-preset.sh \
  --dsn "postgresql://root:123456@127.0.0.1:5432/new-api?sslmode=disable" \
  --base-url "https://your-sub2api.example" \
  --api-key "sk-your-sub2api-key"
```

## 导入内容

脚本只写 New API 数据库，不改服务代码、不改启动参数。它会幂等创建或更新：

| 类型 | 内容 |
| --- | --- |
| 分组 | `sub2api-jimeng-video`, `sub2api-grok-video`, `sub2api-grok-video-per-request`, `sub2api-jimeng-nsfw-video` |
| 模型组 | `sub2api-video-models` |
| 渠道 | 4 个带 `sub2api-video-preset` 标签的 OpenAI/Sora 视频兼容渠道 |
| 能力 | 每个分组对应的视频模型 abilities |

模型清单：

| 分组 | 模型 |
| --- | --- |
| `sub2api-jimeng-video` | `video-ds-2.0`, `video-ds-2.0-fast`, `sd2.0-fast`, `as-sd2.0-fast` |
| `sub2api-grok-video` | `grok-imagine-video`, `grok-imagine-video-1.5-preview` |
| `sub2api-grok-video-per-request` | `grok-image-video`, `grok-video-1.5` |
| `sub2api-jimeng-nsfw-video` | `dreamina-seedance-2-0-hc`, `dreamina-seedance-2-0-fast-hc` |

导入后如果 New API 开启了渠道缓存，重启一次 New API，让新渠道立即进入缓存。

## 用户调用

管理员给用户发放 New API token 后，用户按 OpenAI 视频兼容接口调用：

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"video-ds-2.0-fast","prompt":"A cinematic city night video"}'
```

脚本可重复运行。它只更新带 `sub2api-video-preset` 标签的预置渠道和相关分组，不删除客户自己新增的渠道。
