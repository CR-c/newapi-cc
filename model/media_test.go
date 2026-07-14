package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestApplyMediaTypeLogFilterMatchesStableMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Log{}))
	require.NoError(t, db.Create(&[]Log{
		{UserId: 1, Other: `{"media_type":"image"}`},
		{UserId: 1, Other: `{"media_type": "video", "task_id":"task_1"}`},
		{UserId: 1, Other: `{"media_type":"video","note":"image"}`},
		{UserId: 1, Other: `{"media_type":"image","note":"video"}`},
		{UserId: 1, Other: `{"image":true}`},
	}).Error)

	var imageLogs []Log
	require.NoError(t, ApplyMediaTypeLogFilter(db.Model(&Log{}), MediaTypeImage).Find(&imageLogs).Error)
	require.Len(t, imageLogs, 2)
	assert.Contains(t, imageLogs[0].Other, "image")

	var videoLogs []Log
	require.NoError(t, ApplyMediaTypeLogFilter(db.Model(&Log{}), MediaTypeVideo).Find(&videoLogs).Error)
	require.Len(t, videoLogs, 2)
	assert.Contains(t, videoLogs[0].Other, "task_1")
}

func TestGetPlaygroundImageHistoryScopesUserAndLast24Hours(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundMediaHistory{}))

	now := time.Now().Unix()
	require.NoError(t, db.Create(&[]PlaygroundMediaHistory{
		{UserID: 1, MediaType: MediaTypeImage, Model: "image-current", Prompt: "current", Result: `[{"url":"https://example.com/current.png"}]`, CreatedAt: now - 60, ExpiresAt: now + 60},
		{UserID: 1, MediaType: MediaTypeImage, Model: "image-boundary-inside", Prompt: "inside", Result: `[{"url":"https://example.com/inside.png"}]`, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds + 1, ExpiresAt: now + 1},
		{UserID: 1, MediaType: MediaTypeImage, Model: "image-boundary", Prompt: "boundary", Result: `[{"url":"https://example.com/boundary.png"}]`, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds, ExpiresAt: now},
		{UserID: 1, MediaType: MediaTypeImage, Model: "image-expired", Prompt: "expired", Result: `[{"url":"https://example.com/expired.png"}]`, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds - 1, ExpiresAt: now - 1},
		{UserID: 2, MediaType: MediaTypeImage, Model: "image-other-user", Prompt: "private", Result: `[{"url":"https://example.com/private.png"}]`, CreatedAt: now - 30, ExpiresAt: now + 60},
	}).Error)

	items, err := GetPlaygroundImageHistory(db, 1, now, 20)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "image-current", items[0].Model)
	assert.Equal(t, "image-boundary-inside", items[1].Model)
	assert.Equal(t, 1, items[0].UserID)

	var expiredCount int64
	require.NoError(t, db.Model(&PlaygroundMediaHistory{}).Where("expires_at <= ?", now).Count(&expiredCount).Error)
	assert.Zero(t, expiredCount, "expired rows should be removed server-side")
}

func TestGetPlaygroundVideoHistoryScopesUserAndLast24Hours(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Task{}))

	now := time.Now().Unix()
	require.NoError(t, db.Create(&[]Task{
		{TaskID: "task_current", UserId: 1, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - 60, SubmitTime: now - 60, Properties: Properties{IsPlayground: true}},
		{TaskID: "task_boundary_inside", UserId: 1, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds + 1, SubmitTime: now - PlaygroundMediaHistoryTTLSeconds + 1, Properties: Properties{Input: "legacy playground prompt"}},
		{TaskID: "task_api", UserId: 1, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - 30, SubmitTime: now - 30},
		{TaskID: "task_boundary", UserId: 1, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds, SubmitTime: now - PlaygroundMediaHistoryTTLSeconds, Properties: Properties{IsPlayground: true}},
		{TaskID: "task_expired", UserId: 1, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - PlaygroundMediaHistoryTTLSeconds - 1, SubmitTime: now - PlaygroundMediaHistoryTTLSeconds - 1, Properties: Properties{IsPlayground: true}},
		{TaskID: "task_other_user", UserId: 2, Platform: "59", Status: TaskStatusSuccess, CreatedAt: now - 30, SubmitTime: now - 30, Properties: Properties{IsPlayground: true}},
		{TaskID: "task_audio", UserId: 1, Platform: constant.TaskPlatformSuno, Status: TaskStatusSuccess, CreatedAt: now - 30, SubmitTime: now - 30, Properties: Properties{IsPlayground: true}},
	}).Error)

	items, err := GetPlaygroundVideoHistory(db, 1, now, 20)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "task_current", items[0].TaskID)
	assert.Equal(t, "task_boundary_inside", items[1].TaskID)
	assert.Equal(t, 1, items[0].UserId)
}

func TestPlaygroundImageHistoryInsertCapsRowsPerAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&PlaygroundMediaHistory{}))

	now := time.Now()
	for i := 0; i <= PlaygroundMediaHistoryMaxItems; i++ {
		history := NewPlaygroundImageHistory(1, "image", "prompt", `[{"url":"https://example.com/image.png"}]`, now.Add(time.Duration(i)*time.Second))
		require.NoError(t, history.Insert(db))
	}

	var count int64
	require.NoError(t, db.Model(&PlaygroundMediaHistory{}).Where("user_id = ?", 1).Count(&count).Error)
	assert.Equal(t, int64(PlaygroundMediaHistoryMaxItems), count)
}
