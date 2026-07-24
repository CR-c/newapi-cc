package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoProxyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}, &model.User{}))
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

func TestVideoProxyAdminCanPreviewOtherUsersTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoProxyTestDB(t)

	channel := &model.Channel{
		Id:     42,
		Type:   constant.ChannelTypeKling,
		Status: common.ChannelStatusEnabled,
		Name:   "kling-test",
	}
	require.NoError(t, db.Create(channel).Error)

	owner := &model.User{Id: 7, Username: "owner", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff-owner"}
	admin := &model.User{Id: 1, Username: "admin", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "aff-admin"}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(admin).Error)

	task := &model.Task{
		TaskID:    "task_other_user_video",
		UserId:    7,
		ChannelId: 42,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "data:video/mp4;base64,AAAA",
		},
	}
	require.NoError(t, db.Create(task).Error)

	// Non-owner, non-admin: should not see other users' tasks
	foreignRecorder := httptest.NewRecorder()
	foreignCtx, _ := gin.CreateTestContext(foreignRecorder)
	foreignCtx.Params = gin.Params{{Key: "task_id", Value: "task_other_user_video"}}
	foreignCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_other_user_video/content", nil)
	foreignCtx.Set("id", 8)
	VideoProxy(foreignCtx)
	require.Equal(t, http.StatusNotFound, foreignRecorder.Code)
	assert.Contains(t, foreignRecorder.Body.String(), "Task not found")

	// Admin: should preview any user's successful video task
	adminRecorder := httptest.NewRecorder()
	adminCtx, _ := gin.CreateTestContext(adminRecorder)
	adminCtx.Params = gin.Params{{Key: "task_id", Value: "task_other_user_video"}}
	adminCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_other_user_video/content", nil)
	adminCtx.Set("id", 1)
	VideoProxy(adminCtx)
	require.Equal(t, http.StatusOK, adminRecorder.Code)
	assert.Equal(t, "video/mp4", adminRecorder.Header().Get("Content-Type"))
	assert.NotEmpty(t, adminRecorder.Body.Bytes())
}

func TestClientWithoutAuthorizationOnRedirectDoesNotLeakUpstreamKey(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/content" {
			assert.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
			http.Redirect(writer, request, upstream.URL+"/result.mp4", http.StatusTemporaryRedirect)
			return
		}
		assert.Equal(t, "/result.mp4", request.URL.Path)
		assert.Empty(t, request.Header.Get("Authorization"))
		writer.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	request, err := http.NewRequest(http.MethodGet, upstream.URL+"/content", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer upstream-secret")
	response, err := clientWithoutAuthorizationOnRedirect(http.DefaultClient).Do(request)

	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestBuildDoubaoVideoContentRequestUsesPrivateAuthenticatedEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://placeholder.invalid", nil)
	channel := &model.Channel{Key: "current-channel-key"}
	task := &model.Task{
		TaskID: "task_public_123",
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "seedance/id with spaces",
			Key:            "selected-task-key",
		},
	}

	videoURL, err := buildDoubaoVideoContentRequest(
		request,
		"https://api.guyscode.com/",
		channel,
		task,
	)

	require.NoError(t, err)
	assert.Equal(t, "Bearer selected-task-key", request.Header.Get("Authorization"))
	assert.Equal(t,
		"https://api.guyscode.com/api/v3/contents/generations/tasks/seedance%2Fid%20with%20spaces/content",
		videoURL,
	)
}

func TestResolveVideoChannelBaseURLUsesChannelTypeDefault(t *testing.T) {
	channel := &model.Channel{Type: 54}

	assert.Equal(t, "https://ark.cn-beijing.volces.com", resolveVideoChannelBaseURL(channel))
}

func TestWriteVideoDataURLRejectsExecutableContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	err := writeVideoDataURL(context, "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==")

	require.Error(t, err)
	assert.Empty(t, recorder.Body.String())
}

func TestWriteVideoDataURLDisablesSharedCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	require.NoError(t, writeVideoDataURL(context, "data:video/mp4;base64,AAAA"))
	assert.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
}
