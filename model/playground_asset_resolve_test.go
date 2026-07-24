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

func TestPlaygroundAssetIDFromURL(t *testing.T) {
	assert.Equal(t, "abc123", PlaygroundAssetIDFromURL("https://vp.example/pg/assets/abc123/file.png"))
	assert.Equal(t, "abc123", PlaygroundAssetIDFromURL("https://vp.example/pg/assets/abc123/%E4%B8%AD%E6%96%87.png"))
	assert.Empty(t, PlaygroundAssetIDFromURL("https://cdn.example/other.png"))
	assert.Empty(t, PlaygroundAssetIDFromURL("asset://asset-local"))
}

func TestResolvePlaygroundAssetIDsFromMediaRefsUsesVideoAssetSourceURL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&VideoAsset{}))
	previous := DB
	DB = db
	t.Cleanup(func() { DB = previous })

	require.NoError(t, db.Create(&VideoAsset{
		ID: "asset-local-1", UserID: 30, Group: "video-dddd", ChannelID: 4,
		UpstreamID: "upstream-1", AssetType: "Image", Name: "ref",
		SourceURL: "https://vp.example/pg/assets/pgAssetOne/ref.png",
	}).Error)

	ids := ResolvePlaygroundAssetIDsFromMediaRefs(30, "video-dddd", []string{
		"asset://asset-local-1",
		"https://vp.example/pg/assets/pgAssetTwo/other.png",
		"https://cdn.example/ignore.png",
	})
	assert.ElementsMatch(t, []string{"pgAssetOne", "pgAssetTwo"}, ids)
}

func TestDeletePlaygroundAssetsByIDsOnlyRemovesOwnedFiles(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("PLAYGROUND_ASSET_DIR", directory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundAsset{}))

	ownedPath := filepath.Join(directory, "owned.png")
	foreignPath := filepath.Join(directory, "foreign.png")
	require.NoError(t, os.WriteFile(ownedPath, []byte("owned"), 0o640))
	require.NoError(t, os.WriteFile(foreignPath, []byte("foreign"), 0o640))
	require.NoError(t, db.Create(&PlaygroundAsset{
		ID: "owned-id", UserID: 30, Kind: "image", Filename: "owned.png",
		ContentType: "image/png", StorageName: "owned.png", Size: 5, CreatedAt: 1, ExpiresAt: 999,
	}).Error)
	require.NoError(t, db.Create(&PlaygroundAsset{
		ID: "foreign-id", UserID: 99, Kind: "image", Filename: "foreign.png",
		ContentType: "image/png", StorageName: "foreign.png", Size: 7, CreatedAt: 1, ExpiresAt: 999,
	}).Error)

	deleted, err := DeletePlaygroundAssetsByIDs(db, 30, []string{"owned-id", "foreign-id", "missing"})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	_, err = os.Stat(ownedPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(foreignPath)
	assert.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&PlaygroundAsset{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestIsPlaygroundAssetQuotaUnlimitedIncludesVP002(t *testing.T) {
	// Without user cache the helper returns false; with env username match after cache is harder
	// without full redis. Built-in map must at least contain vp002 for ops clarity.
	loadPlaygroundAssetUnlimitedUsernames()
	_, ok := playgroundAssetUnlimitedUsernames["vp002"]
	assert.True(t, ok)
}
