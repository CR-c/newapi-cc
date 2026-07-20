package middleware

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func resolveVideoAssetReferences(c *gin.Context, usingGroup string) (int, error) {
	if c.Request.Method != http.MethodPost || c.Request.URL.Path != "/v1/videos" {
		return 0, nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return 0, err
	}
	data, err := storage.Bytes()
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = storage.Seek(0, io.SeekStart)
		c.Request.Body = io.NopCloser(storage)
	}()
	imageCount := int(gjson.GetBytes(data, "images.#").Int())
	if usingGroup == "video-dddd" && imageCount > 4 {
		return 0, errors.New("at most 4 reference images are supported")
	}
	assetIDs := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for index := 0; index < imageCount; index++ {
		imageResult := gjson.GetBytes(data, "images."+strconv.Itoa(index))
		if imageResult.Type != gjson.String {
			return 0, fmt.Errorf("images[%d] must be a string", index)
		}
		image := imageResult.String()
		if !strings.HasPrefix(image, "asset://") {
			continue
		}
		if usingGroup != "video-dddd" {
			return 0, errors.New("video asset references are only available to the video-dddd group")
		}
		assetID := strings.TrimPrefix(image, "asset://")
		if len(assetID) > 64 || assetID == "" || strings.ContainsAny(assetID, "/?#") || !strings.HasPrefix(assetID, "asset-") {
			return 0, errors.New("video asset reference is invalid")
		}
		if _, exists := seen[assetID]; !exists {
			seen[assetID] = struct{}{}
			assetIDs = append(assetIDs, assetID)
		}
	}
	if len(assetIDs) == 0 {
		return 0, nil
	}
	assets, err := model.GetVideoAssetsForUser(c.GetInt("id"), assetIDs, usingGroup)
	if err != nil {
		return 0, errors.New("video asset lookup failed")
	}
	if len(assets) != len(assetIDs) {
		return 0, errors.New("video asset not found or unavailable")
	}
	channelID := 0
	references := make(map[string]string, len(assets))
	for _, assetID := range assetIDs {
		asset := assets[assetID]
		if asset.AssetType != "Image" {
			return 0, errors.New("video-dddd currently supports Image assets only")
		}
		if channelID == 0 {
			channelID = asset.ChannelID
		} else if channelID != asset.ChannelID {
			return 0, errors.New("all video assets in one request must belong to the same channel")
		}
		references[assetID] = asset.UpstreamID
	}
	common.SetContextKey(c, constant.ContextKeyVideoAssetChannelId, channelID)
	common.SetContextKey(c, constant.ContextKeyVideoAssetReferences, references)
	return channelID, nil
}

func resolveVideoAssetChannel(c *gin.Context, usingGroup, modelName string) (*model.Channel, error) {
	channelID, err := resolveVideoAssetReferences(c, usingGroup)
	if err != nil || channelID == 0 {
		return nil, err
	}
	channel, err := model.CacheGetChannel(channelID)
	if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
		return nil, errors.New("video asset channel is unavailable")
	}
	if channel.Type != constant.ChannelTypeServiceInference || channel.ChannelInfo.IsMultiKey {
		return nil, errors.New("video asset channel is incompatible")
	}
	if !model.IsChannelEnabledForGroupModel(usingGroup, modelName, channelID) {
		return nil, errors.New("video asset channel cannot serve the requested model")
	}
	return channel, nil
}
