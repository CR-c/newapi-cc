package model

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	// Temporary /pg/assets lifetime. Video success also deletes used assets earlier;
	// this TTL is a safety net so leftovers never linger longer than 30 minutes.
	PlaygroundAssetTTLSeconds = 30 * 60
	// Per-user limits cover multi-ref workflows (up to 9 images + 3 videos + 3 audios
	// per task, plus retries) without hitting temporary asset quota exceeded too soon.
	PlaygroundAssetMaxItemsPerUser = 500
	PlaygroundAssetMaxBytesPerUser = 2 << 30 // 2 GiB
	PlaygroundAssetMaxBytesGlobal  = 40 << 30 // 40 GiB site-wide safety cap
)

// playgroundAssetUnlimitedUsernames lists accounts with no per-user temporary asset
// item/byte quota. Global disk safety (PlaygroundAssetMaxBytesGlobal) still applies.
// Extra usernames can be appended via PLAYGROUND_ASSET_UNLIMITED_USERNAMES (comma-separated).
var (
	playgroundAssetUnlimitedUsernames = map[string]struct{}{
		"vp002": {},
	}
	playgroundAssetUnlimitedOnce sync.Once
)

var ErrPlaygroundAssetStorageFull = errors.New("playground asset storage quota exceeded")

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

type PlaygroundAssetStorageState struct {
	ID int `gorm:"primaryKey"`
}

type PlaygroundAssetStorageReservation struct {
	ID        string `gorm:"primaryKey;type:varchar(64)"`
	Bytes     int64
	ExpiresAt int64 `gorm:"index"`
}

func PlaygroundAssetStorageDir() string {
	if value := os.Getenv("PLAYGROUND_ASSET_DIR"); value != "" {
		return value
	}
	return filepath.Join(".", "playground-assets")
}

func UploadedImageStorageDir() string {
	if value := os.Getenv("IMAGE_UPLOAD_DIR"); value != "" {
		return value
	}
	return filepath.Join(".", "uploaded-images")
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

func GetPlaygroundAssetGlobalUsage(db *gorm.DB, now int64) (int64, error) {
	var bytes int64
	err := db.Model(&PlaygroundAsset{}).
		Select("COALESCE(SUM(size), 0)").
		Where("expires_at > ?", now).
		Scan(&bytes).Error
	return bytes, err
}

func ReservePlaygroundAssetStorage(db *gorm.DB, bytes, now int64) (string, error) {
	if bytes <= 0 || bytes > PlaygroundAssetMaxBytesGlobal {
		return "", ErrPlaygroundAssetStorageFull
	}
	reservationID, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return "", err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		state := PlaygroundAssetStorageState{ID: 1}
		if err := tx.FirstOrCreate(&state, PlaygroundAssetStorageState{ID: 1}).Error; err != nil {
			return err
		}
		if err := lockForUpdate(tx).Where("id = ?", 1).First(&state).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at <= ?", now).Delete(&PlaygroundAssetStorageReservation{}).Error; err != nil {
			return err
		}
		assetBytes, err := GetPlaygroundAssetGlobalUsage(tx, now)
		if err != nil {
			return err
		}
		var reservedBytes int64
		if err = tx.Model(&PlaygroundAssetStorageReservation{}).
			Select("COALESCE(SUM(bytes), 0)").
			Where("expires_at > ?", now).
			Scan(&reservedBytes).Error; err != nil {
			return err
		}
		if reservedBytes > PlaygroundAssetMaxBytesGlobal-bytes || assetBytes > PlaygroundAssetMaxBytesGlobal-reservedBytes-bytes {
			return ErrPlaygroundAssetStorageFull
		}
		reservation := PlaygroundAssetStorageReservation{ID: reservationID, Bytes: bytes, ExpiresAt: now + 10*60}
		if err = tx.Create(&reservation).Error; err != nil {
			return fmt.Errorf("create playground asset storage reservation: %w", err)
		}
		return nil
	})
	return reservationID, err
}

func ReleasePlaygroundAssetStorage(db *gorm.DB, reservationID string) error {
	if reservationID == "" {
		return nil
	}
	return db.Delete(&PlaygroundAssetStorageReservation{}, "id = ?", reservationID).Error
}

// CleanupExpiredPlaygroundAssets deletes expired temporary assets (DB rows + files).
// It loops in batches until none remain (or maxRounds) so a large backlog cannot
// linger across multiple 5-minute ticks and fill the disk.
func CleanupExpiredPlaygroundAssets(db *gorm.DB, now int64, limit int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	batchSize := limit * 10
	if batchSize > 2000 {
		batchSize = 2000
	}
	const maxRounds = 20
	totalDeleted := 0
	for round := 0; round < maxRounds; round++ {
		var assets []PlaygroundAsset
		if err := db.Where("expires_at <= ?", now).Order("expires_at, id").Limit(batchSize).Find(&assets).Error; err != nil {
			return totalDeleted, err
		}
		if len(assets) == 0 {
			break
		}
		ids := make([]string, 0, len(assets))
		for _, asset := range assets {
			if err := os.Remove(PlaygroundAssetStoragePath(asset.StorageName)); err != nil && !errors.Is(err, os.ErrNotExist) {
				common.SysError("remove expired playground asset error: " + err.Error())
				continue
			}
			ids = append(ids, asset.ID)
		}
		if len(ids) == 0 {
			break
		}
		if err := db.Delete(&PlaygroundAsset{}, "id IN ?", ids).Error; err != nil {
			return totalDeleted, err
		}
		totalDeleted += len(ids)
		if len(assets) < batchSize {
			break
		}
	}
	return totalDeleted, nil
}

