package helper

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestGetAndValidOpenAIImageRequestMultipartStream verifies multipart image
// edit parsing: the stream field is parsed and validated, and the request body
// stays replayable for the upstream request.
func TestGetAndValidOpenAIImageRequestMultipartStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, streamValue string, withImage bool) (*gin.Context, string) {
		t.Setenv("IMAGE_UPLOAD_DIR", t.TempDir())
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "edit this image"))
		require.NoError(t, writer.WriteField("stream", streamValue))
		if withImage {
			part, err := writer.CreateFormFile("image", "input.png")
			require.NoError(t, err)
			input := image.NewNRGBA(image.Rect(0, 0, 320, 320))
			input.Set(1, 1, color.NRGBA{R: 255, A: 255})
			require.NoError(t, png.Encode(part, input))
		}
		require.NoError(t, writer.Close())
		originalBody := body.String()

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return c, originalBody
	}

	t.Run("valid stream value keeps body replayable", func(t *testing.T) {
		c, originalBody := newContext(t, "true", true)

		req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.NoError(t, err)
		require.NotNil(t, req.Stream)
		require.True(t, *req.Stream)
		require.True(t, req.IsStream(c))

		bodyAfterValidation, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, originalBody, string(bodyAfterValidation))

		form, err := common.ParseMultipartFormReusable(c)
		require.NoError(t, err)
		require.Equal(t, "true", url.Values(form.Value).Get("stream"))
		require.Len(t, form.File["image"], 1)

		value, exists := c.Get(service.NormalizedImageUploadsContextKey)
		require.True(t, exists)
		stored := value.([]service.StoredUploadedImage)
		require.Len(t, stored, 1)
		require.FileExists(t, stored[0].Path)
		CleanupNormalizedImageUploads(c)
		_, statErr := os.Stat(stored[0].Path)
		require.ErrorIs(t, statErr, os.ErrNotExist)
	})

	t.Run("invalid stream value is rejected", func(t *testing.T) {
		c, _ := newContext(t, "notabool", false)

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid stream value")
	})
}

func TestGetAndValidOpenAIImageRequestRejectsTooManyMultipartImagesBeforePersistence(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("IMAGE_UPLOAD_DIR", storageDir)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit these images"))
	for index := 0; index <= maxImageEditFiles; index++ {
		part, err := writer.CreateFormFile("image[]", fmt.Sprintf("input-%d.png", index))
		require.NoError(t, err)
		input := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		require.NoError(t, png.Encode(part, input))
	}
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

	require.EqualError(t, err, fmt.Sprintf("at most %d edit images are allowed", maxImageEditFiles))
	entries, readErr := os.ReadDir(storageDir)
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func TestGetAndValidOpenAIImageRequestRejectsExcessiveTotalPixelsBeforePersistence(t *testing.T) {
	storageDir := t.TempDir()
	t.Setenv("IMAGE_UPLOAD_DIR", storageDir)
	large := image.NewGray(image.Rect(0, 0, 4096, 4096))
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, large))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit these images"))
	for index := 0; index < 5; index++ {
		part, err := writer.CreateFormFile("image[]", fmt.Sprintf("large-%d.png", index))
		require.NoError(t, err)
		_, err = part.Write(encoded.Bytes())
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

	require.EqualError(t, err, fmt.Sprintf("image edit uploads exceed %d total pixels", maxImageEditTotalPixels))
	entries, readErr := os.ReadDir(storageDir)
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func TestGetAndValidOpenAIImageRequestRejectsInvalidMultipartImage(t *testing.T) {
	t.Setenv("IMAGE_UPLOAD_DIR", t.TempDir())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit this image"))
	part, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("\x89PNG\r\n\x1a\ntruncated"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid image upload")
}

// TestGetAndValidOpenAIImageRequestNBounds guards the billing invariant that
// the image generation count can never reach quota calculation with a value
// large enough to overflow int64 into a negative charge.
func TestGetAndValidOpenAIImageRequestNBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newJSONContext := func(t *testing.T, body string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}

	boundErr := fmt.Sprintf("n must be an integer between 1 and %d", dto.MaxImageN)

	tests := []struct {
		name    string
		body    string
		wantErr string
		wantN   uint
	}{
		{
			name:    "overflowed uint64 n is rejected",
			body:    `{"model":"gpt-image-1","prompt":"a cat","n":18446744073686646784}`,
			wantErr: boundErr,
		},
		{
			name:    "n above max is rejected",
			body:    fmt.Sprintf(`{"model":"gpt-image-1","prompt":"a cat","n":%d}`, dto.MaxImageN+1),
			wantErr: boundErr,
		},
		{
			name:  "n at max is accepted",
			body:  fmt.Sprintf(`{"model":"gpt-image-1","prompt":"a cat","n":%d}`, dto.MaxImageN),
			wantN: dto.MaxImageN,
		},
		{
			name:  "absent n defaults to 1",
			body:  `{"model":"gpt-image-1","prompt":"a cat"}`,
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newJSONContext(t, tt.body)
			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, req.N)
			require.Equal(t, tt.wantN, *req.N)
		})
	}

	t.Run("negative multipart n is rejected", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "edit this image"))
		require.NoError(t, writer.WriteField("n", "-22904832"))
		require.NoError(t, writer.Close())

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.Error(t, err)
		require.Contains(t, err.Error(), boundErr)
	})
}

func TestGetAndValidOpenAIImageRequestPreservesInputReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/images/generations",
		bytes.NewBufferString(`{
			"model":"gpt-image-2",
			"prompt":"use these references",
			"input_reference":["https://example.com/one.png","data:image/png;base64,AAAA"]
		}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	require.NoError(t, err)
	encoded, err := common.Marshal(req)
	require.NoError(t, err)
	require.JSONEq(t, `[
		"https://example.com/one.png",
		"data:image/png;base64,AAAA"
	]`, string(req.InputReference))
	require.Contains(t, string(encoded), `"input_reference"`)
}
