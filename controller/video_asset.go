package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	videoAssetGroup        = "video-dddd"
	videoAssetRequestLimit = 64 << 10
	videoAssetRequestWait  = 30 * time.Second
)

type createVideoAssetRequest struct {
	URL       string `json:"URL"`
	Name      string `json:"Name"`
	AssetType string `json:"AssetType"`
}

func videoAssetError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}

func requireVideoAssetGroup(c *gin.Context) bool {
	if common.GetContextKeyString(c, constant.ContextKeyUsingGroup) == videoAssetGroup {
		return true
	}
	videoAssetError(c, http.StatusForbidden, "this endpoint is only available to the video-dddd group")
	return false
}

func CreateVideoAsset(c *gin.Context) {
	if !requireVideoAssetGroup(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, videoAssetRequestLimit)
	var request createVideoAssetRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		videoAssetError(c, http.StatusBadRequest, "invalid JSON request body")
		return
	}
	request.URL = strings.TrimSpace(request.URL)
	request.Name = strings.TrimSpace(request.Name)
	request.AssetType = strings.TrimSpace(request.AssetType)
	if err := validateVideoAssetSourceURL(request.URL); err != nil {
		videoAssetError(c, http.StatusBadRequest, "URL must use an administrator-approved HTTPS media domain")
		return
	}
	if len([]rune(request.Name)) == 0 || len([]rune(request.Name)) > 128 || strings.IndexFunc(request.Name, unicode.IsControl) >= 0 {
		videoAssetError(c, http.StatusBadRequest, "Name must contain 1 to 128 characters without control characters")
		return
	}
	if request.AssetType != "Image" && request.AssetType != "Video" && request.AssetType != "Audio" {
		videoAssetError(c, http.StatusBadRequest, "AssetType must be Image, Video, or Audio")
		return
	}

	channel, err := model.GetEnabledChannelByGroupAndType(videoAssetGroup, constant.ChannelTypeServiceInference)
	if err != nil {
		logger.LogError(c.Request.Context(), "video asset channel lookup failed: "+err.Error())
		videoAssetError(c, http.StatusInternalServerError, "failed to select video asset channel")
		return
	}
	if channel == nil || channel.ChannelInfo.IsMultiKey || strings.TrimSpace(channel.Key) == "" {
		videoAssetError(c, http.StatusServiceUnavailable, "video asset service is unavailable")
		return
	}
	client, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
	if err != nil {
		logger.LogError(c.Request.Context(), "video asset proxy setup failed: "+err.Error())
		videoAssetError(c, http.StatusBadGateway, "video asset service is unavailable")
		return
	}
	requestContext, cancel := context.WithTimeout(c.Request.Context(), videoAssetRequestWait)
	defer cancel()
	upstream, err := service.RequestVideoAsset(requestContext, client, http.MethodPost, channel.GetBaseURL(), channel.Key, "", request)
	if err != nil || upstream.Data.ID == "" {
		if err == nil {
			err = errors.New("upstream response did not include an asset id")
		}
		logger.LogError(c.Request.Context(), "video asset create failed: "+err.Error())
		videoAssetError(c, http.StatusBadGateway, "failed to create video asset")
		return
	}
	asset := &model.VideoAsset{
		UserID: c.GetInt("id"), TokenID: c.GetInt("token_id"), Group: videoAssetGroup,
		ChannelID: channel.Id, UpstreamID: upstream.Data.ID, AssetType: request.AssetType,
		Name: request.Name, SourceURL: request.URL,
	}
	if err = model.CreateVideoAsset(asset); err != nil {
		logger.LogError(c.Request.Context(), "video asset mapping create failed: "+err.Error())
		videoAssetError(c, http.StatusInternalServerError, "failed to save video asset")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"Id":        asset.ID,
			"base_resp": upstream.Data.BaseResp,
		},
	})
}

func validateVideoAssetSourceURL(value string) error {
	if err := taskcommon.ValidateMediaURL(value, false, taskcommon.MediaURLPortPolicyEnforceConfigured); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Port() != "" {
		return errors.New("video asset URL must use HTTPS on the default port")
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if net.ParseIP(host) != nil || !videoAssetDomainAllowed(host) {
		return errors.New("video asset URL domain is not approved")
	}
	return common.ValidateURLWithFetchSetting(
		value,
		true,
		false,
		false,
		false,
		nil,
		nil,
		[]string{"443"},
		true,
	)
}

func videoAssetDomainAllowed(host string) bool {
	configured := os.Getenv("SESSION_COOKIE_TRUSTED_URL") + "," + os.Getenv("VIDEO_ASSET_ALLOWED_DOMAINS")
	for _, rawEntry := range strings.Split(configured, ",") {
		entry := strings.TrimSpace(strings.ToLower(rawEntry))
		if entry == "" {
			continue
		}
		if parsed, err := url.Parse(entry); err == nil && parsed.Hostname() != "" {
			entry = parsed.Hostname()
		}
		entry = strings.TrimSuffix(entry, ".")
		if strings.HasPrefix(entry, "*.") {
			base := strings.TrimPrefix(entry, "*.")
			if host != base && strings.HasSuffix(host, "."+base) {
				return true
			}
			continue
		}
		if host == entry {
			return true
		}
	}
	return false
}

func GetVideoAsset(c *gin.Context) {
	if !requireVideoAssetGroup(c) {
		return
	}
	assetID := c.Param("asset_id")
	if len(assetID) > 64 || !strings.HasPrefix(assetID, "asset-") || strings.ContainsAny(assetID, "/?#") {
		videoAssetError(c, http.StatusNotFound, "video asset not found")
		return
	}
	asset, exists, err := model.GetVideoAssetForUser(c.GetInt("id"), assetID, videoAssetGroup)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("video asset lookup failed for user %d: %v", c.GetInt("id"), err))
		videoAssetError(c, http.StatusInternalServerError, "failed to query video asset")
		return
	}
	if !exists || asset == nil {
		videoAssetError(c, http.StatusNotFound, "video asset not found")
		return
	}
	channel, err := model.GetChannelById(asset.ChannelID, true)
	if err != nil || channel.Status != common.ChannelStatusEnabled || channel.Type != constant.ChannelTypeServiceInference || channel.ChannelInfo.IsMultiKey {
		videoAssetError(c, http.StatusServiceUnavailable, "video asset service is unavailable")
		return
	}
	client, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
	if err != nil {
		videoAssetError(c, http.StatusBadGateway, "video asset service is unavailable")
		return
	}
	requestContext, cancel := context.WithTimeout(c.Request.Context(), videoAssetRequestWait)
	defer cancel()
	upstream, err := service.RequestVideoAsset(requestContext, client, http.MethodGet, channel.GetBaseURL(), channel.Key, asset.UpstreamID, nil)
	if err == nil && (upstream.Data.ID == "" || strings.TrimSpace(upstream.Data.Status) == "") {
		err = errors.New("upstream response did not include asset identity or status")
	}
	if err != nil {
		logger.LogError(c.Request.Context(), "video asset query failed: "+err.Error())
		videoAssetError(c, http.StatusBadGateway, "failed to query video asset")
		return
	}
	upstream.Data.ID = asset.ID
	upstream.Data.AssetType = asset.AssetType
	upstream.Data.Name = asset.Name
	upstream.Data.URL = asset.SourceURL
	upstream.Data.GroupID = nil
	c.JSON(http.StatusOK, upstream)
}
