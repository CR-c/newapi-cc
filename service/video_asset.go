package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const VideoAssetResponseMaxBytes = 1 << 20

type VideoAssetBaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type VideoAssetUpstreamData struct {
	ID         string                  `json:"Id"`
	Status     string                  `json:"Status,omitempty"`
	AssetType  string                  `json:"AssetType,omitempty"`
	Name       string                  `json:"Name,omitempty"`
	URL        string                  `json:"URL,omitempty"`
	GroupID    any                     `json:"GroupId"`
	CreateTime string                  `json:"CreateTime,omitempty"`
	UpdateTime string                  `json:"UpdateTime,omitempty"`
	BaseResp   *VideoAssetBaseResponse `json:"base_resp"`
}

type VideoAssetUpstreamResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    VideoAssetUpstreamData `json:"data"`
}

// VideoAssetUpstreamError is returned when the asset library upstream rejects a request.
// StatusCode is the HTTP status from upstream (0 when the body was a logical failure on 2xx).
type VideoAssetUpstreamError struct {
	StatusCode int
	Message    string
}

func (e *VideoAssetUpstreamError) Error() string {
	if e == nil {
		return "video asset upstream request was rejected"
	}
	if e.Message != "" && e.StatusCode > 0 {
		return fmt.Sprintf("video asset upstream returned status %d: %s", e.StatusCode, e.Message)
	}
	if e.Message != "" {
		return e.Message
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("video asset upstream returned status %d", e.StatusCode)
	}
	return "video asset upstream request was rejected"
}

func RequestVideoAsset(ctx context.Context, client *http.Client, method, baseURL, apiKey, assetID string, payload any) (*VideoAssetUpstreamResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}
	requestURL := strings.TrimRight(baseURL, "/") + "/v1/sd/assets"
	if assetID != "" {
		requestURL += "/" + url.PathEscape(assetID)
	}
	var body io.Reader
	if payload != nil {
		data, err := common.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("video asset upstream request failed")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, VideoAssetResponseMaxBytes+1))
	if err != nil {
		return nil, errors.New("video asset upstream response failed")
	}
	if len(data) > VideoAssetResponseMaxBytes {
		return nil, errors.New("video asset upstream response is too large")
	}

	var result VideoAssetUpstreamResponse
	unmarshalErr := common.Unmarshal(data, &result)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(result.Message)
		if unmarshalErr != nil || message == "" {
			message = strings.TrimSpace(string(data))
		}
		if message == "" {
			message = fmt.Sprintf("status %d", response.StatusCode)
		}
		return nil, &VideoAssetUpstreamError{StatusCode: response.StatusCode, Message: message}
	}
	if unmarshalErr != nil {
		return nil, errors.New("video asset upstream returned an invalid response")
	}
	if !result.Success || result.Data.BaseResp == nil || result.Data.BaseResp.StatusCode != 0 {
		message := strings.TrimSpace(result.Message)
		if message == "" && result.Data.BaseResp != nil {
			message = strings.TrimSpace(result.Data.BaseResp.StatusMsg)
		}
		if message == "" {
			message = "video asset upstream request was rejected"
		}
		return nil, &VideoAssetUpstreamError{StatusCode: response.StatusCode, Message: message}
	}
	return &result, nil
}
