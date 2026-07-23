package doubao

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToRequestPayloadMapsUnifiedVideoFields(t *testing.T) {
	generateAudio := true
	watermark := false
	adaptor := &TaskAdaptor{}

	payload, err := adaptor.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:         "doubao-seedance-2-0-260128",
		Prompt:        "animate the scene",
		Images:        []string{"https://example.com/reference.png"},
		Videos:        []string{"https://example.com/motion.mp4"},
		Audios:        []string{"https://example.com/music.mp3"},
		AspectRatio:   "16:9",
		Resolution:    "1080p",
		Seconds:       "8",
		GenerateAudio: &generateAudio,
		Watermark:     &watermark,
	})

	require.NoError(t, err)
	assert.Equal(t, "16:9", payload.Ratio)
	assert.Equal(t, "1080p", payload.Resolution)
	require.NotNil(t, payload.Duration)
	assert.Equal(t, dto.IntValue(8), *payload.Duration)
	require.NotNil(t, payload.GenerateAudio)
	assert.Equal(t, dto.BoolValue(true), *payload.GenerateAudio)
	require.NotNil(t, payload.Watermark)
	assert.Equal(t, dto.BoolValue(false), *payload.Watermark)
	require.Len(t, payload.Content, 4)
	assert.Equal(t, "image_url", payload.Content[0].Type)
	assert.Equal(t, "video_url", payload.Content[1].Type)
	assert.Equal(t, "audio_url", payload.Content[2].Type)
	assert.Equal(t, "text", payload.Content[3].Type)
}

func TestEstimateBillingRecordsVideoCostTier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Metadata: map[string]interface{}{
			"resolution": "1080p",
			"content": []interface{}{
				map[string]interface{}{"type": "video_url"},
			},
		},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-video-alias",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-seedance-2-0-260128",
		},
	}

	ratios := (&TaskAdaptor{}).EstimateBilling(context, info)

	assert.Equal(t, relaycommon.VideoCostTier1080pReference, info.CostTier)
	assert.NotEmpty(t, ratios)
}

func TestEstimateBillingUsesUnifiedVideoFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Resolution: "1080p",
		Videos:     []string{"https://example.com/reference.mp4"},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-2-0-260128",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-seedance-2-0-260128",
		},
	}

	ratios := (&TaskAdaptor{}).EstimateBilling(context, info)

	assert.Equal(t, relaycommon.VideoCostTier1080pReference, info.CostTier)
	assert.NotEmpty(t, ratios)
}

func TestBuildRequestHeaderAddsAuthenticationAndRequestIdentity(t *testing.T) {
	adaptor := &TaskAdaptor{apiKey: "upstream-secret"}
	request := httptest.NewRequest(http.MethodPost, "https://api.guyscode.com/tasks", nil)
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"}}

	require.NoError(t, adaptor.BuildRequestHeader(nil, request, info))

	assert.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
	assert.Equal(t, "task_public_123", request.Header.Get("Idempotency-Key"))
	assert.Equal(t, "task_public_123", request.Header.Get("X-Request-Id"))
}

func TestParseTaskResultAcceptsStringDurationAndKeepsResultPrivate(t *testing.T) {
	adaptor := &TaskAdaptor{ChannelType: constant.ChannelTypeDoubaoVideo}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"seedance_upstream_123",
		"status":"succeeded",
		"duration":"8",
		"content":{"video_url":"/api/v3/contents/generations/tasks/seedance_upstream_123/content"},
		"usage":{"completion_tokens":1234,"total_tokens":1234}
	}`))

	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusSuccess), result.Status)
	assert.Equal(t, 1234, result.CompletionTokens)
	assert.Equal(t, 1234, result.TotalTokens)
	assert.Empty(t, result.Url)
}

func TestParseTaskResultPreservesDirectURLForVolcEngine(t *testing.T) {
	adaptor := &TaskAdaptor{ChannelType: constant.ChannelTypeVolcEngine}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"volc_upstream_123",
		"status":"succeeded",
		"duration":8,
		"content":{"video_url":"https://cdn.example.com/result.mp4"}
	}`))

	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/result.mp4", result.Url)
}

func TestParseTaskResultTreatsExpiredAsTerminalFailure(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"seedance_upstream_123",
		"status":"expired",
		"duration":"5"
	}`))

	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusFailure), result.Status)
	assert.Equal(t, "video task expired", result.Reason)
}

func TestParseTaskResultHidesUpstreamFailureMessage(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"id":"seedance_upstream_123",
		"status":"failed",
		"error":{"code":"UPSTREAM_INTERNAL","message":"private provider stack detail"}
	}`))

	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusFailure), result.Status)
	assert.Equal(t, "Video generation failed", result.Reason)
}

