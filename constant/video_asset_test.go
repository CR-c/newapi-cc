package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVideoAssetGroup(t *testing.T) {
	t.Parallel()

	assert.True(t, IsVideoAssetGroup(VideoAssetGroupDefault))
	assert.True(t, IsVideoAssetGroup(VideoAssetGroupSD))
	assert.True(t, IsVideoAssetGroup(VideoAssetGroupCorn))
	assert.True(t, IsVideoAssetGroup(VideoAssetGroupCornVP00001))
	assert.True(t, IsVideoAssetGroup("video-dddd"))
	assert.True(t, IsVideoAssetGroup("sd-dddd"))
	assert.True(t, IsVideoAssetGroup("Corn专用"))
	assert.True(t, IsVideoAssetGroup("Corn-vp00001"))

	assert.False(t, IsVideoAssetGroup(""))
	assert.False(t, IsVideoAssetGroup("sd-token"))
	assert.False(t, IsVideoAssetGroup("sd-video"))
	assert.False(t, IsVideoAssetGroup("default"))
}
