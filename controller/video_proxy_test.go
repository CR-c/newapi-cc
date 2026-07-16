package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