func TestSanitizeTaskDataRemovesPrivateIdentifiersAndURLs(t *testing.T) {
	adaptor := &TaskAdaptor{}

	sanitized := adaptor.SanitizeTaskData([]byte(`{
		"id":"seedance_upstream_123",
		"status":"succeeded",
		"model":"doubao-seedance-2-0-260128",
		"content":{
			"video_url":"/api/v3/contents/generations/tasks/seedance_upstream_123/content",
			"last_frame_url":"/api/v3/contents/generations/tasks/seedance_upstream_123/last-frame"
		},
		"error":{"code":"INVALID_ASSET","message":"private provider detail"},
		"usage":{"total_tokens":1234}
	}`))

	assert.NotContains(t, string(sanitized), "seedance_upstream_123")
	assert.NotContains(t, string(sanitized), "video_url")
	assert.NotContains(t, string(sanitized), "last_frame_url")
	assert.NotContains(t, string(sanitized), "private provider detail")
	assert.Contains(t, string(sanitized), "VIDEO_GENERATION_FAILED")
	assert.Contains(t, string(sanitized), `"total_tokens":1234`)
}

func TestSanitizeTaskDataDropsUnknownFields(t *testing.T) {
	adaptor := &TaskAdaptor{}

	sanitized := adaptor.SanitizeTaskData([]byte(`{
		"status":"running",
		"debug_token":"private-token",
		"request_context":{"signed_url":"https://storage.example/private"},
		"usage":{
			"completion_tokens":12,
			"total_tokens":34,
			"debug_token":"nested-private-token",
			"tool_usage":{"web_search":1,"signed_url":"https://storage.example/private"}
		},
		"error":"private string error"
	}`))

	assert.JSONEq(t, `{
		"status":"running",
		"usage":{"completion_tokens":12,"total_tokens":34},
		"error":{"code":"VIDEO_GENERATION_FAILED","message":"Video generation failed"}
	}`, string(sanitized))
}

func TestSanitizeTaskDataRejectsInvalidAllowedFieldTypes(t *testing.T) {
	adaptor := &TaskAdaptor{}

	sanitized := adaptor.SanitizeTaskData([]byte(`{
		"status":{"value":"running","debug_token":"private-token"},
		"usage":{"total_tokens":34}
	}`))

	assert.JSONEq(t, `{}`, string(sanitized))
}

func TestFetchTaskEscapesUpstreamTaskID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t,
			"/api/v3/contents/generations/tasks/seedance%2Fid%20with%20spaces",
			request.URL.EscapedPath(),
		)
		assert.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"seedance","status":"running","duration":"5"}`))
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	response, err := adaptor.FetchTask(server.URL, "upstream-secret", map[string]any{
		"task_id": "seedance/id with spaces",
	}, "")

	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestFetchTaskConvertsPermanentUpstreamErrorToTaskFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"code":"INVALID_API_KEY","message":"private detail"}`))
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	response, err := adaptor.FetchTask(server.URL, "upstream-secret", map[string]any{
		"task_id": "seedance_upstream_123",
	}, "")

	require.NoError(t, err)
	require.NotNil(t, response)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "private detail")
	result, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusFailure), result.Status)
	assert.Equal(t, "Video generation failed", result.Reason)
}

func TestFetchTaskRetriesRateLimitStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	response, err := adaptor.FetchTask(server.URL, "upstream-secret", map[string]any{
		"task_id": "seedance_upstream_123",
	}, "")

	require.Error(t, err)
	assert.Nil(t, response)
}

func TestParseTaskResultRejectsUnknownStatus(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{"id":"seedance_upstream_123","status":"mystery"}`))

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestConvertToOpenAIVideoUsesPublicContentProxy(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID:   "task_public_123",
		Platform: constant.TaskPlatform("54"),
		Status:   model.TaskStatusSuccess,
		Data: []byte(`{
			"id":"seedance_upstream_123",
			"status":"succeeded",
			"duration":"8",
			"content":{"video_url":"/api/v3/contents/generations/tasks/seedance_upstream_123/content"}
		}`),
	}

	payload, err := adaptor.ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	assert.Contains(t, string(payload), "/v1/videos/task_public_123/content")
	assert.NotContains(t, string(payload), "seedance_upstream_123")
}

func TestConvertExpiredTaskIncludesPublicError(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID:   "task_public_123",
		Platform: constant.TaskPlatform("54"),
		Status:   model.TaskStatusFailure,
		Data:     []byte(`{"status":"expired","duration":"8"}`),
	}

	payload, err := adaptor.ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	assert.Contains(t, string(payload), `"code":"VIDEO_TASK_EXPIRED"`)
	assert.Contains(t, string(payload), `"message":"Video task expired"`)
	assert.NotContains(t, string(payload), "/content")
}

func TestConvertQueuedTaskDoesNotExposeContentURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		TaskID:   "task_public_123",
		Platform: constant.TaskPlatform("54"),
		Status:   model.TaskStatusQueued,
		Data:     []byte(`{"status":"queued","duration":"8"}`),
	}

	payload, err := adaptor.ConvertToOpenAIVideo(task)

	require.NoError(t, err)
	assert.NotContains(t, string(payload), "/content")
}

func TestRejectRedirectsOnlyForDoubaoVideo(t *testing.T) {
	assert.True(t, (&TaskAdaptor{ChannelType: constant.ChannelTypeDoubaoVideo}).RejectRedirects())
	assert.False(t, (&TaskAdaptor{ChannelType: constant.ChannelTypeVolcEngine}).RejectRedirects())
}
