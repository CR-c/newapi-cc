package serviceinference

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBodyUploadsImagesAndPreservesOptionalParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assetRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet && request.URL.Path == "/v1/sd/assets/asset-created" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-created","Status":"Active","base_resp":{"status_code":0,"status_msg":"success"}}}`))
			return
		}
		if request.Method != http.MethodPost || request.URL.Path != "/v1/sd/assets" {
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
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetPollInterval: time.Millisecond,
	}
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

func TestBuildRequestBodyWaitsForReferenceAssetToBecomeActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requests := make([]string, 0, 3)
	statusRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/sd/assets":
			requests = append(requests, "create")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-created","base_resp":{"status_code":0,"status_msg":"success"}}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/sd/assets/asset-created":
			requests = append(requests, "status")
			statusRequests++
			status := "Processing"
			if statusRequests > 1 {
				status = "Active"
			}
			_, _ = fmt.Fprintf(writer, `{"success":true,"data":{"Id":"asset-created","Status":%q,"base_resp":{"status_code":0,"status_msg":"success"}}}`, status)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/reference.png"},
	})
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetPollInterval: time.Millisecond,
	}

	body, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"create", "status", "status"}, requests)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"url":"asset://asset-created"`)
}

func TestBuildRequestBodyPreparesReferenceImagesConcurrentlyAndPreservesOrder(t *testing.T) {
	var mutex sync.Mutex
	created := 0
	allCreated := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodPost {
			var payload map[string]string
			if err := common.DecodeJson(request.Body, &payload); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			index, err := strconv.Atoi(strings.TrimPrefix(payload["URL"], "https://example.com/"))
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			mutex.Lock()
			created++
			if created == 3 {
				close(allCreated)
			}
			mutex.Unlock()
			_, _ = fmt.Fprintf(writer, `{"success":true,"data":{"Id":"asset-%d","base_resp":{"status_code":0,"status_msg":"success"}}}`, index)
			return
		}
		select {
		case <-allCreated:
			assetID := strings.TrimPrefix(request.URL.Path, "/v1/sd/assets/")
			_, _ = fmt.Fprintf(writer, `{"success":true,"data":{"Id":%q,"Status":"Active","base_resp":{"status_code":0,"status_msg":"success"}}}`, assetID)
		case <-time.After(time.Second):
			http.Error(writer, "images were prepared serially", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/1", "https://example.com/2", "https://example.com/3"},
	})
	adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key", assetPollInterval: time.Millisecond}

	body, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
	})

	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload requestPayload
	require.NoError(t, common.Unmarshal(data, &payload))
	require.Len(t, payload.Content, 4)
	actualURLs := make([]string, 0, 3)
	for _, item := range payload.Content[1:] {
		require.NotNil(t, item.ImageURL)
		actualURLs = append(actualURLs, item.ImageURL.URL)
	}
	assert.Equal(t, []string{"asset://asset-1", "asset://asset-2", "asset://asset-3"}, actualURLs)
}

func TestBuildRequestBodyValidatesAllStoredReferencesBeforeStartingUploads(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/1", "asset://missing"},
	})
	adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key", assetPollInterval: time.Millisecond}

	_, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
	})

	require.EqualError(t, err, "video asset reference is not available")
	assert.Zero(t, requests)
}

func TestWaitForImageAssetRetriesEmbeddedServerError(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		polls++
		writer.Header().Set("Content-Type", "application/json")
		if polls == 1 {
			_, _ = writer.Write([]byte(`{"success":false,"data":{"base_resp":{"status_code":500,"status_msg":"temporary"}}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-created","Status":"Active","base_resp":{"status_code":0,"status_msg":"success"}}}`))
	}))
	defer server.Close()
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key", assetPollInterval: time.Millisecond, assetMaxPolls: 3,
	}

	err := adaptor.waitForImageAsset(t.Context(), server.Client(), "asset-created")

	require.NoError(t, err)
	assert.Equal(t, 2, polls)
}

