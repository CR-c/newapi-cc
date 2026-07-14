package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUploadAndServePlaygroundAsset(t *testing.T) {
	t.Setenv("PLAYGROUND_ASSET_DIR", t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	fileWriter, err := writer.CreateFormFile("file", "reference.png")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("\x89PNG\r\n\x1a\nreference-image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	uploadRecorder := httptest.NewRecorder()
	uploadContext, _ := gin.CreateTestContext(uploadRecorder)
	uploadContext.Request = httptest.NewRequest(http.MethodPost, "https://vp.example/pg/assets", &requestBody)
	uploadContext.Request.Header.Set("Content-Type", writer.FormDataContentType())
	uploadContext.Set("id", 7)
	UploadPlaygroundAsset(uploadContext)

	require.Equal(t, http.StatusOK, uploadRecorder.Code)
	uploadResponse := struct {
		Success bool `json:"success"`
		Data    struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"data"`
	}{}
	require.NoError(t, common.Unmarshal(uploadRecorder.Body.Bytes(), &uploadResponse))
	assert.True(t, uploadResponse.Success)
	assert.NotEmpty(t, uploadResponse.Data.ID)
	assert.Contains(t, uploadResponse.Data.URL, "/pg/assets/"+uploadResponse.Data.ID+"/reference.png")

	serveRecorder := httptest.NewRecorder()
	serveContext, _ := gin.CreateTestContext(serveRecorder)
	serveContext.Request = httptest.NewRequest(http.MethodGet, uploadResponse.Data.URL, nil)
	serveContext.Params = gin.Params{{Key: "asset_id", Value: uploadResponse.Data.ID}, {Key: "filename", Value: "reference.png"}}
	GetPlaygroundAsset(serveContext)

	assert.Equal(t, http.StatusOK, serveRecorder.Code)
	assert.Equal(t, "image/png", serveRecorder.Header().Get("Content-Type"))
	assert.Equal(t, []byte("\x89PNG\r\n\x1a\nreference-image"), serveRecorder.Body.Bytes())
}

func TestGetPlaygroundAssetRejectsExpiredAsset(t *testing.T) {
	t.Setenv("PLAYGROUND_ASSET_DIR", t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, db.Create(&model.PlaygroundAsset{
		ID: "expiredasset", UserID: 7, Kind: "image", Filename: "old.png",
		ContentType: "image/png", StorageName: "expiredasset.png", CreatedAt: time.Now().Unix() - 100, ExpiresAt: time.Now().Unix() - 1,
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/pg/assets/expiredasset/old.png", nil)
	context.Params = gin.Params{{Key: "asset_id", Value: "expiredasset"}, {Key: "filename", Value: "old.png"}}
	GetPlaygroundAsset(context)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestUploadPlaygroundAssetRejectsSpoofedContentType(t *testing.T) {
	t.Setenv("PLAYGROUND_ASSET_DIR", t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="payload.png"`)
	header.Set("Content-Type", "image/png")
	fileWriter, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("<html><script>alert(1)</script></html>"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/assets", &requestBody)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("id", 7)
	UploadPlaygroundAsset(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestSafePlaygroundAssetFilenameUsesDetectedExtension(t *testing.T) {
	assert.Equal(t, "reference.jpg", safePlaygroundAssetFilename("../../reference.txt", ".jpg"))
	assert.Equal(t, "asset.png", safePlaygroundAssetFilename("", ".png"))
}

func TestPlaygroundAssetSignatureRejectsArbitraryBinary(t *testing.T) {
	assert.False(t, hasPlaygroundAssetSignature("video", []byte("arbitrary binary payload")))
	assert.False(t, hasPlaygroundAssetSignature("audio", []byte("arbitrary binary payload")))
	assert.True(t, hasPlaygroundAssetSignature("video", []byte("\x00\x00\x00\x18ftypisom")))
	assert.True(t, hasPlaygroundAssetSignature("audio", []byte("RIFF\x00\x00\x00\x00WAVE")))
}
