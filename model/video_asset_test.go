package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVideoAssetOwnershipAndReferenceLookup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&VideoAsset{}))

	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	now := time.Now().Unix()
	asset := &VideoAsset{
		ID: "asset_local", UserID: 7, ChannelID: 59, UpstreamID: "asset-upstream",
		AssetType: "Image", Name: "avatar_front", CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, CreateVideoAsset(asset))

	owned, exists, err := GetVideoAssetForUser(7, asset.ID)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, "asset-upstream", owned.UpstreamID)

	_, exists, err = GetVideoAssetForUser(8, asset.ID)
	require.NoError(t, err)
	assert.False(t, exists)

	references, err := GetVideoAssetsForUser(7, []string{asset.ID})
	require.NoError(t, err)
	require.Len(t, references, 1)
	assert.Equal(t, 59, references[asset.ID].ChannelID)
}
