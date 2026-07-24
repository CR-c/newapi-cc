package wxart

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWxartImagine15AcceptsDocumentedRanges(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Prompt:      "slight head turn",
		Duration:    15,
		AspectRatio: "2:3",
		Resolution:  "480p",
		Images:      []string{"https://example.com/start.png"},
	}
	require.NoError(t, validateWxartVideoRequest(&req, ModelImagine15, ModelImagine15))
	assert.Equal(t, 15, req.Duration)
	assert.Equal(t, "2:3", req.AspectRatio)
	assert.Equal(t, "480p", req.Resolution)
}

func TestValidateWxartImagine15RejectsInvalid(t *testing.T) {
	tests := []struct {
		name string
		req  relaycommon.TaskSubmitReq
		want string
	}{
		{name: "missing image", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 6}, want: "exactly one"},
		{name: "two images", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 6, Images: []string{"https://example.com/1", "https://example.com/2"}}, want: "exactly one"},
		{name: "duration zero uses default then ok", req: relaycommon.TaskSubmitReq{Prompt: "x", Images: []string{"https://example.com/1"}}, want: ""},
		{name: "duration 16", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 16, Images: []string{"https://example.com/1"}}, want: "between 1 and 15"},
		{name: "bad ratio", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 6, AspectRatio: "21:9", Images: []string{"https://example.com/1"}}, want: "unsupported aspect_ratio"},
		{name: "1080p", req: relaycommon.TaskSubmitReq{Prompt: "x", Duration: 6, Resolution: "1080p", Images: []string{"https://example.com/1"}}, want: "480p or 720p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWxartVideoRequest(&tt.req, ModelImagine15, ModelImagine15)
			if tt.want == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidateWxartVideo3Modes(t *testing.T) {
	tests := []struct {
		name     string
		req      relaycommon.TaskSubmitReq
		wantMode string
		wantErr  string
	}{
		{
			name:     "text mode",
			req:      relaycommon.TaskSubmitReq{Prompt: "a cat", Mode: "text", Duration: 6, Resolution: "480p"},
			wantMode: "text",
		},
		{
			name:     "frame mode",
			req:      relaycommon.TaskSubmitReq{Prompt: "motion", Mode: "frame", Duration: 10, Images: []string{"https://example.com/1"}, Resolution: "720p"},
			wantMode: "frame",
		},
		{
			name:     "ref mode two images",
			req:      relaycommon.TaskSubmitReq{Prompt: "style", Mode: "ref", Duration: 12, Images: []string{"https://example.com/1", "https://example.com/2"}, AspectRatio: "1:1"},
			wantMode: "ref",
		},
		{
			name:     "infer text from zero images",
			req:      relaycommon.TaskSubmitReq{Prompt: "a cat", Duration: 16},
			wantMode: "text",
		},
		{
			name:     "infer frame from one image",
			req:      relaycommon.TaskSubmitReq{Prompt: "motion", Duration: 20, Images: []string{"https://example.com/1"}},
			wantMode: "frame",
		},
		{
			name:     "infer ref from multi images",
			req:      relaycommon.TaskSubmitReq{Prompt: "style", Duration: 6, Images: []string{"https://example.com/1", "https://example.com/2", "https://example.com/3"}},
			wantMode: "ref",
		},
		{
			name:    "text with image",
			req:     relaycommon.TaskSubmitReq{Prompt: "x", Mode: "text", Duration: 6, Images: []string{"https://example.com/1"}},
			wantErr: "does not accept images",
		},
		{
			name:    "frame needs one",
			req:     relaycommon.TaskSubmitReq{Prompt: "x", Mode: "frame", Duration: 6},
			wantErr: "exactly 1",
		},
		{
			name:    "ref max 7",
			req:     relaycommon.TaskSubmitReq{Prompt: "x", Mode: "ref", Duration: 6, Images: repeatedURLs(8)},
			wantErr: "1 to 7",
		},
		{
			name:    "bad duration",
			req:     relaycommon.TaskSubmitReq{Prompt: "x", Mode: "text", Duration: 8},
			wantErr: "6, 10, 12, 16, 20",
		},
		{
			name:    "bad ratio",
			req:     relaycommon.TaskSubmitReq{Prompt: "x", Mode: "text", Duration: 6, AspectRatio: "4:3"},
			wantErr: "unsupported aspect_ratio",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWxartVideoRequest(&tt.req, ModelVideo3, ModelVideo3)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, tt.req.Mode)
		})
	}
}

