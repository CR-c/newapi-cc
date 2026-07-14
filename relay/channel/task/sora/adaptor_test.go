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
