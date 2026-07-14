package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskAdaptorSupportsLegacyOpenAIVideoAPI(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.119337.xyz",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				LegacyOpenAIVideoAPI: true,
			},
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	adaptor.Init(info)
	url, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.119337.xyz/v1/video/generations", url)

	result, err := adaptor.ParseTaskResult([]byte(`{
		"code":"success",
		"data":{
			"task_id":"task_119337",
			"status":"SUCCESS",
			"progress":"100%",
			"result_url":"https://video.example/result.mp4"
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", result.Status)
	assert.Equal(t, "100%", result.Progress)
	assert.Equal(t, "https://video.example/result.mp4", result.Url)
}

func TestTaskAdaptorLegacySubmitReturnsOpenAIVideoEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	response := &http.Response{Body: io.NopCloser(bytes.NewBufferString(`{
		"code":"success",
		"data":{"task_id":"upstream-task","status":"SUBMITTED","progress":"10%"}
	}`))}
	adaptor := &TaskAdaptor{legacyOpenAIVideoAPI: true}
	info := &relaycommon.RelayInfo{
		OriginModelName: "sora-2",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	upstreamID, _, taskErr := adaptor.DoResponse(context, response, info)

	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", upstreamID)
	var body responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "task_public", body.ID)
	assert.Equal(t, dto.VideoStatusQueued, body.Status)
	assert.Equal(t, 10, body.Progress)
}

func TestTaskAdaptorLegacySubmitAcceptsDirectOpenAIVideoEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	response := &http.Response{Body: io.NopCloser(bytes.NewBufferString(`{
		"id":"task_upstream","task_id":"task_upstream","object":"video",
		"model":"grok-image-video","status":"queued","progress":0,"created_at":1784020946
	}`))}
	adaptor := &TaskAdaptor{legacyOpenAIVideoAPI: true}
	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-image-video",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	upstreamID, _, taskErr := adaptor.DoResponse(context, response, info)

	require.Nil(t, taskErr)
	assert.Equal(t, "task_upstream", upstreamID)
	var body responseTask
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "task_public", body.ID)
	assert.Equal(t, "task_public", body.TaskID)
	assert.Equal(t, dto.VideoStatusQueued, body.Status)
	assert.Equal(t, int64(1784020946), body.CreatedAt)
}

func TestTaskAdaptorRemovesPlaygroundGroupFromJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/videos", bytes.NewBufferString(`{"model":"sora-2","group":"vip","prompt":"boat"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "sora-upstream"}}

	body, err := (&TaskAdaptor{}).BuildRequestBody(context, info)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(bodyBytes, &payload))
	assert.Equal(t, "sora-upstream", payload["model"])
	assert.NotContains(t, payload, "group")
}

func TestLegacyGrokAdaptorConvertsUnifiedVideoParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/videos", bytes.NewBufferString(`{
		"model":"grok-image-video",
		"group":"vip",
		"prompt":"animate product",
		"seconds":"10",
		"aspect_ratio":"9:16",
		"resolution":"720p",
		"images":["https://example.com/one.png","https://example.com/two.png"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "grok-image-video",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName:    "grok-image-video",
			ChannelOtherSettings: dto.ChannelOtherSettings{LegacyOpenAIVideoAPI: true},
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))

	body, err := adaptor.BuildRequestBody(context, info)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(bodyBytes, &payload))
	assert.Equal(t, float64(10), payload["seconds"])
	assert.Equal(t, "9:16", payload["aspect_ratio"])
	assert.Equal(t, "720p", payload["resolution"])
	assert.Equal(t, []any{"https://example.com/one.png", "https://example.com/two.png"}, payload["image_urls"])
	assert.NotContains(t, payload, "images")
	assert.NotContains(t, payload, "group")
}

