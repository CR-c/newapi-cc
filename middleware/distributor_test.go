package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetModelRequestRecognizesPlaygroundVideos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/pg/videos",
		bytes.NewBufferString(`{"model":"sora-2","group":"vip","prompt":"A paper boat floating on a river"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	request, shouldSelectChannel, err := getModelRequest(context)

	require.NoError(t, err)
	require.NotNil(t, request)
	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "sora-2", request.Model)
	assert.Equal(t, "vip", request.Group)
	relayMode, exists := context.Get("relay_mode")
	require.True(t, exists)
	assert.Equal(t, relayconstant.RelayModeVideoSubmit, relayMode)
}

func TestResolveRequestGroupKeepsTokenGroupForExternalVideoRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(`{"model":"videos","group":"video-dddd","prompt":"test"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(context, constant.ContextKeyTokenGroup, "sd-video")
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "sd-video")

	request, shouldSelectChannel, err := getModelRequest(context)
	require.NoError(t, err)
	require.NotNil(t, request)
	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "video-dddd", request.Group)

	usingGroup, allowed := resolveRequestGroup(context, request.Group)

	assert.True(t, allowed)
	assert.Equal(t, "sd-video", usingGroup)
	assert.Equal(t, "sd-video", common.GetContextKeyString(context, constant.ContextKeyTokenGroup))
	assert.Equal(t, "sd-video", common.GetContextKeyString(context, constant.ContextKeyUsingGroup))
}

func TestGetModelRequestRecognizesPlaygroundImageGenerationGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/pg/images/generations",
		bytes.NewBufferString(`{"model":"gpt-image-1","group":"vip","prompt":"A paper boat"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	request, shouldSelectChannel, err := getModelRequest(context)

	require.NoError(t, err)
	require.NotNil(t, request)
	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "gpt-image-1", request.Model)
	assert.Equal(t, "vip", request.Group)
}

func TestGetModelRequestRecognizesPlaygroundImageEditMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-1"))
	require.NoError(t, writer.WriteField("group", "vip"))
	require.NoError(t, writer.WriteField("prompt", "Add a paper boat"))
	imagePart, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("image-data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/pg/images/edits",
		&body,
	)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())

	request, shouldSelectChannel, err := getModelRequest(context)

	require.NoError(t, err)
	require.NotNil(t, request)
	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "gpt-image-1", request.Model)
	assert.Equal(t, "vip", request.Group)
}

func TestRelayRequestPathNormalizesPlaygroundRoutes(t *testing.T) {
	assert.Equal(t, "/v1/images/generations", relayRequestPath("/pg/images/generations"))
	assert.Equal(t, "/v1/videos", relayRequestPath("/pg/videos"))
	assert.Equal(t, "/v1/chat/completions", relayRequestPath("/v1/chat/completions"))
}

func TestResolveVideoAssetReferencesEnforcesOwnershipAndChannel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoAsset{}, &model.Channel{}, &model.Ability{}))
	previousDB := model.DB
	previousCache := common.MemoryCacheEnabled
	model.DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousCache
	})
	channel := &model.Channel{Id: 59, Type: constant.ChannelTypeServiceInference, Status: common.ChannelStatusEnabled}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group: "video-dddd", Model: "dreamina-seedance-2-0-mini-hc", ChannelId: 59, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.VideoAsset{
		ID: "asset-local", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-upstream", AssetType: "Image",
	}).Error)

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc","images":["asset://asset-local"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)

	channelID, err := resolveVideoAssetReferences(context, "video-dddd")

	require.NoError(t, err)
	assert.Equal(t, 59, channelID)
	referencesValue, exists := common.GetContextKey(context, constant.ContextKeyVideoAssetReferences)
	require.True(t, exists)
	references, ok := referencesValue.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "asset-upstream", references["asset-local"])

	boundChannel, err := resolveVideoAssetChannel(context, "video-dddd", "dreamina-seedance-2-0-mini-hc")
	require.NoError(t, err)
	require.NotNil(t, boundChannel)
	assert.Equal(t, 59, boundChannel.Id)

	foreignContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	foreignContext.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc","images":["asset://asset-local"]
	}`))
	foreignContext.Request.Header.Set("Content-Type", "application/json")
	foreignContext.Set("id", 8)

	_, err = resolveVideoAssetReferences(foreignContext, "video-dddd")
	assert.ErrorContains(t, err, "not found or unavailable")
}