func TestBuildRequestBodyImagine15UsesSizeAndImagesURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:      "slight motion",
		Duration:    6,
		AspectRatio: "9:16",
		Resolution:  "480p",
		Images:      []string{"https://example.com/start.png"},
	})

	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelImagine15}}
	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, ModelImagine15, payload["model"])
	assert.Equal(t, "slight motion", payload["prompt"])
	assert.Equal(t, float64(6), payload["duration"])
	assert.Equal(t, "9:16", payload["aspect_ratio"])
	assert.Equal(t, "480p", payload["size"])
	_, hasResolution := payload["resolution"]
	assert.False(t, hasResolution)
	_, hasMode := payload["mode"]
	assert.False(t, hasMode)
	images, ok := payload["images_url"].([]any)
	require.True(t, ok)
	require.Len(t, images, 1)
	assert.Equal(t, "https://example.com/start.png", images[0])
}

func TestBuildRequestBodyVideo3UsesResolutionAndMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt:      "a cat walking",
		Mode:        "text",
		Duration:    10,
		AspectRatio: "16:9",
		Resolution:  "720p",
	})

	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelVideo3}}
	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, ModelVideo3, payload["model"])
	assert.Equal(t, "text", payload["mode"])
	assert.Equal(t, "720p", payload["resolution"])
	_, hasSize := payload["size"]
	assert.False(t, hasSize)
	_, hasImages := payload["images_url"]
	assert.False(t, hasImages)
}

func TestDoResponseHidesUpstreamTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"id":"task_upstream_secret",
			"status":"queued",
			"created_at":1784678400,
			"object":"video"
		}`)),
	}
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		OriginModelName: ModelVideo3,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}
	upstreamID, data, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "task_upstream_secret", upstreamID)
	assert.NotContains(t, string(data), "task_upstream_secret")

	var client dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &client))
	assert.Equal(t, "task_public", client.ID)
	assert.Equal(t, ModelVideo3, client.Model)
}

func TestParseTaskResultMapsCompletedVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	result, err := a.ParseTaskResult([]byte(`{
		"id":"task_up",
		"status":"completed",
		"url":"https://cdn.wxart.space/video-delivery/task_up"
	}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "https://cdn.wxart.space/video-delivery/task_up", result.Url)
}

func TestParseTaskResultMapsFailure(t *testing.T) {
	a := &TaskAdaptor{}
	result, err := a.ParseTaskResult([]byte(`{
		"id":"task_up",
		"status":"failed",
		"error":{"message":"moderation blocked"}
	}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, "moderation blocked", result.Reason)
}

func TestConvertToOpenAIVideoUsesProxyURL(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID: "task_public",
		Status: model.TaskStatusSuccess,
	}
	task.PrivateData.ResultURL = "https://cdn.wxart.space/video-delivery/secret"
	data, err := a.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	assert.Contains(t, string(data), "/v1/videos/task_public/content")
	assert.NotContains(t, string(data), "cdn.wxart.space")
}

func TestValidateSecondsAlias(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Prompt:      "x",
		Seconds:     "8",
		Images:      []string{"https://example.com/a.png"},
		AspectRatio: "16:9",
		Resolution:  "720p",
	}
	seconds, err := strconv.Atoi(req.Seconds)
	require.NoError(t, err)
	req.Duration = seconds
	require.NoError(t, validateWxartVideoRequest(&req, ModelImagine15, ModelImagine15))
	assert.Equal(t, 8, req.Duration)
}

func repeatedURLs(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = "https://example.com/img" + strconv.Itoa(i) + ".png"
	}
	return out
}