func TestGrokVideoValidationEnforcesReferenceRules(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		request relaycommon.TaskSubmitReq
		wantErr string
	}{
		{name: "preview model requires image", model: "grok-video-1.5", request: relaycommon.TaskSubmitReq{Seconds: "4", AspectRatio: "16:9", Resolution: "480p"}, wantErr: "exactly one"},
		{name: "preview model accepts one image", model: "grok-video-1.5", request: relaycommon.TaskSubmitReq{Seconds: "15", AspectRatio: "9:16", Resolution: "720p", Images: []string{"https://example.com/one.png"}}},
		{name: "multi reference caps duration", model: "grok-image-video", request: relaycommon.TaskSubmitReq{Seconds: "12", AspectRatio: "16:9", Resolution: "720p", Images: []string{"https://example.com/one.png", "https://example.com/two.png"}}},
		{name: "rejects eighth image", model: "grok-image-video", request: relaycommon.TaskSubmitReq{Seconds: "10", AspectRatio: "16:9", Resolution: "720p", Images: []string{"https://example.com/1.png", "https://example.com/2.png", "https://example.com/3.png", "https://example.com/4.png", "https://example.com/5.png", "https://example.com/6.png", "https://example.com/7.png", "https://example.com/8.png"}}, wantErr: "at most 7"},
		{name: "rejects invalid seconds", model: "grok-image-video", request: relaycommon.TaskSubmitReq{Seconds: "invalid", AspectRatio: "16:9", Resolution: "720p"}, wantErr: "seconds must be"},
		{name: "rejects reference video", model: "grok-image-video", request: relaycommon.TaskSubmitReq{Seconds: "4", AspectRatio: "16:9", Resolution: "720p", Videos: []string{"https://example.com/ref.mp4"}}, wantErr: "reference images"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateGrokVideoRequest(&test.request, test.model)
			if test.wantErr == "" {
				require.NoError(t, err)
				if test.name == "multi reference caps duration" {
					assert.Equal(t, "10", test.request.Seconds)
				}
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestSub2APIVideoValidationEnforcesMediaLimits(t *testing.T) {
	assert.NoError(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{
		Seconds: "15", AspectRatio: "9:16",
		Images: []string{"https://example.com/1.png", "https://example.com/2.png", "https://example.com/3.png", "https://example.com/4.png"},
		Videos: []string{"https://example.com/1.mp4", "https://example.com/2.mp4", "https://example.com/3.mp4"},
		Audios: []string{"https://example.com/1.mp3"},
	}, "video-ds-2.0"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{Images: make([]string, 5)}, "video-ds-2.0"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{Videos: make([]string, 4)}, "video-ds-2.0-fast"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{Audios: make([]string, 2)}, "as-sd2.0-fast"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{Seconds: "4"}, "video-ds-2.0"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{Seconds: "invalid"}, "video-ds-2.0"))
	assert.Error(t, validateSub2APIVideoRequest(relaycommon.TaskSubmitReq{AspectRatio: "4:3"}, "video-ds-2.0"))
}

func TestSub2APIBuildRequestNormalizesAndWhitelistsFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/videos", bytes.NewBufferString(`{
		"model":"video-ds-2.0","group":"vip","prompt":"animate","seconds":15,
		"aspect_ratio":"9:16","images":["https://example.com/ref.png"],"unknown_cost_field":999
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "video-ds-2.0", Prompt: "animate", Seconds: "15", AspectRatio: "9:16",
		Images: []string{"https://example.com/ref.png"},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "video-ds-2.0",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "video-ds-2.0"},
	}

	body, err := (&TaskAdaptor{}).BuildRequestBody(context, info)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(bodyBytes, &payload))
	assert.Equal(t, "15", payload["seconds"])
	assert.NotContains(t, payload, "group")
	assert.NotContains(t, payload, "unknown_cost_field")
}

func TestTaskAdaptorConvertsStoredLegacyTaskToOpenAIVideo(t *testing.T) {
	task := &model.Task{TaskID: "task_public", Status: model.TaskStatusSuccess, Progress: "100%"}
	task.Properties.OriginModelName = "sora-2"

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var video dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(body, &video))
	assert.Equal(t, "task_public", video.ID)
	assert.Equal(t, dto.VideoStatusCompleted, video.Status)
}

func TestTaskAdaptorPreservesLegacyTaskFailure(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusFailure,
		Progress:   "100%",
		FailReason: "reference image is unavailable",
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var video dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(body, &video))
	require.NotNil(t, video.Error)
	assert.Equal(t, "reference image is unavailable", video.Error.Message)
}

func TestTaskAdaptorPreservesStandardOpenAIVideoFields(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Data:   []byte(`{"id":"upstream","status":"in_progress","seconds":"8","size":"1280x720","expires_at":123}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var video map[string]any
	require.NoError(t, common.Unmarshal(body, &video))
	assert.Equal(t, "task_public", video["id"])
	assert.Equal(t, "8", video["seconds"])
	assert.Equal(t, "1280x720", video["size"])
	assert.Equal(t, float64(123), video["expires_at"])
}

func TestTaskAdaptorReportsLegacyOpenAIVideoFailure(t *testing.T) {
	adaptor := &TaskAdaptor{legacyOpenAIVideoAPI: true}

	result, err := adaptor.ParseTaskResult([]byte(`{
		"code":"success",
		"data":{
			"task_id":"task_119337",
			"status":"FAILURE",
			"fail_reason":"reference image is unavailable"
		}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "FAILURE", result.Status)
	assert.Equal(t, "reference image is unavailable", result.Reason)
}

func TestLegacyGrokValidationUsesMappedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/videos", bytes.NewBufferString(`{
		"model":"public-video-alias","prompt":"animate","seconds":"4"
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-video-alias",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName:    "grok-video-1.5",
			ChannelOtherSettings: dto.ChannelOtherSettings{LegacyOpenAIVideoAPI: true},
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)

	taskErr := adaptor.ValidateRequestAndSetAction(context, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "exactly one reference image")
}
