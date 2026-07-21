package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestImageRequestPassThroughDisabledForNormalizedEditUploads(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelSetting: dto.ChannelSettings{PassThroughBodyEnabled: true},
	}}

	assert.True(t, imageRequestPassThroughEnabled(context, info, true))
	context.Set(service.NormalizedImageUploadsContextKey, []service.StoredUploadedImage{{Path: "/tmp/edit.png"}})
	assert.False(t, imageRequestPassThroughEnabled(context, info, true))
	assert.True(t, imageRequestPassThroughEnabled(context, info, false))
}