func TestResolveVideoAssetReferencesSupportsVideoAndAudioAssets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoAsset{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})
	for _, asset := range []*model.VideoAsset{
		{ID: "asset-image", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-image-upstream", AssetType: "Image"},
		{ID: "asset-video", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-video-upstream", AssetType: "Video"},
		{ID: "asset-audio", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-audio-upstream", AssetType: "Audio"},
	} {
		require.NoError(t, db.Create(asset).Error)
	}

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc",
		"images":["asset://asset-image"],
		"videos":["asset://asset-video"],
		"audios":["asset://asset-audio"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)

	channelID, err := resolveVideoAssetReferences(context, "video-dddd")

	require.NoError(t, err)
	assert.Equal(t, 59, channelID)
	referencesValue, exists := common.GetContextKey(context, constant.ContextKeyVideoAssetReferences)
	require.True(t, exists)
	references, ok := referencesValue.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "asset-image-upstream", references["asset-image"])
	assert.Equal(t, "asset-video-upstream", references["asset-video"])
	assert.Equal(t, "asset-audio-upstream", references["asset-audio"])
}

func TestResolveVideoAssetReferencesRejectsMismatchedAssetType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoAsset{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})
	require.NoError(t, db.Create(&model.VideoAsset{
		ID: "asset-video", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-video-upstream", AssetType: "Video",
	}).Error)

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc",
		"images":["asset://asset-video"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)

	_, err = resolveVideoAssetReferences(context, "video-dddd")

	assert.ErrorContains(t, err, "image references must use Image assets")
}

func TestResolveVideoAssetReferencesRejectsMoreThanThreeVideosAndAudiosBeforeLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "videos",
			body: `{"model":"dreamina-seedance-2-0-mini-hc","videos":["asset://asset-1","asset://asset-2","asset://asset-3","asset://asset-4"]}`,
			want: "at most 3 reference videos",
		},
		{
			name: "audios",
			body: `{"model":"dreamina-seedance-2-0-mini-hc","audios":["asset://asset-1","asset://asset-2","asset://asset-3","asset://asset-4"]}`,
			want: "at most 3 reference audios",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(test.body))
			context.Request.Header.Set("Content-Type", "application/json")

			_, err := resolveVideoAssetReferences(context, "video-dddd")

			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestResolveVideoAssetReferencesRejectsMoreThanNineImagesBeforeLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc",
		"images":["asset://asset-1","asset://asset-2","asset://asset-3","asset://asset-4","asset://asset-5","asset://asset-6","asset://asset-7","asset://asset-8","asset://asset-9","asset://asset-10"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")

	_, err := resolveVideoAssetReferences(context, "video-dddd")

	assert.ErrorContains(t, err, "at most 9")
}

func TestResolveVideoAssetReferencesAllowsNineImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc",
		"images":[
			"https://example.com/1.png","https://example.com/2.png","https://example.com/3.png",
			"https://example.com/4.png","https://example.com/5.png","https://example.com/6.png",
			"https://example.com/7.png","https://example.com/8.png","https://example.com/9.png"
		]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")

	channelID, err := resolveVideoAssetReferences(context, "video-dddd")

	require.NoError(t, err)
	assert.Zero(t, channelID)
}

func TestResolveVideoAssetReferencesPreservesOtherGroupImageLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"grok-image-video",
		"images":[
			"https://example.com/1.png","https://example.com/2.png","https://example.com/3.png",
			"https://example.com/4.png","https://example.com/5.png","https://example.com/6.png",
			"https://example.com/7.png"
		]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")

	channelID, err := resolveVideoAssetReferences(context, "grok按次")

	require.NoError(t, err)
	assert.Zero(t, channelID)
}

func TestDistributeResolvesVideoAssetsForSpecifiedChannel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VideoAsset{}, &model.Channel{}, &model.Ability{}))
	previousDB := model.DB
	previousCache := common.MemoryCacheEnabled
	model.DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.MemoryCacheEnabled = previousCache
	})
	require.NoError(t, db.Create(&model.Channel{
		Id: 59, Type: constant.ChannelTypeServiceInference, Status: common.ChannelStatusEnabled, Key: "upstream-key",
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group: "video-dddd", Model: "dreamina-seedance-2-0-mini-hc", ChannelId: 59, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.VideoAsset{
		ID: "asset-local", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-upstream", AssetType: "Image",
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(`{
		"model":"dreamina-seedance-2-0-mini-hc","images":["asset://asset-local"]
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "video-dddd")
	common.SetContextKey(context, constant.ContextKeyTokenSpecificChannelId, "59")

	Distribute()(context)

	assert.False(t, context.IsAborted())
	assert.Equal(t, 59, common.GetContextKeyInt(context, constant.ContextKeyVideoAssetChannelId))
	referencesValue, exists := common.GetContextKey(context, constant.ContextKeyVideoAssetReferences)
	require.True(t, exists)
	references, ok := referencesValue.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "asset-upstream", references["asset-local"])
}
