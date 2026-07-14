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