func TestWaitForImageAssetStopsOnFailureAndTimeout(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		maxPolls  int
		wantError string
		wantPolls int
	}{
		{
			name: "terminal failure", status: "Failed", maxPolls: 3,
			wantError: "reference asset processing failed", wantPolls: 1,
		},
		{
			name: "processing timeout", status: "Processing", maxPolls: 2,
			wantError: "reference asset processing timed out", wantPolls: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			polls := 0
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				polls++
				writer.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(writer, `{"success":true,"data":{"Id":"asset-created","Status":%q,"base_resp":{"status_code":0,"status_msg":"success"}}}`, test.status)
			}))
			defer server.Close()

			adaptor := &TaskAdaptor{
				baseURL: server.URL, apiKey: "test-key",
				assetPollInterval: time.Millisecond,
				assetMaxPolls:     test.maxPolls,
			}

			err := adaptor.waitForImageAsset(t.Context(), server.Client(), "asset-created")

			require.EqualError(t, err, test.wantError)
			assert.Equal(t, test.wantPolls, polls)
		})
	}
}

func TestWaitForImageAssetStopsOnPermanentHTTPError(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		polls++
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetPollInterval: time.Millisecond,
		assetMaxPolls:     30,
	}

	err := adaptor.waitForImageAsset(t.Context(), server.Client(), "asset-created")

	require.EqualError(t, err, "asset readiness check failed")
	assert.Equal(t, 1, polls)
}

func TestWaitForImageAssetBoundsBlockedStatusRequest(t *testing.T) {
	requestCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
		close(requestCanceled)
	}))
	defer server.Close()
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetPollInterval: time.Millisecond,
		assetMaxPolls:     30,
		assetMaxWait:      50 * time.Millisecond,
	}

	err := adaptor.waitForImageAsset(t.Context(), server.Client(), "asset-created")

	require.EqualError(t, err, "reference asset processing timed out")
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("blocked status request was not canceled by the readiness deadline")
	}
}

func TestBuildRequestBodyBoundsBlockedAssetUpload(t *testing.T) {
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-releaseHandler
	}))
	defer server.Close()
	defer close(releaseHandler)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/reference.png"},
	})
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetMaxWait: 50 * time.Millisecond,
	}

	result := make(chan error, 1)
	go func() {
		_, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
		})
		result <- err
	}()
	select {
	case err := <-result:
		require.EqualError(t, err, "reference asset preparation timed out")
	case <-time.After(time.Second):
		t.Fatal("blocked asset upload exceeded the preparation deadline")
	}
}

func TestBuildRequestBodyBoundsBlockedAssetStatus(t *testing.T) {
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-created","base_resp":{"status_code":0,"status_msg":"success"}}}`))
			return
		}
		<-releaseHandler
	}))
	defer server.Close()
	defer close(releaseHandler)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/reference.png"},
	})
	adaptor := &TaskAdaptor{
		baseURL: server.URL, apiKey: "test-key",
		assetPollInterval: time.Millisecond,
		assetMaxWait:      50 * time.Millisecond,
	}

	result := make(chan error, 1)
	go func() {
		_, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
			ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
		})
		result <- err
	}()
	select {
	case err := <-result:
		require.EqualError(t, err, "reference asset processing timed out")
	case <-time.After(time.Second):
		t.Fatal("blocked asset status check exceeded the preparation deadline")
	}
}

