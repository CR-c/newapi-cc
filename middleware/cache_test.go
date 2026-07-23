package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheRevalidatesDocumentationPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cache())
	router.GET("/docs.html", func(c *gin.Context) {})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/docs.html", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, 200, recorder.Code)
	assert.Equal(t, "no-cache, must-revalidate", recorder.Header().Get("Cache-Control"))
}
