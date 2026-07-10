#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tools/sub2api-video-preset.sh --dsn POSTGRES_DSN --base-url SUB2API_URL --api-key SUB2API_KEY
  tools/sub2api-video-preset.sh --container postgres --dsn POSTGRES_DSN --base-url SUB2API_URL --api-key SUB2API_KEY

Example:
  tools/sub2api-video-preset.sh \
    --dsn "postgresql://root:123456@127.0.0.1:5432/new-api?sslmode=disable" \
    --base-url "https://your-sub2api.example" \
    --api-key "sk-your-sub2api-key"

The script writes only New API database rows: groups/options, prefill group,
and OpenAI-compatible Sora video channels pointing at Sub2API.
EOF
}

dsn=""
base_url=""
api_key=""
container=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dsn)
      dsn="${2:-}"
      shift 2
      ;;
    --base-url)
      base_url="${2:-}"
      shift 2
      ;;
    --api-key)
      api_key="${2:-}"
      shift 2
      ;;
    --container)
      container="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$dsn" || -z "$base_url" || -z "$api_key" ]]; then
  usage >&2
  exit 2
fi

if [[ -n "$container" ]] && ! command -v docker >/dev/null 2>&1; then
  echo "docker is required when --container is used." >&2
  exit 1
fi

if [[ -z "$container" ]] && ! command -v psql >/dev/null 2>&1; then
  echo "psql is required. Install PostgreSQL client tools first." >&2
  exit 1
fi

base_url="${base_url%/}"
tmp_sql="$(mktemp)"
trap 'rm -f "$tmp_sql"' EXIT

cat >"$tmp_sql" <<'SQL'
BEGIN;

CREATE TEMP TABLE sub2api_video_preset_groups (
  name text PRIMARY KEY,
  description text NOT NULL,
  models text NOT NULL
) ON COMMIT DROP;

INSERT INTO sub2api_video_preset_groups (name, description, models) VALUES
  ('sub2api-jimeng-video', 'Sub2API 即梦视频', 'video-ds-2.0,video-ds-2.0-fast,sd2.0-fast,as-sd2.0-fast'),
  ('sub2api-grok-video', 'Sub2API Grok 视频', 'grok-imagine-video,grok-imagine-video-1.5-preview'),
  ('sub2api-grok-video-per-request', 'Sub2API Grok 视频按次', 'grok-image-video,grok-video-1.5'),
  ('sub2api-jimeng-nsfw-video', 'Sub2API 即梦 NSFW 视频', 'dreamina-seedance-2-0-hc,dreamina-seedance-2-0-fast-hc');

WITH current_group_ratio AS (
  SELECT COALESCE((SELECT value::jsonb FROM options WHERE key = 'GroupRatio'), '{}'::jsonb) AS value
), merged_group_ratio AS (
  SELECT value || jsonb_object_agg(name, 1) AS value
  FROM current_group_ratio, sub2api_video_preset_groups
  GROUP BY value
)
INSERT INTO options (key, value)
SELECT 'GroupRatio', value::text FROM merged_group_ratio
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

WITH current_user_groups AS (
  SELECT COALESCE((SELECT value::jsonb FROM options WHERE key = 'UserUsableGroups'), '{}'::jsonb) AS value
), merged_user_groups AS (
  SELECT value || jsonb_object_agg(name, description) AS value
  FROM current_user_groups, sub2api_video_preset_groups
  GROUP BY value
)
INSERT INTO options (key, value)
SELECT 'UserUsableGroups', value::text FROM merged_user_groups
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

WITH current_auto_groups AS (
  SELECT COALESCE((SELECT value::jsonb FROM options WHERE key = 'AutoGroups'), '[]'::jsonb) AS value
), merged_auto_groups AS (
  SELECT jsonb_agg(DISTINCT group_name ORDER BY group_name) AS value
  FROM (
    SELECT jsonb_array_elements_text(value) AS group_name FROM current_auto_groups
    UNION ALL
    SELECT name AS group_name FROM sub2api_video_preset_groups
  ) groups
)
INSERT INTO options (key, value)
SELECT 'AutoGroups', value::text FROM merged_auto_groups
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

