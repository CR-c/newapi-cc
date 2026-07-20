package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
