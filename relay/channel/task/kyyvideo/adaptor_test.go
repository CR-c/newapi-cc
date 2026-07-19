package kyyvideo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateKyyVideoRequestAcceptsDocumentedModelCapabilities(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		req    relaycommon.TaskSubmitReq
		ratio  string
		images int
		videos int
		audios int
	}{
		{
			name:  "videos supports nine images three videos and three audios",
			model: "videos",
			req:   relaycommon.TaskSubmitReq{Duration: 4, AspectRatio: "21:9"},
			ratio: "21:9", images: 9, videos: 3, audios: 3,
		},
		{
			name:  "stable supports four images three videos and one audio",
			model: "videos_stable",
			req:   relaycommon.TaskSubmitReq{Duration: 15, AspectRatio: "1:1"},
			ratio: "1:1", images: 4, videos: 3, audios: 1,
		},
		{
			name:  "stable fast supports ten seconds",
			model: "videos_stable_fast",
			req:   relaycommon.TaskSubmitReq{Duration: 10},
			ratio: "16:9", images: 4, videos: 3, audios: 1,
		},
		{
			name:  "pro supports image and audio references",
			model: "videos_pro",
			req:   relaycommon.TaskSubmitReq{Duration: 15},
			ratio: "16:9", images: 9, audios: 3,
		},
		{
			name:  "pro fast supports image and audio references",
			model: "videos_pro_fast",
			req:   relaycommon.TaskSubmitReq{Duration: 10, AspectRatio: "9:16"},
			ratio: "9:16", images: 9, audios: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			req.Prompt = "animate the scene"
			req.Images = repeatedURLs("image", tt.images)
			req.Videos = repeatedURLs("video", tt.videos)
			req.Audios = repeatedURLs("audio", tt.audios)

			require.NoError(t, validateKyyVideoRequest(&req, tt.model))
			assert.Equal(t, tt.ratio, req.AspectRatio)
			assert.Equal(t, "720p", req.Resolution)
		})
	}
}

func TestValidateKyyVideoRequestRejectsDocumentedInvalidCombinations(t *testing.T) {
	tests := []struct {
		name  string
		model string
		req   relaycommon.TaskSubmitReq
		want  string
	}{
		{name: "unknown model", model: "videos_unknown", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10}, want: "unsupported model"},
		{name: "missing duration", model: "videos", req: relaycommon.TaskSubmitReq{Prompt: "x"}, want: "duration is required"},
		{name: "stable fast duration", model: "videos_stable_fast", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 11}, want: "10 or 15"},
		{name: "pro video references", model: "videos_pro", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, Videos: repeatedURLs("video", 1)}, want: "does not support reference videos"},
		{name: "pro audio only", model: "videos_pro_fast", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, Audios: repeatedURLs("audio", 1)}, want: "audio references require at least one reference image"},
		{name: "too many stable images", model: "videos_stable", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, Images: repeatedURLs("image", 5)}, want: "at most 4 reference images"},
		{name: "unsupported ratio", model: "videos", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, AspectRatio: "3:2"}, want: "unsupported aspect_ratio"},
		{name: "unsupported resolution", model: "videos", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, Resolution: "1080p"}, want: "resolution must be 720p"},
		{name: "frame pair required", model: "videos", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, FirstImage: "https://example.com/first.png"}, want: "must be provided together"},
		{name: "frame and references conflict", model: "videos", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 10, FirstImage: "https://example.com/first.png", LastImage: "https://example.com/last.png", Images: repeatedURLs("image", 1)}, want: "cannot be combined"},
		{name: "stable prompt too long", model: "videos_stable", req: relaycommon.TaskSubmitReq{Prompt: strings.Repeat("x", 5001), Duration: 10}, want: "at most 5000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKyyVideoRequest(&tt.req, tt.model)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestBuildRequestMapsUnifiedFieldsToKyyVideoContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	autoFace := false
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:      "animate",
		Duration:    10,
		AspectRatio: "9:16",
		Resolution:  "720p",
		Images:      []string{"https://example.com/image.png"},
		Videos:      []string{"https://example.com/video.mp4"},
		Audios:      []string{"https://example.com/audio.mp3"},
		AutoFace:    &autoFace,
	})
	adaptor := &TaskAdaptor{baseURL: "https://zcbservice.aizfw.cn/kyyReactApiServer", apiKey: "test-key"}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "videos"}}

	requestURL, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://zcbservice.aizfw.cn/kyyReactApiServer/v1/videos/videos", requestURL)

	body, err := adaptor.BuildRequestBody(context, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload requestPayload
	require.NoError(t, common.Unmarshal(data, &payload))
	assert.Equal(t, "videos", payload.Model)
	assert.Equal(t, "animate", payload.Prompt)
	assert.Equal(t, 10, payload.Duration)
	assert.Equal(t, "9:16", payload.Ratio)
	assert.Equal(t, "720p", payload.Resolution)
	assert.Equal(t, []string{"https://example.com/image.png"}, payload.ReferenceImages)
	assert.Equal(t, []string{"https://example.com/video.mp4"}, payload.ReferenceVideos)
	assert.Equal(t, []string{"https://example.com/audio.mp3"}, payload.ReferenceAudios)
	require.NotNil(t, payload.AutoFace)
	assert.False(t, *payload.AutoFace)
}

