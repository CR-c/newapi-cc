package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