INSERT INTO prefill_groups (name, type, items, description, created_time, updated_time, deleted_at)
VALUES (
  'sub2api-video-models',
  'model',
  (SELECT jsonb_agg(DISTINCT model ORDER BY model)::json FROM (
    SELECT unnest(string_to_array(models, ',')) AS model FROM sub2api_video_preset_groups
  ) models),
  'Models exposed by the Sub2API video preset',
  EXTRACT(EPOCH FROM now())::bigint,
  EXTRACT(EPOCH FROM now())::bigint,
  NULL
)
ON CONFLICT (name) WHERE deleted_at IS NULL DO UPDATE SET
  type = EXCLUDED.type,
  items = EXCLUDED.items,
  description = EXCLUDED.description,
  updated_time = EXCLUDED.updated_time;

CREATE TEMP TABLE sub2api_video_preset_channels AS
SELECT
  55 AS type,
  :'api_key' AS key,
  1 AS status,
  'Sub2API - ' || description AS name,
  10::bigint AS weight,
  EXTRACT(EPOCH FROM now())::bigint AS created_time,
  :'base_url' AS base_url,
  models,
  name AS "group",
  '{}' AS model_mapping,
  '{}' AS status_code_mapping,
  100::bigint AS priority,
  0 AS auto_ban,
  'sub2api-video-preset' AS tag,
  '{"is_multi_key":false,"multi_key_size":0,"multi_key_status_list":null,"multi_key_polling_index":0,"multi_key_mode":""}'::json AS channel_info
FROM sub2api_video_preset_groups;

UPDATE channels c SET
  type = p.type,
  key = p.key,
  status = p.status,
  weight = p.weight,
  base_url = p.base_url,
  models = p.models,
  "group" = p."group",
  model_mapping = p.model_mapping,
  status_code_mapping = p.status_code_mapping,
  priority = p.priority,
  auto_ban = p.auto_ban,
  tag = p.tag,
  channel_info = p.channel_info
FROM sub2api_video_preset_channels p
WHERE c.tag = 'sub2api-video-preset' AND c.name = p.name;

INSERT INTO channels (
  type, key, status, name, weight, created_time, base_url, models, "group",
  model_mapping, status_code_mapping, priority, auto_ban, tag, channel_info
)
SELECT
  p.type, p.key, p.status, p.name, p.weight, p.created_time, p.base_url, p.models, p."group",
  p.model_mapping, p.status_code_mapping, p.priority, p.auto_ban, p.tag, p.channel_info
FROM sub2api_video_preset_channels p
WHERE NOT EXISTS (
  SELECT 1 FROM channels c WHERE c.tag = 'sub2api-video-preset' AND c.name = p.name
);

DELETE FROM abilities WHERE channel_id IN (
  SELECT id FROM channels WHERE tag = 'sub2api-video-preset'
);

WITH preset_channels AS (
  SELECT id, status, priority, weight, tag, models, "group"
  FROM channels
  WHERE tag = 'sub2api-video-preset'
), expanded AS (
  SELECT
    trim(group_name) AS "group",
    trim(model_name) AS model,
    id AS channel_id,
    status = 1 AS enabled,
    priority,
    weight,
    tag
  FROM preset_channels,
    unnest(string_to_array("group", ',')) AS group_name,
    unnest(string_to_array(models, ',')) AS model_name
)
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, tag)
SELECT "group", model, channel_id, enabled, priority, weight, tag FROM expanded
ON CONFLICT ("group", model, channel_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  priority = EXCLUDED.priority,
  weight = EXCLUDED.weight,
  tag = EXCLUDED.tag;

COMMIT;

SELECT name, "group", models, base_url
FROM channels
WHERE tag = 'sub2api-video-preset'
ORDER BY id;
SQL

if [[ -n "$container" ]]; then
  docker exec -i "$container" psql "$dsn" \
    --set=ON_ERROR_STOP=1 \
    --set=base_url="$base_url" \
    --set=api_key="$api_key" <"$tmp_sql"
else
  psql "$dsn" \
    --set=ON_ERROR_STOP=1 \
    --set=base_url="$base_url" \
    --set=api_key="$api_key" \
    --file "$tmp_sql"
fi

echo "Sub2API video preset imported. Restart New API if channel cache is enabled."