// CleanupOrphanPlaygroundAssetFiles removes on-disk files under the playground asset
// directory that are older than maxAge and no longer referenced by any DB row.
// This catches crash leftovers and prevents silent disk growth.
func CleanupOrphanPlaygroundAssetFiles(maxAge time.Duration, limit int) (int, error) {
	if maxAge <= 0 {
		maxAge = time.Duration(PlaygroundAssetTTLSeconds) * time.Second
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	dir := PlaygroundAssetStorageDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	cutoff := time.Now().Add(-maxAge)
	// Build set of storage names still referenced.
	var storageNames []string
	if DB != nil {
		_ = DB.Model(&PlaygroundAsset{}).Pluck("storage_name", &storageNames).Error
	}
	referenced := make(map[string]struct{}, len(storageNames))
	for _, name := range storageNames {
		referenced[name] = struct{}{}
	}
	deleted := 0
	for _, entry := range entries {
		if deleted >= limit {
			break
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := referenced[name]; ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			common.SysError("remove orphan playground asset file: " + err.Error())
			continue
		}
		deleted++
	}
	return deleted, nil
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

func loadPlaygroundAssetUnlimitedUsernames() {
	playgroundAssetUnlimitedOnce.Do(func() {
		for _, raw := range strings.Split(os.Getenv("PLAYGROUND_ASSET_UNLIMITED_USERNAMES"), ",") {
			name := strings.TrimSpace(raw)
			if name != "" {
				playgroundAssetUnlimitedUsernames[name] = struct{}{}
			}
		}
	})
}

// IsPlaygroundAssetQuotaUnlimited reports whether userID skips per-user temporary
// asset count/byte limits (global storage cap still applies).
func IsPlaygroundAssetQuotaUnlimited(userID int) bool {
	if userID <= 0 {
		return false
	}
	loadPlaygroundAssetUnlimitedUsernames()
	// Prefer direct DB lookup: upload path is not hot enough to require cache,
	// and avoids Redis nil-client panics in unit tests.
	user, err := GetUserById(userID, false)
	if err != nil || user == nil || user.Username == "" {
		return false
	}
	_, ok := playgroundAssetUnlimitedUsernames[user.Username]
	return ok
}

// PlaygroundAssetIDFromURL extracts the local temporary asset id from a /pg/assets URL.
func PlaygroundAssetIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+2 < len(segments); i++ {
		if segments[i] == "pg" && segments[i+1] == "assets" {
			id := segments[i+2]
			if id != "" && !strings.ContainsAny(id, "/?#") {
				return id
			}
		}
	}
	return ""
}

// ResolvePlaygroundAssetIDsFromMediaRefs collects temporary /pg/assets ids referenced
// either by direct HTTPS URLs or by asset:// video-library ids whose SourceURL points
// at a playground asset.
func ResolvePlaygroundAssetIDsFromMediaRefs(userID int, group string, mediaRefs []string) []string {
	if len(mediaRefs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(mediaRefs))
	out := make([]string, 0, len(mediaRefs))
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	videoAssetIDs := make([]string, 0)
	for _, ref := range mediaRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if id := PlaygroundAssetIDFromURL(ref); id != "" {
			add(id)
			continue
		}
		if strings.HasPrefix(ref, "asset://") {
			videoAssetIDs = append(videoAssetIDs, strings.TrimPrefix(ref, "asset://"))
		}
	}
	if len(videoAssetIDs) == 0 {
		return out
	}
	assets, err := GetVideoAssetsForUser(userID, videoAssetIDs, group)
	if err != nil {
		return out
	}
	for _, asset := range assets {
		add(PlaygroundAssetIDFromURL(asset.SourceURL))
	}
	return out
}

// DeletePlaygroundAssetsByIDs removes temporary assets owned by userID (files + rows).
func DeletePlaygroundAssetsByIDs(db *gorm.DB, userID int, ids []string) (int, error) {
	if db == nil || userID <= 0 || len(ids) == 0 {
		return 0, nil
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return 0, nil
	}
	var assets []PlaygroundAsset
	if err := db.Where("user_id = ? AND id IN ?", userID, unique).Find(&assets).Error; err != nil {
		return 0, err
	}
	if len(assets) == 0 {
		return 0, nil
	}
	deletedIDs := make([]string, 0, len(assets))
	for _, asset := range assets {
		if err := os.Remove(PlaygroundAssetStoragePath(asset.StorageName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			common.SysError("remove playground asset after video complete: " + err.Error())
			continue
		}
		deletedIDs = append(deletedIDs, asset.ID)
	}
	if len(deletedIDs) == 0 {
		return 0, nil
	}
	if err := db.Delete(&PlaygroundAsset{}, "user_id = ? AND id IN ?", userID, deletedIDs).Error; err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}
