package common

import (
	"testing"

	basecommon "github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskSubmitReqParsesUnifiedMediaFieldsAndExplicitFalse(t *testing.T) {
	var request TaskSubmitReq
	require.NoError(t, basecommon.Unmarshal([]byte(`{
		"model":"video-ds-2.0",
		"prompt":"animate",
		"seconds":"10",
		"aspect_ratio":"9:16",
		"resolution":"720p",
		"images":["https://example.com/a.png"],
		"videos":["https://example.com/a.mp4"],
		"audios":["https://example.com/a.mp3"],
		"generate_audio":false,
		"watermark":false
	}`), &request))

	assert.Equal(t, "9:16", request.AspectRatio)
	assert.Equal(t, "720p", request.Resolution)
	assert.Equal(t, []string{"https://example.com/a.png"}, request.Images)
	assert.Equal(t, []string{"https://example.com/a.mp4"}, request.Videos)
	assert.Equal(t, []string{"https://example.com/a.mp3"}, request.Audios)
	require.NotNil(t, request.GenerateAudio)
	assert.False(t, *request.GenerateAudio)
	require.NotNil(t, request.Watermark)
	assert.False(t, *request.Watermark)
}

func TestTaskSubmitReqParsesVideoFrameFieldsAndExplicitAutoFaceFalse(t *testing.T) {
	var request TaskSubmitReq
	require.NoError(t, basecommon.Unmarshal([]byte(`{
		"model":"videos",
		"prompt":"animate",
		"first_image":"https://example.com/first.png",
		"last_image":"https://example.com/last.png",
		"auto_face":false
	}`), &request))

	assert.Equal(t, "https://example.com/first.png", request.FirstImage)
	assert.Equal(t, "https://example.com/last.png", request.LastImage)
	require.NotNil(t, request.AutoFace)
	assert.False(t, *request.AutoFace)
}

func TestTaskSubmitReqRejectsInvalidDurationTypes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "seconds integer overflow", body: `{"seconds":18446744073709551616}`},
		{name: "seconds non numeric string", body: `{"seconds":"ten"}`},
		{name: "duration decimal", body: `{"duration":1.5}`},
		{name: "duration boolean", body: `{"duration":true}`},
		{name: "duration object", body: `{"duration":{"value":10}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request TaskSubmitReq
			err := basecommon.Unmarshal([]byte(tt.body), &request)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "18446744073709551616")
		})
	}
}

func TestTaskSubmitReqAllowsNullDurationAliases(t *testing.T) {
	var request TaskSubmitReq
	require.NoError(t, basecommon.Unmarshal([]byte(`{"duration":null,"seconds":null}`), &request))
	assert.Zero(t, request.Duration)
	assert.Empty(t, request.Seconds)
}
