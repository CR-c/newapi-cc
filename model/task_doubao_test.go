package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
)

func TestInitTaskStoresSelectedDoubaoVideoKey(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeDoubaoVideo,
			ApiKey:      "selected-task-key",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"},
	}

	task := InitTask(constant.TaskPlatform("54"), info)

	assert.Equal(t, "selected-task-key", task.PrivateData.Key)
}