func TestBuildRequestBodyHidesAssetUploadFailureDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"data":{"base_resp":{"status_code":500,"status_msg":"request-id secret-upstream-host"}}}`))
	}))
	defer server.Close()

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
		Images: []string{"https://example.com/reference.png"},
	})
	adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key"}

	_, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
	})

	require.EqualError(t, err, "asset upload failed")
	assert.NotContains(t, err.Error(), "request-id")
	assert.NotContains(t, err.Error(), "secret-upstream-host")
}

func TestBuildRequestBodyMapsSmallReferenceImageToClientError(t *testing.T) {
	messages := []string{
		"CreateAsset failed: code=InvalidParameter.WidthTooSmall request_id=secret-request-id",
		"CreateAsset failed: InvalidParameter.WidthTooSmall: Width must be between 300px and 6000px",
	}

	for _, message := range messages {
		t.Run(message, func(t *testing.T) {
			data, err := common.Marshal(map[string]any{"success": false, "message": message})
			require.NoError(t, err)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write(data)
			}))
			defer server.Close()

			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Set("task_request", relaycommon.TaskSubmitReq{
				Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
				Images: []string{"https://example.com/reference.png"},
			})
			adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key"}

			_, err = adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
			})

			var requestErr *service.TaskRequestError
			require.ErrorAs(t, err, &requestErr)
			assert.Equal(t, "reference image width must be between 300px and 6000px", requestErr.Error())
			taskErr := service.TaskErrorFromBuildRequest(err)
			assert.Equal(t, "invalid_reference_image", taskErr.Code)
			assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			assert.True(t, taskErr.LocalError)
			assert.NotContains(t, err.Error(), "secret-request-id")
		})
	}
}

func TestBuildRequestBodyDoesNotTrustUnclassifiedAssetErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "unknown invalid parameter",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"message":"code=InvalidParameter.Unknown request_id=secret-request-id"}`,
		},
		{
			name:       "similar error code",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"message":"code=InvalidParameter.WidthTooSmallEnough request_id=secret-request-id"}`,
		},
		{
			name:       "malformed response",
			statusCode: http.StatusBadRequest,
			body:       `code=InvalidParameter.WidthTooSmall request_id=secret-request-id`,
		},
		{
			name:       "credential failure with misleading body",
			statusCode: http.StatusUnauthorized,
			body:       `{"success":false,"message":"code=InvalidParameter.WidthTooSmall request_id=secret-request-id"}`,
		},
		{
			name:       "server failure with misleading body",
			statusCode: http.StatusInternalServerError,
			body:       `{"success":false,"message":"code=InvalidParameter.WidthTooSmall request_id=secret-request-id"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.statusCode)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Set("task_request", relaycommon.TaskSubmitReq{
				Model: "dreamina-seedance-2-0-hc", Prompt: "animate", Duration: 5,
				Images: []string{"https://example.com/reference.png"},
			})
			adaptor := &TaskAdaptor{baseURL: server.URL, apiKey: "test-key"}

			_, err := adaptor.BuildRequestBody(context, &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dreamina-seedance-2-0-hc"},
			})

			var requestErr *service.TaskRequestError
			require.NotErrorAs(t, err, &requestErr)
			assert.EqualError(t, err, "asset upload failed")
			assert.NotContains(t, err.Error(), "secret-request-id")
			taskErr := service.TaskErrorFromBuildRequest(err)
			assert.Equal(t, test.statusCode, taskErr.StatusCode)
			assert.False(t, taskErr.LocalError)
		})
	}
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
	assert.Equal(t, relaycommon.VideoCostTier480p720pReference, info.CostTier)
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

