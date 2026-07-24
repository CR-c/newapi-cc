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
	type mediaField struct {
		Name      string
		AssetType string
		Max       int
	}
	mediaFields := []mediaField{
		{Name: "images", AssetType: "Image", Max: 9},
		{Name: "videos", AssetType: "Video", Max: 3},
		{Name: "audios", AssetType: "Audio", Max: 3},
	}
	assetIDs := make([]string, 0, 9)
	assetTypes := make(map[string]string, 9)
	seen := make(map[string]struct{}, 9)
	for _, field := range mediaFields {
		count := int(gjson.GetBytes(data, field.Name+".#").Int())
		if constant.IsVideoAssetGroup(usingGroup) && count > field.Max {
			return 0, fmt.Errorf("at most %d reference %s are supported", field.Max, field.Name)
		}
		for index := 0; index < count; index++ {
			mediaResult := gjson.GetBytes(data, field.Name+"."+strconv.Itoa(index))
			if mediaResult.Type != gjson.String {
				return 0, fmt.Errorf("%s[%d] must be a string", field.Name, index)
			}
			mediaURL := mediaResult.String()
			if !strings.HasPrefix(mediaURL, "asset://") {
				continue
			}
			if !constant.IsVideoAssetGroup(usingGroup) {
				return 0, errors.New("video asset references are only available to video asset groups")
			}
			assetID := strings.TrimPrefix(mediaURL, "asset://")
			if len(assetID) > 64 || assetID == "" || strings.ContainsAny(assetID, "/?#") || !strings.HasPrefix(assetID, "asset-") {
				return 0, errors.New("video asset reference is invalid")
			}
			if previousType, exists := assetTypes[assetID]; exists && previousType != field.AssetType {
				return 0, errors.New("video asset reference type does not match request field")
			}
			assetTypes[assetID] = field.AssetType
			if _, exists := seen[assetID]; !exists {
				seen[assetID] = struct{}{}
				assetIDs = append(assetIDs, assetID)
			}
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
		expectedType := assetTypes[assetID]
		if asset.AssetType != expectedType {
			return 0, fmt.Errorf("%s references must use %s assets", strings.ToLower(expectedType), expectedType)
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
