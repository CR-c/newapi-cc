package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInitChannelCachePreservesExistingCacheWhenDatabaseReadFails(t *testing.T) {
	originalDB := DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalGroupCache := group2model2channels
	originalChannelCache := channelsIDM
	originalRouteCache := channel2advancedCustomConfig
	t.Cleanup(func() {
		DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		group2model2channels = originalGroupCache
		channelsIDM = originalChannelCache
		channel2advancedCustomConfig = originalRouteCache
	})

	db, err := gorm.Open(sqlite.Open("file:channel_cache_failure?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	DB = db
	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"video-dddd": {"dreamina-seedance-2-0-hc": {59}},
	}
	channelsIDM = map[int]*Channel{59: {Id: 59, Name: "existing-video-channel"}}
	channel2advancedCustomConfig = map[int]*dto.AdvancedCustomConfig{59: {
		Routes: []dto.AdvancedCustomRoute{{IncomingPath: "/v1/videos"}},
	}}

	InitChannelCache()

	assert.Equal(t, []int{59}, group2model2channels["video-dddd"]["dreamina-seedance-2-0-hc"])
	require.Contains(t, channelsIDM, 59)
	assert.Equal(t, "existing-video-channel", channelsIDM[59].Name)
	require.Contains(t, channel2advancedCustomConfig, 59)
	assert.True(t, channel2advancedCustomConfig[59].SupportsPath("/v1/videos"))
}

func TestInitChannelCachePreservesExistingCacheWhenAbilitiesReadFails(t *testing.T) {
	originalDB := DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalGroupCache := group2model2channels
	originalChannelCache := channelsIDM
	originalRouteCache := channel2advancedCustomConfig
	t.Cleanup(func() {
		DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		group2model2channels = originalGroupCache
		channelsIDM = originalChannelCache
		channel2advancedCustomConfig = originalRouteCache
	})

	db, err := gorm.Open(sqlite.Open("file:channel_cache_abilities_failure?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}))
	require.NoError(t, db.Create(&Channel{Id: 60, Name: "new-channel", Key: "test-key"}).Error)

	DB = db
	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"sd-video": {"videos": {6}},
	}
	channelsIDM = map[int]*Channel{6: {Id: 6, Name: "existing-kyy-channel"}}
	channel2advancedCustomConfig = map[int]*dto.AdvancedCustomConfig{6: {
		Routes: []dto.AdvancedCustomRoute{{IncomingPath: "/v1/videos"}},
	}}

	InitChannelCache()

	assert.Equal(t, []int{6}, group2model2channels["sd-video"]["videos"])
	require.Contains(t, channelsIDM, 6)
	assert.Equal(t, "existing-kyy-channel", channelsIDM[6].Name)
	require.Contains(t, channel2advancedCustomConfig, 6)
	assert.True(t, channel2advancedCustomConfig[6].SupportsPath("/v1/videos"))
}
