package model

import (
	"os"
	"path/filepath"
	"testing"

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

	require.NoError(t, CleanupExpiredPlaygroundAssets(db, 3, 10))

	var count int64
	require.NoError(t, db.Model(&PlaygroundAsset{}).Where("id = ?", asset.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
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
