package model

import (
	"testing"

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
		{UserId: 1, Other: `{"image":true}`},
	}).Error)

	var imageLogs []Log
	require.NoError(t, ApplyMediaTypeLogFilter(db.Model(&Log{}), MediaTypeImage).Find(&imageLogs).Error)
	require.Len(t, imageLogs, 1)
	assert.Contains(t, imageLogs[0].Other, "image")

	var videoLogs []Log
	require.NoError(t, ApplyMediaTypeLogFilter(db.Model(&Log{}), MediaTypeVideo).Find(&videoLogs).Error)
	require.Len(t, videoLogs, 1)
	assert.Contains(t, videoLogs[0].Other, "task_1")
}
