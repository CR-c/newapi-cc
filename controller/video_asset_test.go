package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
	return db
}

func TestCreateVideoAssetUsesUpstreamCredentialAndStoresOwnership(t *testing.T) {
	t.Setenv("VIDEO_ASSET_ALLOWED_DOMAINS", "example.com")
	db := setupVideoAssetTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v1/sd/assets", request.URL.Path)
		assert.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
		var payload struct {
			URL       string `json:"URL"`
			Name      string `json:"Name"`
			AssetType string `json:"AssetType"`
		}
		require.NoError(t, common.DecodeJson(request.Body, &payload))
		assert.Equal(t, "https://example.com/avatar.jpg", payload.URL)
		assert.Equal(t, "avatar_front", payload.Name)
		assert.Equal(t, "Image", payload.AssetType)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-upstream","base_resp":{"status_code":0,"status_msg":"success"}}}`))
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/sd/assets", bytes.NewBufferString(`{
		"URL":"https://example.com/avatar.jpg","Name":"avatar_front","AssetType":"Image"
	}`))
	context.Request.Header.Set("Authorization", "Bearer user-token")
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "video-dddd")
	channel := &model.Channel{Id: 59, Type: constant.ChannelTypeServiceInference, Status: common.ChannelStatusEnabled, BaseURL: &server.URL, Key: "upstream-secret"}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "video-dddd", Model: "dreamina-seedance-2-0-mini-hc", ChannelId: 59, Enabled: true}).Error)

	CreateVideoAsset(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"Id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Data.ID)
	assert.NotEqual(t, "asset-upstream", response.Data.ID)

	stored, exists, err := model.GetVideoAssetForUser(7, response.Data.ID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, "asset-upstream", stored.UpstreamID)
	assert.Equal(t, 59, stored.ChannelID)

	var count int64
	require.NoError(t, db.Model(&model.VideoAsset{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestGetVideoAssetUsesStoredChannelAndHidesForeignAssets(t *testing.T) {
	db := setupVideoAssetTestDB(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		assert.Equal(t, "/v1/sd/assets/asset-upstream", request.URL.Path)
		assert.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-upstream","Status":"Active","AssetType":"Image","Name":"avatar_front","URL":"https://cdn.example/avatar.jpg","GroupId":null,"CreateTime":"2026-07-04T12:15:34Z","UpdateTime":"2026-07-04T12:15:36Z","base_resp":{"status_code":0,"status_msg":"success"}}}`))
	}))
	defer server.Close()

	channel := &model.Channel{
		Id: 59, Type: constant.ChannelTypeServiceInference, Status: common.ChannelStatusEnabled,
		BaseURL: &server.URL, Key: "upstream-secret",
	}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.VideoAsset{
		ID: "asset-local", UserID: 7, Group: "video-dddd", ChannelID: 59, UpstreamID: "asset-upstream",
		AssetType: "Image", Name: "avatar_front",
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/sd/assets/asset-local", nil)
	context.Params = gin.Params{{Key: "asset_id", Value: "asset-local"}}
	context.Set("id", 7)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "video-dddd")

	GetVideoAsset(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"Id":"asset-local"`)
	assert.NotContains(t, recorder.Body.String(), "asset-upstream")
	assert.Equal(t, 1, requests)

	foreignRecorder := httptest.NewRecorder()
	foreignContext, _ := gin.CreateTestContext(foreignRecorder)
	foreignContext.Request = httptest.NewRequest(http.MethodGet, "/v1/sd/assets/asset-local", nil)
	foreignContext.Params = gin.Params{{Key: "asset_id", Value: "asset-local"}}
	foreignContext.Set("id", 8)
	common.SetContextKey(foreignContext, constant.ContextKeyUsingGroup, "video-dddd")

	GetVideoAsset(foreignContext)

	assert.Equal(t, http.StatusNotFound, foreignRecorder.Code)
	assert.Equal(t, 1, requests)
}

func TestCreateVideoAssetRejectsOtherGroups(t *testing.T) {
	setupVideoAssetTestDB(t)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/sd/assets", bytes.NewBufferString(`{"URL":"https://example.com/avatar.jpg","Name":"avatar","AssetType":"Image"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "sd-video")

	CreateVideoAsset(context)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreateVideoAssetRejectsMultiKeyChannels(t *testing.T) {
	t.Setenv("VIDEO_ASSET_ALLOWED_DOMAINS", "example.com")
	db := setupVideoAssetTestDB(t)
	channel := &model.Channel{
		Id: 59, Type: constant.ChannelTypeServiceInference, Status: common.ChannelStatusEnabled,
		Key: "key-one\nkey-two", ChannelInfo: model.ChannelInfo{IsMultiKey: true, MultiKeySize: 2},
	}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group: "video-dddd", Model: "dreamina-seedance-2-0-mini-hc", ChannelId: 59, Enabled: true,
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/sd/assets", bytes.NewBufferString(`{
		"URL":"https://example.com/avatar.jpg","Name":"avatar","AssetType":"Image"
	}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("id", 7)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "video-dddd")

	CreateVideoAsset(context)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestValidateVideoAssetSourceURLRequiresPublicHTTPS(t *testing.T) {
	t.Setenv("VIDEO_ASSET_ALLOWED_DOMAINS", "example.com,*.trusted.example")
	assert.NoError(t, validateVideoAssetSourceURL("https://example.com/reference.png"))
	assert.True(t, videoAssetDomainAllowed("cdn.trusted.example"))
	assert.False(t, videoAssetDomainAllowed("trusted.example"))
	assert.Error(t, validateVideoAssetSourceURL("http://example.com/reference.png"))
	assert.Error(t, validateVideoAssetSourceURL("https://127.0.0.1/reference.png"))
	assert.Error(t, validateVideoAssetSourceURL("https://example.com:8443/reference.png"))
	assert.Error(t, validateVideoAssetSourceURL("https://untrusted.example/reference.png"))
}

func TestAssetBoundVideoSubmissionIsNotRetried(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(context, constant.ContextKeyVideoAssetChannelId, 59)

	assert.False(t, shouldRetryTaskRelay(context, 59, &dto.TaskError{
		StatusCode: http.StatusInternalServerError,
	}, 2))
}
