package relay

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/kyyvideo"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorReturnsKyyVideoAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKyyVideo)))
	_, ok := adaptor.(*kyyvideo.TaskAdaptor)
	require.True(t, ok)
}

func TestTaskModel2DtoHidesKyyUpstreamIdentifiers(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKyyVideo)),
		Status:   model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream-video-id",
			ResultURL:      "https://storage.example/signed-result.mp4",
		},
		Data: json.RawMessage(`{
			"id":"upstream-video-id",
			"model":"videos_pro",
			"status":"completed",
			"video_url":"https://storage.example/signed-result.mp4",
			"amount":0.32
		}`),
		Properties: model.Properties{UpstreamModelName: "private-kyy-model", OriginModelName: "videos"},
	}

	result := TaskModel2Dto(task)
	require.NotContains(t, result.ResultURL, "storage.example")
	require.Contains(t, result.ResultURL, "/v1/videos/task_public/content")
	require.NotContains(t, string(result.Data), "upstream-video-id")
	require.NotContains(t, string(result.Data), "storage.example")
	require.NotContains(t, string(result.Data), "amount")
	require.NotContains(t, string(result.Data), "videos_pro")
	properties := result.Properties.(model.Properties)
	require.Empty(t, properties.UpstreamModelName)
	require.Equal(t, "videos", properties.OriginModelName)
}

func TestTaskModel2DtoHidesServiceInferenceUpstreamIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
	}{
		{
			name: "success uses content proxy",
			task: &model.Task{
				TaskID:   "task_public",
				Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeServiceInference)),
				Status:   model.TaskStatusSuccess,
				PrivateData: model.TaskPrivateData{
					ResultURL: "https://storage.example/signed-result.mp4",
				},
				Data: json.RawMessage(`{
			"task": {
				"id": "upstream-task-id",
				"status": "completed",
				"outputs": ["https://storage.example/signed-result.mp4"],
				"error": "internal upstream detail",
				"usage": {"completion_tokens": 12, "total_tokens": 34}
			}
		}`),
			},
		},
		{
			name: "failure hides upstream reason",
			task: &model.Task{
				TaskID:     "task_public",
				Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeServiceInference)),
				Status:     model.TaskStatusFailure,
				FailReason: "upstream internal stack detail",
				Data:       json.RawMessage(`{"task":{"status":"failed","error":"internal error"}}`),
				Properties: model.Properties{UpstreamModelName: "private-service-model", OriginModelName: "public-model"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TaskModel2Dto(tt.task)
			properties := result.Properties.(model.Properties)
			require.NotContains(t, string(result.Data), "upstream-task-id")
			require.NotContains(t, string(result.Data), "storage.example")
			require.NotContains(t, string(result.Data), "internal upstream detail")
			require.NotContains(t, result.FailReason, "upstream internal stack detail")
			require.Empty(t, properties.UpstreamModelName)
			if tt.task.Status == model.TaskStatusSuccess {
				require.Contains(t, result.ResultURL, "/v1/videos/task_public/content")
				require.Contains(t, string(result.Data), `"total_tokens":34`)
			} else {
				require.Equal(t, "Video generation failed", result.FailReason)
				require.Empty(t, result.ResultURL)
			}
		})
	}
}

func TestTaskSubmitHTTPErrorHidesServiceInferenceResponseBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body: io.NopCloser(strings.NewReader(
			`{"id":"upstream-id","url":"https://storage.example/signed.mp4","stack":"internal detail"}`,
		)),
	}

	taskErr := taskSubmitHTTPError(response, constant.ChannelTypeServiceInference)

	require.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
	require.Equal(t, "Video upstream request failed", taskErr.Message)
	require.NotContains(t, taskErr.Message, "upstream-id")
	require.NotContains(t, taskErr.Message, "storage.example")
	require.NotContains(t, taskErr.Error.Error(), "internal detail")
}