func TestValidateRequestAndSetActionNormalizesDefaultsAndSeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{
		"model":"videos_stable_fast",
		"prompt":"animate",
		"seconds":"10"
	}`))
	context.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "videos_stable_fast"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info))
	assert.Equal(t, constant.TaskActionGenerate, info.Action)
	req, err := relaycommon.GetTaskRequest(context)
	require.NoError(t, err)
	assert.Equal(t, 10, req.Duration)
	assert.Empty(t, req.Seconds)
	assert.Equal(t, "16:9", req.AspectRatio)
	assert.Equal(t, "720p", req.Resolution)
}

func TestBuildRequestHeaderSetsKyyAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	adaptor := &TaskAdaptor{apiKey: "test-key"}
	require.NoError(t, adaptor.BuildRequestHeader(nil, request, nil))
	assert.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
	assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
	assert.Equal(t, "application/json", request.Header.Get("Accept"))
}

func TestFetchTaskUsesKyyResultEndpointAndBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/result/upstream-video-id", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"upstream-video-id","status":"processing"}`))
	}))
	defer server.Close()

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "test-key", map[string]any{"task_id": "upstream-video-id"}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoResponseHidesUpstreamTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"id":"upstream-video-id",
			"object":"video",
			"created":1761635478,
			"model":"videos",
			"status":"queued",
			"error":null
		}`)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "videos",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(context, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-video-id", upstreamID)
	assert.NotContains(t, string(taskData), "upstream-video-id")
	assert.NotContains(t, recorder.Body.String(), "upstream-video-id")
	assert.Contains(t, recorder.Body.String(), `"id":"task_public"`)
	assert.Contains(t, recorder.Body.String(), `"task_id":"task_public"`)
}

func TestParseTaskResultMapsCompletedVideoURL(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"upstream-video-id",
		"object":"video",
		"created":1761635478,
		"model":"videos_stable_fast",
		"status":"completed",
		"video_url":"https://example.com/result.mp4",
		"actualDuration":"10",
		"amount":"0.32",
		"error":null
	}`))

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "100%", result.Progress)
	assert.Equal(t, "https://example.com/result.mp4", result.Url)
}

func TestParseTaskResultMapsFailureReason(t *testing.T) {
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"upstream-video-id",
		"status":"failed",
		"error":"reference video is invalid"
	}`))

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, "100%", result.Progress)
	assert.Equal(t, "KYY video generation failed", result.Reason)
}

func TestParseTaskResultRejectsMissingStatusAndTaskID(t *testing.T) {
	_, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"upstream-video-id"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")

	_, err = (&TaskAdaptor{}).ParseTaskResult([]byte(`{"status":"processing"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task id is empty")
}

func TestConvertToOpenAIVideoUsesPublicTaskAndStoredResultURL(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  1761635478,
		UpdatedAt:  1761635578,
		Properties: model.Properties{OriginModelName: "videos"},
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream-video-id",
			ResultURL:      "https://example.com/result.mp4",
		},
	}

	data, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var response dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(data, &response))
	assert.Equal(t, "task_public", response.ID)
	assert.Equal(t, dto.VideoStatusCompleted, response.Status)
	assert.NotEqual(t, "https://example.com/result.mp4", response.Metadata["url"])
	assert.Contains(t, response.Metadata["url"], "/v1/videos/task_public/content")
}

func repeatedURLs(kind string, count int) []string {
	urls := make([]string, count)
	for i := range urls {
		urls[i] = "https://example.com/" + kind + "-" + string(rune('a'+i))
	}
	return urls
}
