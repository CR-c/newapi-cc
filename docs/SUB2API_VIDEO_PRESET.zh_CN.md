# Sub2API 视频模型预置

这个分支内置了面向 Sub2API 的视频模型预置。客户部署 `newapi-cc` 后，只需要提供一个可访问 Sub2API 的上游地址和 API key，系统启动时会自动创建视频渠道、分组和模型预填组。

## 启用方式

在 `.env` 或部署平台环境变量中设置：

```bash
SUB2API_VIDEO_PRESET_ENABLED=true
SUB2API_BASE_URL=https://your-sub2api.example
SUB2API_API_KEY=sk-your-sub2api-upstream-key
```

如果使用仓库自带 `docker-compose.yml`，它会从当前源码构建 `newapi-cc:latest`，不会拉取官方 `new-api` 镜像。

```bash
docker compose up -d --build
```

## 自动创建的分组

启动后会创建以下用户可选分组，并加入自动分组：

| 分组 | 模型 |
| --- | --- |
| `sub2api-jimeng-video` | `video-ds-2.0`, `video-ds-2.0-fast`, `as-sd2.0-fast` |
| `sub2api-grok-video` | `grok-imagine-video`, `grok-imagine-video-1.5-preview` |
| `sub2api-grok-video-per-request` | `grok-image-video`, `grok-video-1.5` |
| `sub2api-jimeng-nsfw-video` | `dreamina-seedance-2-0-hc`, `dreamina-seedance-2-0-fast-hc` |

每个分组都会对应一个启用状态的 Sora/OpenAI 视频兼容渠道，统一转发到 `SUB2API_BASE_URL` 的 `/v1/videos`、`/v1/videos/{task_id}`、`/v1/videos/{task_id}/content` 兼容接口。

## 对用户开放

管理员可以给用户发放 New API token。用户调用时按 New API/OpenAI 兼容方式请求：

```bash
curl "$NEWAPI_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $NEWAPI_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"video-ds-2.0-fast","prompt":"A cinematic city night video"}'
```

预置逻辑是幂等的：重复启动会更新带有 `sub2api-video-preset` 标签的预置渠道，不会删除客户自己新增的其他渠道。
