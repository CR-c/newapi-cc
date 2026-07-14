package serviceinference

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
