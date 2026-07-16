package relay

import (
	"encoding/json"
	"strconv"
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
			"status":"completed",
			"video_url":"https://storage.example/signed-result.mp4",
			"amount":0.32
		}`),
	}

	result := TaskModel2Dto(task)
	require.NotContains(t, result.ResultURL, "storage.example")
	require.Contains(t, result.ResultURL, "/v1/videos/task_public/content")
	require.NotContains(t, string(result.Data), "upstream-video-id")
	require.NotContains(t, string(result.Data), "storage.example")
	require.NotContains(t, string(result.Data), "amount")
}
