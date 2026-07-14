package model

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	PlaygroundAssetTTLSeconds      = 24 * 60 * 60
	PlaygroundAssetMaxItemsPerUser = 100
	PlaygroundAssetMaxBytesPerUser = 512 << 20
)

type PlaygroundAsset struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserID      int    `json:"user_id" gorm:"index"`
	Kind        string `json:"kind" gorm:"type:varchar(16);index"`
	Filename    string `json:"filename" gorm:"type:varchar(255)"`
	ContentType string `json:"content_type" gorm:"type:varchar(128)"`
	StorageName string `json:"-" gorm:"type:varchar(128);uniqueIndex"`
	Size        int64  `json:"size"`
	CreatedAt   int64  `json:"created_at" gorm:"index"`
	ExpiresAt   int64  `json:"expires_at" gorm:"index"`
}

func PlaygroundAssetStorageDir() string {
	if value := os.Getenv("PLAYGROUND_ASSET_DIR"); value != "" {
		return value
	}
	return filepath.Join(".", "playground-assets")
}

func PlaygroundAssetStoragePath(storageName string) string {
	return filepath.Join(PlaygroundAssetStorageDir(), filepath.Base(storageName))
}

func GetPlaygroundAsset(db *gorm.DB, id string, now int64) (*PlaygroundAsset, bool, error) {
	var asset PlaygroundAsset
	err := db.Where("id = ? AND expires_at > ?", id, now).First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return &asset, err == nil, err
}

func GetPlaygroundAssetUsage(db *gorm.DB, userID int, now int64) (int64, int64, error) {
	var result struct {
		Count int64
		Bytes int64
	}
	err := db.Model(&PlaygroundAsset{}).
		Select("COUNT(*) AS count, COALESCE(SUM(size), 0) AS bytes").
		Where("user_id = ? AND expires_at > ?", userID, now).
		Scan(&result).Error
	return result.Count, result.Bytes, err
}

func CleanupExpiredPlaygroundAssets(db *gorm.DB, now int64, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var assets []PlaygroundAsset
	if err := db.Where("expires_at <= ?", now).Order("expires_at").Limit(limit).Find(&assets).Error; err != nil {
		return err
	}
	if len(assets) == 0 {
		return nil
	}
	ids := make([]string, 0, len(assets))
	for _, asset := range assets {
		ids = append(ids, asset.ID)
		if err := os.Remove(PlaygroundAssetStoragePath(asset.StorageName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			common.SysError("remove expired playground asset error: " + err.Error())
		}
	}
	return db.Delete(&PlaygroundAsset{}, "id IN ?", ids).Error
}

func DeletePlaygroundAssetsByUser(tx *gorm.DB, userID int) error {
	var assets []PlaygroundAsset
	if err := tx.Where("user_id = ?", userID).Find(&assets).Error; err != nil {
		return err
	}
	for _, asset := range assets {
		if err := os.Remove(PlaygroundAssetStoragePath(asset.StorageName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			common.SysError("remove user playground asset error: " + err.Error())
		}
	}
	return tx.Delete(&PlaygroundAsset{}, "user_id = ?", userID).Error
}
