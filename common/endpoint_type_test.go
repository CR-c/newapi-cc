package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestGetEndpointTypesByChannelTypeRecognizesServiceInferenceVideo(t *testing.T) {
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		GetEndpointTypesByChannelType(
			constant.ChannelTypeServiceInference,
			"dreamina-seedance-2-0-fast-hc",
		),
	)
}

func TestGetEndpointTypesByChannelTypeRecognizesDedicatedVideoChannels(t *testing.T) {
	for _, channelType := range []int{
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeVidu,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeSora,
		constant.ChannelTypeKyyVideo,
	} {
		assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, GetEndpointTypesByChannelType(channelType, "video-model"))
	}
}
