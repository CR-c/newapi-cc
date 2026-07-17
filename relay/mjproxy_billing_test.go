package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var midjourneyBillingTestDBSequence atomic.Uint64

func TestMidjourneyPreConsumeRejectsConcurrentOverspend(t *testing.T) {
	dbName := fmt.Sprintf("%s_%d", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()), midjourneyBillingTestDBSequence.Add(1))
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=30000", dbName)), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.QuotaLedger{}))

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainDBType, originalLogDBType := common.MainDatabaseType(), common.LogDatabaseType()
	originalRedisEnabled, originalBatchUpdateEnabled := common.RedisEnabled, common.BatchUpdateEnabled
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainDBType, originalLogDBType)
		common.RedisEnabled = originalRedisEnabled
		common.BatchUpdateEnabled = originalBatchUpdateEnabled
		require.NoError(t, sqlDB.Close())
	})
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	user := &model.User{
		Id: 995, Username: "midjourney-concurrent", Status: common.UserStatusEnabled,
		Quota: 100, PaidQuota: 100, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, db.Create(user).Error)

	const callers = 2
	gin.SetMode(gin.TestMode)
	start := make(chan struct{})
	results := make(chan bool, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for index := range callers {
		go func() {
			defer wg.Done()
			<-start
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/mj/submit/imagine", nil)
			info := &relaycommon.RelayInfo{
				RequestId: fmt.Sprintf("mj-concurrent-%d", index), UserId: user.Id, IsPlayground: true,
			}
			info.UserSetting.BillingPreference = "wallet_only"
			results <- preConsumeMidjourneyBilling(ctx, info, 80, false, true) == nil
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successCount := 0
	for success := range results {
		if success {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount)

	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 20, stored.Quota)
	assert.Equal(t, 20, stored.PaidQuota)
	var ledgerCount int64
	require.NoError(t, db.Model(&model.QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Equal(t, int64(1), ledgerCount)
}
