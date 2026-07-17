package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMidjourneyFailedZeroChargeTaskDoesNotRefundWallet(t *testing.T) {
	response := []dto.MidjourneyDto{{
		MjId: "mj-zero-charge", Status: "FAILURE", Progress: "100%", FailReason: "upstream failed",
	}}
	encoded, err := common.Marshal(response)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Midjourney{}, &model.QuotaLedger{}, &model.Log{}))
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainDBType, originalLogDBType := common.MainDatabaseType(), common.LogDatabaseType()
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainDBType, originalLogDBType)
		common.RedisEnabled = originalRedisEnabled
	})
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	service.InitHttpClient()

	baseURL := server.URL
	channel := &model.Channel{Id: 994, Name: "midjourney-test", Key: "secret", BaseURL: &baseURL, Status: common.ChannelStatusEnabled}
	user := &model.User{
		Id: 994, Username: "midjourney-zero-charge", Status: common.UserStatusEnabled,
		Quota: 100, PaidQuota: 100, WalletVersion: model.CurrentWalletVersion,
	}
	task := &model.Midjourney{
		UserId: user.Id, MjId: "mj-zero-charge", ChannelId: channel.Id,
		Status: "SUBMITTED", Progress: "0%", SubmitTime: time.Now().UnixMilli(), Quota: 0,
	}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(task).Error)

	summary := runMidjourneyTaskUpdateOnce(context.Background(), nil)
	assert.Equal(t, 1, summary.UnfinishedTasks)
	require.NoError(t, db.First(task, task.Id).Error)
	assert.Equal(t, "FAILURE", task.Status)
	assert.Zero(t, task.Quota)

	var storedUser model.User
	require.NoError(t, db.First(&storedUser, user.Id).Error)
	assert.Equal(t, 100, storedUser.Quota)
	assert.Equal(t, 100, storedUser.PaidQuota)
	var ledgerCount int64
	require.NoError(t, db.Model(&model.QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}
