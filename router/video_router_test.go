package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestVideoAssetRoutesRequireTokenAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v1/sd/assets", nil),
		httptest.NewRequest(http.MethodGet, "/v1/sd/assets/asset-local", nil),
	} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code, request.Method)
	}
}
