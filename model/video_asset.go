package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type VideoAsset struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserID     int    `json:"user_id" gorm:"index:idx_video_asset_owner,priority:1"`
	TokenID    int    `json:"token_id" gorm:"index"`
	Group      string `json:"group" gorm:"type:varchar(64);index:idx_video_asset_owner,priority:2"`
	ChannelID  int    `json:"channel_id" gorm:"index;uniqueIndex:idx_video_asset_upstream,priority:1"`
	UpstreamID string `json:"-" gorm:"type:varchar(191);uniqueIndex:idx_video_asset_upstream,priority:2"`
	AssetType  string `json:"asset_type" gorm:"type:varchar(16)"`
	Name       string `json:"name" gorm:"type:varchar(128)"`
	SourceURL  string `json:"url" gorm:"type:text"`
	CreatedAt  int64  `json:"created_at" gorm:"index"`
	UpdatedAt  int64  `json:"updated_at"`
}

func GenerateVideoAssetID() (string, error) {
	key, err := common.GenerateRandomCharsKey(24)
	if err != nil {
		return "", err
	}
	return "asset-" + key, nil
}

func CreateVideoAsset(asset *VideoAsset) error {
	if asset == nil {
		return errors.New("video asset is nil")
	}
	now := time.Now().Unix()
	if asset.ID == "" {
		generatedID, err := GenerateVideoAssetID()
		if err != nil {
			return err
		}
		asset.ID = generatedID
	}
	if asset.CreatedAt == 0 {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now
	return DB.Create(asset).Error
}

func GetVideoAssetForUser(userID int, assetID string, groups ...string) (*VideoAsset, bool, error) {
	var asset VideoAsset
	query := DB.Where(&VideoAsset{UserID: userID, ID: assetID})
	if len(groups) > 0 && groups[0] != "" {
		query = query.Where(&VideoAsset{Group: groups[0]})
	}
	err := query.First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return &asset, err == nil, err
}

func GetVideoAssetsForUser(userID int, assetIDs []string, groups ...string) (map[string]VideoAsset, error) {
	assetsByID := make(map[string]VideoAsset, len(assetIDs))
	if len(assetIDs) == 0 {
		return assetsByID, nil
	}
	var assets []VideoAsset
	query := DB.Where(&VideoAsset{UserID: userID}).Where("id IN ?", assetIDs)
	if len(groups) > 0 && groups[0] != "" {
		query = query.Where(&VideoAsset{Group: groups[0]})
	}
	if err := query.Find(&assets).Error; err != nil {
		return nil, err
	}
	for _, asset := range assets {
		assetsByID[asset.ID] = asset
	}
	return assetsByID, nil
}

func DeleteVideoAssetsByUser(tx *gorm.DB, userID int) error {
	return tx.Delete(&VideoAsset{}, "user_id = ?", userID).Error
}

func GetEnabledChannelByGroupAndType(group string, channelType int) (*Channel, error) {
	var abilities []Ability
	if err := DB.Model(&Ability{}).
		Where(&Ability{Group: group, Enabled: true}).
		Order("priority DESC").
		Order("weight DESC").
		Order("channel_id ASC").
		Find(&abilities).Error; err != nil {
		return nil, err
	}
	seen := make(map[int]struct{}, len(abilities))
	for _, ability := range abilities {
		if _, exists := seen[ability.ChannelId]; exists {
			continue
		}
		seen[ability.ChannelId] = struct{}{}
		channel, err := GetChannelById(ability.ChannelId, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		if channel.Type == channelType && channel.Status == common.ChannelStatusEnabled && !channel.ChannelInfo.IsMultiKey {
			return channel, nil
		}
	}
	return nil, nil
}
