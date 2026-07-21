package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestImageEditRequestBodyLimitRejectsOversizedMultipartBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ImageEditRequestBodyLimit())
	handled := false
	router.POST("/v1/images/edits", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader([]byte("small")))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	request.ContentLength = constant.MaxImageEditMultipartBytes + 1
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.False(t, handled)
}