func TestValidateRequestEnforcesResolutionByModel(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		resolution string
		wantError  string
	}{
		{name: "fast rejects 1080p", model: "dreamina-seedance-2-0-fast-hc", resolution: "1080p", wantError: "resolution"},
		{name: "mini rejects 4k", model: "dreamina-seedance-2-0-mini-hc", resolution: "4k", wantError: "resolution"},
		{name: "fast accepts 720p", model: "dreamina-seedance-2-0-fast-hc", resolution: "720p"},
		{name: "mini accepts 480p", model: "dreamina-seedance-2-0-mini-hc", resolution: "480p"},
		{name: "full model accepts 1080p", model: "dreamina-seedance-2-0-hc", resolution: "1080p"},
		{name: "full model accepts 4k", model: "dreamina-seedance-2-0-hc", resolution: "4k"},
		{name: "full model normalizes 4K", model: "dreamina-seedance-2-0-hc", resolution: "4K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(
				http.MethodPost,
				"/v1/videos",
				strings.NewReader(`{"model":"`+tt.model+`","prompt":"test","seconds":4,"resolution":"`+tt.resolution+`"}`),
			)
			context.Request.Header.Set("Content-Type", "application/json")
			info := &relaycommon.RelayInfo{
				OriginModelName: tt.model,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: tt.model},
				TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
			}

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)
			if tt.wantError == "" {
				require.Nil(t, taskErr)
				request, err := relaycommon.GetTaskRequest(context)
				require.NoError(t, err)
				assert.Equal(t, strings.ToLower(tt.resolution), request.Resolution)
				return
			}
			require.NotNil(t, taskErr)
			assert.Contains(t, taskErr.Message, tt.wantError)
		})
	}
}

func TestDoResponseHidesServiceInferenceUpstreamIdentifiers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	response := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{
			"task": {
				"id": "upstream-task-id",
				"status": "queued",
				"outputs": ["https://storage.example/result.mp4"],
				"error": "internal upstream detail",
				"usage": {"completion_tokens": 12, "total_tokens": 34}
			}
		}`)),
	}

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(context, response, &relaycommon.RelayInfo{
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
		OriginModelName: "dreamina-seedance-2-0-hc",
	})

	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task-id", upstreamID)
	assert.NotContains(t, string(taskData), "upstream-task-id")
	assert.NotContains(t, string(taskData), "storage.example")
	assert.NotContains(t, string(taskData), "internal upstream detail")
	assert.Contains(t, string(taskData), `"status":"queued"`)
	assert.Contains(t, string(taskData), `"total_tokens":34`)
	assert.NotContains(t, recorder.Body.String(), "upstream-task-id")
	assert.Contains(t, recorder.Body.String(), "task_public")
}

func TestSanitizeTaskDataFailsClosedForInvalidJSON(t *testing.T) {
	sanitized := (&TaskAdaptor{}).SanitizeTaskData([]byte(`upstream-task-id https://storage.example/signed-result.mp4`))

	require.JSONEq(t, `{"task":{}}`, string(sanitized))
	assert.NotContains(t, string(sanitized), "upstream-task-id")
	assert.NotContains(t, string(sanitized), "storage.example")
}

func TestConvertToOpenAIVideoHidesServiceInferenceResultDetails(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
	}{
		{
			name: "success uses content proxy",
			task: &model.Task{
				TaskID: "task_public", Status: model.TaskStatusSuccess,
				PrivateData: model.TaskPrivateData{ResultURL: "https://storage.example/signed-result.mp4"},
			},
		},
		{
			name: "failure hides upstream reason",
			task: &model.Task{
				TaskID: "task_public", Status: model.TaskStatusFailure,
				FailReason: "upstream internal stack detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(tt.task)
			require.NoError(t, err)
			assert.NotContains(t, string(data), "storage.example")
			assert.NotContains(t, string(data), "upstream internal stack detail")
			if tt.task.Status == model.TaskStatusSuccess {
				assert.Contains(t, string(data), "/v1/videos/task_public/content")
			} else {
				assert.Contains(t, string(data), "Video generation failed")
			}
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

func TestBuildRequestBodyPassesResolvedAssetReferenceWithoutUploadingAgain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	context.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "dreamina-seedance-2-0-mini-hc", Prompt: "animate", Duration: 4,
		Images: []string{"asset://asset-local"},
	})
	common.SetContextKey(context, constant.ContextKeyVideoAssetReferences, map[string]string{
		"asset-local": "asset-upstream",
	})

	body, err := (&TaskAdaptor{}).BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	})

	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"url":"asset://asset-upstream"`)
}
