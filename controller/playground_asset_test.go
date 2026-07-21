package controller

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
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
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}, &model.PlaygroundAssetStorageState{}, &model.PlaygroundAssetStorageReservation{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	fileWriter, err := writer.CreateFormFile("file", "reference.png")
	require.NoError(t, err)
	expectedImage := image.NewNRGBA(image.Rect(0, 0, 320, 320))
	expectedImage.Set(12, 34, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
	require.NoError(t, png.Encode(fileWriter, expectedImage))
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
	decoded, format, err := image.Decode(bytes.NewReader(serveRecorder.Body.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, "png", format)
	assert.Equal(t, expectedImage.Bounds(), decoded.Bounds())
}

func TestUploadPlaygroundAssetRejectsCorruptImageBeforePersistence(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", storageDir)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}, &model.PlaygroundAssetStorageState{}, &model.PlaygroundAssetStorageReservation{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	fileWriter, err := writer.CreateFormFile("file", "corrupt.png")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("\x89PNG\r\n\x1a\ntruncated"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/assets", &requestBody)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("id", 7)

	UploadPlaygroundAsset(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var count int64
	require.NoError(t, db.Model(&model.PlaygroundAsset{}).Count(&count).Error)
	assert.Zero(t, count)
	entries, err := os.ReadDir(storageDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestUploadPlaygroundAssetRejectsReferenceImageOutsideProviderDimensions(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", storageDir)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}, &model.PlaygroundAssetStorageState{}, &model.PlaygroundAssetStorageReservation{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	fileWriter, err := writer.CreateFormFile("file", "tiny.png")
	require.NoError(t, err)
	require.NoError(t, png.Encode(fileWriter, image.NewNRGBA(image.Rect(0, 0, 64, 64))))
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/assets", &requestBody)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("id", 7)

	UploadPlaygroundAsset(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	entries, err := os.ReadDir(storageDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestUploadPlaygroundAssetRechecksQuotaAfterImageNormalization(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", storageDir)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.PlaygroundAsset{}, &model.PlaygroundAssetStorageState{}, &model.PlaygroundAssetStorageReservation{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	input := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	state := uint32(1)
	for index := range input.Pix {
		state = state*1664525 + 1013904223
		input.Pix[index] = uint8(state >> 24)
	}
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, input, &jpeg.Options{Quality: 70}))
	preview, err := service.NormalizeAndStoreUploadedImage(
		bytes.NewReader(encoded.Bytes()), t.TempDir(), "preview", int64(encoded.Len()), service.MaxNormalizedUploadedImageBytes,
	)
	require.NoError(t, err)
	require.Greater(t, preview.Size, int64(encoded.Len()))

	now := time.Now().Unix()
	existingSize := model.PlaygroundAssetMaxBytesPerUser - preview.Size + 1
	require.NoError(t, db.Create(&model.PlaygroundAsset{
		ID: "existing", UserID: 7, Kind: "image", Filename: "existing.png", ContentType: "image/png",
		StorageName: "existing.png", Size: existingSize, CreatedAt: now, ExpiresAt: now + 3600,
	}).Error)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	require.NoError(t, writer.WriteField("kind", "image"))
	fileWriter, err := writer.CreateFormFile("file", "reference.jpg")
	require.NoError(t, err)
	_, err = fileWriter.Write(encoded.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/assets", &requestBody)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("id", 7)

	UploadPlaygroundAsset(context)

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	entries, err := os.ReadDir(storageDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
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
