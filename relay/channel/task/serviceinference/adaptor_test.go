package serviceinference

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBodyUploadsImagesAndPreservesOptionalParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assetRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/sd/assets" {
			http.NotFound(writer, request)
			return
		}
		assetRequests++
		assert.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-created","base_resp":{"status_code":0,"status_msg":"success"}}}`))
	}))
	defer server.Close()

	generateAudio := false
	watermark := false
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		AspectRatio: "1:1", Resolution: "480p",
		Images:        []string{"https://example.com/reference.png"},
		GenerateAudio: &generateAudio, Watermark: &watermark,
	})
	adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key"}
	body, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
	})
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload requestPayload
	require.NoError(t, common.Unmarshal(data, &payload))

	assert.Equal(t, 1, assetRequests)
	assert.Equal(t, "480p", payload.Resolution)
	assert.Equal(t, "1:1", payload.Ratio)
	require.NotNil(t, payload.GenerateAudio)
	assert.False(t, *payload.GenerateAudio)
	require.NotNil(t, payload.Watermark)
	assert.False(t, *payload.Watermark)
	require.Len(t, payload.Content, 2)
	require.NotNil(t, payload.Content[1].ImageURL)
	assert.Equal(t, "asset://asset-created", payload.Content[1].ImageURL.URL)
	assert.Equal(t, "reference_image", payload.Content[1].Role)
}

func TestBillingRatio(t *testing.T) {
	tests := []struct {
		model      string
		resolution string
		hasRef     bool
		want       float64
	}{
		{"dreamina-seedance-2-0-fast-hc", "480p", false, 1},
		{"dreamina-seedance-2-0-fast-hc", "720p", true, 3.3 / 5.6},
		{"dreamina-seedance-2-0-hc", "4k", false, 4.0 / 7.0},
		{"dreamina-seedance-2-0-hc", "4k", true, 2.4 / 7.0},
		{"dreamina-seedance-2-0-hc", "1080p", true, 4.7 / 7.0},
		{"dreamina-seedance-2-0-mini-hc", "720p", true, 2.1 / 3.5},
	}

	for _, tt := range tests {
		t.Run(tt.model+tt.resolution, func(t *testing.T) {
			ratio, ok := billingRatio(tt.model, tt.resolution, tt.hasRef)
			require.True(t, ok)
			assert.InDelta(t, tt.want, ratio, 0.0000001)
		})
	}
}

func TestEstimateBillingUsesMappedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Resolution: "720p",
		Images:     []string{"https://example.com/reference.png"},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "public-video-alias",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "dreamina-seedance-2-0-mini-hc",
		},
	}

	ratios := (&TaskAdaptor{}).EstimateBilling(context, info)
	assert.InDelta(t, 2.1/3.5, ratios["reference_image"], 0.0000001)
}

func TestTaskResolutionMapsPlaygroundVideoSizes(t *testing.T) {
	tests := map[string]string{
		"720x1280":  "720p",
		"1280x720":  "720p",
		"1024x1792": "1080p",
		"1792x1024": "1080p",
	}
	for size, want := range tests {
		assert.Equal(t, want, taskResolution(relaycommon.TaskSubmitReq{Size: size}))
	}
	assert.Equal(t, "4k", taskResolution(relaycommon.TaskSubmitReq{
		Size:     "720x1280",
		Metadata: map[string]any{"resolution": "4k"},
	}))
}

func TestParseTaskResultMapsCompletedUsage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(`{"task":{"id":"mvt-1","status":"completed","outputs":["https://cdn.example/video.mp4"],"usage":{"completion_tokens":40594,"total_tokens":40594}}}`))

	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "100%", result.Progress)
	assert.Equal(t, "https://cdn.example/video.mp4", result.Url)
	assert.Equal(t, 40594, result.TotalTokens)
}

func TestValidateRequestEnforcesServiceInferenceDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		duration   int
		wantError  bool
		wantStatus int
	}{
		{name: "below minimum", duration: 3, wantError: true, wantStatus: http.StatusBadRequest},
		{name: "minimum", duration: 4},
		{name: "maximum", duration: 15},
		{name: "above maximum", duration: 16, wantError: true, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(
				http.MethodPost,
				"/v1/videos",
				strings.NewReader(`{"model":"dreamina-seedance-2-0-mini-hc","prompt":"test","duration":`+fmt.Sprint(tt.duration)+`}`),
			)
			context.Request.Header.Set("Content-Type", "application/json")

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, &relaycommon.RelayInfo{
				TaskRelayInfo: &relaycommon.TaskRelayInfo{},
			})
			if tt.wantError {
				require.NotNil(t, taskErr)
				assert.Equal(t, tt.wantStatus, taskErr.StatusCode)
				assert.Contains(t, taskErr.Message, "between 4 and 15")
				return
			}
			require.Nil(t, taskErr)
		})
	}
}

func TestValidateRequestRejectsInvalidSecondsAndUnsupportedAssetReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{
		`{"model":"dreamina-seedance-2-0-mini-hc","prompt":"test","seconds":"invalid"}`,
		`{"model":"dreamina-seedance-2-0-mini-hc","prompt":"test","seconds":"4","images":["asset://foreign-asset"]}`,
		`{"model":"dreamina-seedance-2-0-mini-hc","prompt":"test","seconds":"4","videos":["https://example.com/ref.mp4"]}`,
	} {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(body))
		context.Request.Header.Set("Content-Type", "application/json")

		taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, &relaycommon.RelayInfo{
			TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		})
		require.NotNil(t, taskErr, body)
		assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	}
}
