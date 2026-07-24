package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUploadedImageStorageDirectoriesUseConfiguredNewImageDirectory(t *testing.T) {
	t.Setenv("PLAYGROUND_ASSET_DIR", "/home/new-image")
	t.Setenv("IMAGE_UPLOAD_DIR", "/home/new-image")

	assert.Equal(t, filepath.Clean("/home/new-image"), PlaygroundAssetStorageDir())
	assert.Equal(t, filepath.Clean("/home/new-image"), UploadedImageStorageDir())
}

func TestPlaygroundAssetQuotaSupportsMultiReferenceVideoWorkflows(t *testing.T) {
	// video-dddd may use up to 9 images + 3 videos + 3 audios per task; users also retry.
	assert.GreaterOrEqual(t, PlaygroundAssetMaxItemsPerUser, 500)
	assert.GreaterOrEqual(t, PlaygroundAssetMaxBytesPerUser, 2<<30)
	assert.GreaterOrEqual(t, PlaygroundAssetMaxBytesGlobal, 40<<30)
}

func TestPlaygroundAssetTTLIsAtMostThirtyMinutes(t *testing.T) {
	assert.Equal(t, 30*60, PlaygroundAssetTTLSeconds)
}

func TestCleanupExpiredPlaygroundAssetsKeepsRecordWhenFileRemovalFails(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", directory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundAsset{}))
	asset := PlaygroundAsset{
		ID: "expired", UserID: 1, Kind: "image", Filename: "expired.png", ContentType: "image/png",
		StorageName: "blocked", Size: 1, CreatedAt: 1, ExpiresAt: 2,
	}
	require.NoError(t, db.Create(&asset).Error)
	blockedPath := PlaygroundAssetStoragePath(asset.StorageName)
	require.NoError(t, os.MkdirAll(blockedPath, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(blockedPath, "child"), []byte("x"), 0o640))

	deleted, err := CleanupExpiredPlaygroundAssets(db, 3, 10)
	require.NoError(t, err)
	assert.Zero(t, deleted)

	var count int64
	require.NoError(t, db.Model(&PlaygroundAsset{}).Where("id = ?", asset.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestCleanupExpiredPlaygroundAssetsDeletesExpiredRowsAndFiles(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", directory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundAsset{}))
	path := filepath.Join(directory, "gone.png")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o640))
	require.NoError(t, db.Create(&PlaygroundAsset{
		ID: "expired-ok", UserID: 1, Kind: "image", Filename: "gone.png", ContentType: "image/png",
		StorageName: "gone.png", Size: 1, CreatedAt: 1, ExpiresAt: 2,
	}).Error)

	deleted, err := CleanupExpiredPlaygroundAssets(db, 3, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
	var count int64
	require.NoError(t, db.Model(&PlaygroundAsset{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCleanupOrphanPlaygroundAssetFilesRemovesUnreferencedOldFiles(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", directory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundAsset{}))
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })

	orphanPath := filepath.Join(directory, "orphan.bin")
	keepPath := filepath.Join(directory, "keep.bin")
	require.NoError(t, os.WriteFile(orphanPath, []byte("orphan"), 0o640))
	require.NoError(t, os.WriteFile(keepPath, []byte("keep"), 0o640))
	// Make orphan look old enough.
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(orphanPath, old, old))
	require.NoError(t, db.Create(&PlaygroundAsset{
		ID: "keep-id", UserID: 1, Kind: "image", Filename: "keep.bin", ContentType: "application/octet-stream",
		StorageName: "keep.bin", Size: 4, CreatedAt: 1, ExpiresAt: time.Now().Unix() + 3600,
	}).Error)

	deleted, err := CleanupOrphanPlaygroundAssetFiles(30*time.Minute, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	_, err = os.Stat(orphanPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(keepPath)
	assert.NoError(t, err)
}

func TestReservePlaygroundAssetStorageAccountsForActiveReservations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundAsset{}, &PlaygroundAssetStorageState{}, &PlaygroundAssetStorageReservation{}))
	now := int64(100)
	require.NoError(t, db.Create(&PlaygroundAsset{
		ID: "existing", UserID: 1, Kind: "image", Filename: "existing.png", ContentType: "image/png",
		StorageName: "existing.png", Size: PlaygroundAssetMaxBytesGlobal - 100, CreatedAt: 1, ExpiresAt: now + 60,
	}).Error)

	reservationID, err := ReservePlaygroundAssetStorage(db, 60, now)
	require.NoError(t, err)
	require.NotEmpty(t, reservationID)
	_, err = ReservePlaygroundAssetStorage(db, 50, now)
	require.ErrorIs(t, err, ErrPlaygroundAssetStorageFull)
	require.NoError(t, ReleasePlaygroundAssetStorage(db, reservationID))
	secondReservationID, err := ReservePlaygroundAssetStorage(db, 50, now)
	require.NoError(t, err)
	require.NoError(t, ReleasePlaygroundAssetStorage(db, secondReservationID))
}
