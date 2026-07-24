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

	for _, path := range []string{"/docs.html", "/docs-official.html", "/docs.css", "/docs.js"} {
		path := path
		router.GET(path, func(c *gin.Context) {})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", path, nil)
		router.ServeHTTP(recorder, request)

		require.Equal(t, 200, recorder.Code, path)
		assert.Equal(
			t,
			"no-cache, must-revalidate",
			recorder.Header().Get("Cache-Control"),
			path,
		)
	}
}
