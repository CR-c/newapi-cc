package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestVideoAssetReturnsUpstreamErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v1/sd/assets", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"success":false,"message":"URL is required"}`))
	}))
	defer server.Close()

	_, err := RequestVideoAsset(context.Background(), server.Client(), http.MethodPost, server.URL, "key", "", map[string]string{
		"Name": "x", "AssetType": "Image",
	})

	require.Error(t, err)
	var upstreamErr *VideoAssetUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusBadRequest, upstreamErr.StatusCode)
	assert.Equal(t, "URL is required", upstreamErr.Message)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestRequestVideoAssetSucceedsOnValidUpstreamResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"Id":"asset-upstream","base_resp":{"status_code":0,"status_msg":"success"}}}`))
	}))
	defer server.Close()

	result, err := RequestVideoAsset(context.Background(), server.Client(), http.MethodPost, server.URL, "key", "", map[string]string{
		"URL": "https://example.com/a.png", "Name": "a", "AssetType": "Image",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "asset-upstream", result.Data.ID)
}
