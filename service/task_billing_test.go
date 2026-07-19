package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.Task{},
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.TopUp{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.SubscriptionPreConsumeRecord{},
		&model.SystemTask{},
		&model.SystemTaskLock{},
		&model.QuotaLedger{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

func truncate(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM tasks")
		model.DB.Exec("DELETE FROM users")
		model.DB.Exec("DELETE FROM tokens")
		model.DB.Exec("DELETE FROM logs")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM top_ups")
		model.DB.Exec("DELETE FROM user_subscriptions")
		model.DB.Exec("DELETE FROM subscription_pre_consume_records")
		model.DB.Exec("DELETE FROM subscription_plans")
		model.DB.Exec("DELETE FROM system_task_locks")
		model.DB.Exec("DELETE FROM system_tasks")
		model.DB.Exec("DELETE FROM quota_ledgers")
	})
}

func TestWalletFundingSettlementPreservesPromoFirstAllocation(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 991, Username: "wallet-funding", Status: common.UserStatusEnabled,
		Quota: 100, PaidQuota: 50, PromoQuota: 50, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)

	funding := &WalletFunding{userId: user.Id, requestId: "req-wallet-funding"}
	require.NoError(t, funding.PreConsume(80))
	assert.Equal(t, model.WalletAllocation{PaidQuota: 30, PromoQuota: 50}, funding.Allocation())

	require.NoError(t, funding.Settle(-20))
	assert.Equal(t, model.WalletAllocation{PaidQuota: 10, PromoQuota: 50}, funding.Allocation())

	var stored model.User
	require.NoError(t, model.DB.First(&stored, user.Id).Error)
	assert.Equal(t, 40, stored.PaidQuota)
	assert.Zero(t, stored.PromoQuota)
	assert.Equal(t, 40, stored.Quota)
}

func TestFailedTaskSettlementRefundsOnlyConfirmedCharge(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 992, Username: "failed-task-settlement", Status: common.UserStatusEnabled,
		Quota: 100, PaidQuota: 100, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)

	funding := &WalletFunding{userId: user.Id, requestId: "req-failed-task-settlement"}
	require.NoError(t, funding.PreConsume(80))
	info := &relaycommon.RelayInfo{UserId: user.Id, IsPlayground: true}
	session := &BillingSession{relayInfo: info, funding: funding, preConsumedQuota: 80}
	info.Billing = session
	session.syncRelayInfo()

	gin.SetMode(gin.TestMode)
	requestContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestContext.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	err := SettleBilling(requestContext, info, 120)
	require.ErrorIs(t, err, model.ErrInsufficientWalletQuota)
	assert.Equal(t, 80, info.FinalPreConsumedQuota)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 80}, funding.Allocation())

	task := makeTask(user.Id, 0, info.FinalPreConsumedQuota, 0, BillingSourceWallet, 0)
	task.PrivateData.WalletAllocation = funding.Allocation()
	RefundTaskQuota(context.Background(), task, "settlement failed")

	assert.Equal(t, 100, getUserQuota(t, user.Id))
	assert.Zero(t, task.PrivateData.WalletAllocation.Total())
	var refundLedger model.QuotaLedger
	require.NoError(t, model.DB.Where("operation = ?", "refund").First(&refundLedger).Error)
	assert.Equal(t, 80, refundLedger.Amount)
}

func TestLegacyPostConsumeRejectsRefundBeyondTrackedAllocation(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 993, Username: "legacy-post-refund", Status: common.UserStatusEnabled,
		Quota: 20, PaidQuota: 20, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)
	info := &relaycommon.RelayInfo{
		RequestId: "req-legacy-post-refund", UserId: user.Id,
		WalletPaidQuota: 30,
	}

	err := PostConsumeQuota(info, -50, 0, false)
	require.ErrorContains(t, err, "refund exceeds consumed allocation")
	assert.Equal(t, 20, getUserQuota(t, user.Id))
	assert.Equal(t, 30, info.WalletPaidQuota)

	var ledgerCount int64
	require.NoError(t, model.DB.Model(&model.QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)
}

func TestBillingSessionTokenFailureKeepsConfirmedFundingAllocation(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 994, Username: "token-adjust-failure", Status: common.UserStatusEnabled,
		Quota: 120, PaidQuota: 120, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)
	funding := &WalletFunding{userId: user.Id, requestId: "req-token-adjust-failure"}
	require.NoError(t, funding.PreConsume(80))

	injectedErr := errors.New("injected token update failure")
	callbackName := "test:fail-token-adjust"
	require.NoError(t, model.DB.Callback().Update().Before("gorm:update").Register(callbackName, func(db *gorm.DB) {
		if db.Statement.Schema != nil && db.Statement.Schema.Name == "Token" {
			db.AddError(injectedErr)
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, model.DB.Callback().Update().Remove(callbackName))
	})

	info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: 77, TokenKey: "sk-fail-token"}
	session := &BillingSession{relayInfo: info, funding: funding, preConsumedQuota: 80, tokenConsumed: 80}
	info.Billing = session
	session.syncRelayInfo()

	err := session.Settle(100)
	require.ErrorIs(t, err, injectedErr)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 100}, funding.Allocation())
	assert.Equal(t, 100, info.WalletPaidQuota+info.WalletPromoQuota+info.WalletLegacyQuota)
	assert.Equal(t, 20, getUserQuota(t, user.Id))

	task := makeTask(user.Id, 0, 100, 0, BillingSourceWallet, 0)
	task.PrivateData.WalletAllocation = funding.Allocation()
	RefundTaskQuota(context.Background(), task, "task failed after token adjustment error")
	assert.Equal(t, 120, getUserQuota(t, user.Id))
}

func TestTaskSubscriptionAdjustmentPreservesMixedFundingAllocation(t *testing.T) {
	truncate(t)
	sub := &model.UserSubscription{
		Id: 990, UserId: 991, AmountTotal: 1_000, AmountUsed: 100,
		FundingPaidPPM: 500_000, FundingPromoPPM: 300_000,
		FundingLegacyPPM: 200_000, FundingVersion: 1,
	}
	require.NoError(t, model.DB.Create(sub).Error)
	task := &model.Task{
		TaskID: "task-subscription-funding", UserId: sub.UserId, Quota: 100,
		PrivateData: model.TaskPrivateData{
			BillingSource: BillingSourceSubscription, SubscriptionId: sub.Id,
			WalletAllocation: model.WalletAllocation{PaidQuota: 50, PromoQuota: 30, LegacyQuota: 20},
		},
	}

	refunded, err := taskAdjustFunding(task, -40)
	require.NoError(t, err)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 20, PromoQuota: 12, LegacyQuota: 8}, refunded)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 30, PromoQuota: 18, LegacyQuota: 12}, task.PrivateData.WalletAllocation)
	assert.Equal(t, int64(60), getSubscriptionUsed(t, sub.Id))
}

func TestPreV2TaskRefundReturnsHistoricalAllocationAsPaid(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 993, Username: "pre-v2-task-refund", Status: common.UserStatusEnabled,
		Quota: 70, PaidQuota: 70, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)
	task := makeTask(user.Id, 0, 30, 0, BillingSourceWallet, 0)
	task.PrivateData.WalletAllocationVersion = 1
	task.PrivateData.WalletAllocation = model.WalletAllocation{PromoQuota: 30}

	refunded, err := taskAdjustFunding(task, -30)
	require.NoError(t, err)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 30}, refunded)
	assert.Zero(t, task.PrivateData.WalletAllocation.Total())

	var stored model.User
	require.NoError(t, model.DB.First(&stored, user.Id).Error)
	assert.Equal(t, 100, stored.Quota)
	assert.Equal(t, 100, stored.PaidQuota)
	assert.Zero(t, stored.PromoQuota)
	assert.Zero(t, stored.LegacyUnknownQuota)
}

func TestPreV2TaskNormalizesBeforePostV2DebitAndRefund(t *testing.T) {
	truncate(t)
	user := &model.User{
		Id: 994, Username: "mixed-version-task", Status: common.UserStatusEnabled,
		Quota: 20, PromoQuota: 20, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, model.DB.Create(user).Error)
	task := makeTask(user.Id, 0, 30, 0, BillingSourceWallet, 0)
	task.PrivateData.WalletAllocationVersion = 1
	task.PrivateData.WalletAllocation = model.WalletAllocation{PromoQuota: 30}

	debited, err := taskAdjustFunding(task, 20)
	require.NoError(t, err)
	assert.Equal(t, model.WalletAllocation{PromoQuota: 20}, debited)
	assert.Equal(t, model.CurrentWalletVersion, task.PrivateData.WalletAllocationVersion)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 30, PromoQuota: 20}, task.PrivateData.WalletAllocation)
	task.Quota = 50

	refunded, err := taskAdjustFunding(task, -50)
	require.NoError(t, err)
	assert.Equal(t, model.WalletAllocation{PaidQuota: 30, PromoQuota: 20}, refunded)
	assert.Zero(t, task.PrivateData.WalletAllocation.Total())

	var stored model.User
	require.NoError(t, model.DB.First(&stored, user.Id).Error)
	assert.Equal(t, 50, stored.Quota)
	assert.Equal(t, 30, stored.PaidQuota)
	assert.Equal(t, 20, stored.PromoQuota)
	assert.Zero(t, stored.LegacyUnknownQuota)
}

func TestAppendBillingInfoIncludesSubscriptionFundingAndAdminUsage(t *testing.T) {
	info := &relaycommon.RelayInfo{
		BillingSource:   BillingSourceSubscription,
		WalletPaidQuota: 20, WalletPromoQuota: 30, WalletLegacyQuota: 10,
		UserRole: common.RoleAdminUser, IsAdminUsage: true,
	}
	other := map[string]interface{}{}

	appendBillingInfo(info, other)

	assert.Equal(t, BillingSourceSubscription, other["billing_source"])
	assert.Equal(t, common.RoleAdminUser, other["user_role_snapshot"])
	assert.Equal(t, true, other["is_admin_usage"])
	assert.Equal(t, map[string]interface{}{
		"version": 1, "paid_quota": 20, "promo_quota": 30, "legacy_quota": 10,
	}, other["wallet_funding"])
}

func seedUser(t *testing.T, id int, quota int) {
	t.Helper()
	user := &model.User{Id: id, Username: "test_user", Quota: quota, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)
}

func seedToken(t *testing.T, id int, userId int, key string, remainQuota int) {
	t.Helper()
	token := &model.Token{
		Id:          id,
		UserId:      userId,
		Key:         key,
		Name:        "test_token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: remainQuota,
		UsedQuota:   0,
	}
	require.NoError(t, model.DB.Create(token).Error)
}

func seedSubscription(t *testing.T, id int, userId int, amountTotal int64, amountUsed int64) {
	t.Helper()
	sub := &model.UserSubscription{
		Id:          id,
		UserId:      userId,
		AmountTotal: amountTotal,
		AmountUsed:  amountUsed,
		Status:      "active",
		StartTime:   time.Now().Unix(),
		EndTime:     time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	require.NoError(t, model.DB.Create(sub).Error)
}

func seedChannel(t *testing.T, id int) {
	t.Helper()
	ch := &model.Channel{Id: id, Name: "test_channel", Key: "sk-test", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(ch).Error)
}

func makeTask(userId, channelId, quota, tokenId int, billingSource string, subscriptionId int) *model.Task {
	return &model.Task{
		TaskID:    "task_" + time.Now().Format("150405.000"),
		UserId:    userId,
		ChannelId: channelId,
		Quota:     quota,
		Status:    model.TaskStatus(model.TaskStatusInProgress),
		Group:     "default",
		Data:      json.RawMessage(`{}`),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Properties: model.Properties{
			OriginModelName: "test-model",
		},
		PrivateData: model.TaskPrivateData{
			BillingSource:           billingSource,
			WalletAllocationVersion: model.CurrentWalletVersion,
			SubscriptionId:          subscriptionId,
			TokenId:                 tokenId,
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.02,
				GroupRatio:      1.0,
				OriginModelName: "test-model",
			},
		},
	}
}

func TestPriceDataOtherRatiosFilterAndSnapshot(t *testing.T) {
	priceData := types.PriceData{}

	priceData.AddOtherRatio("zero", 0)
	priceData.AddOtherRatio("negative", -0.5)
	priceData.AddOtherRatio("nan", math.NaN())
	priceData.AddOtherRatio("inf", math.Inf(1))
	priceData.AddOtherRatio("one", 1)
	priceData.AddOtherRatio("positive", 2.5)

	ratios := priceData.OtherRatios()
	require.Len(t, ratios, 2)
	assert.Equal(t, 1.0, ratios["one"])
	assert.Equal(t, 2.5, ratios["positive"])
	assert.True(t, priceData.HasOtherRatio("one"))
	assert.False(t, priceData.HasOtherRatio("zero"))

	ratios["positive"] = 99
	ratios["new"] = 3
	nextSnapshot := priceData.OtherRatios()
	assert.Equal(t, 2.5, nextSnapshot["positive"])
	assert.NotContains(t, nextSnapshot, "new")
}

func TestPriceDataReplaceAndApplyOtherRatios(t *testing.T) {
	priceData := types.PriceData{}

	replaced := priceData.ReplaceOtherRatios(map[string]float64{
		"zero":     0,
		"negative": -3,
		"nan":      math.NaN(),
		"inf":      math.Inf(1),
		"one":      1,
		"duration": 2,
		"size":     1.5,
	})

	require.True(t, replaced)
	assert.Equal(t, 3.0, priceData.OtherRatioMultiplier())
	assert.Equal(t, 30.0, priceData.ApplyOtherRatiosToFloat(10))
	assert.Equal(t, 10.0, priceData.RemoveOtherRatiosFromFloat(30))
	assert.True(t, decimal.NewFromInt(30).Equal(priceData.ApplyOtherRatiosToDecimal(decimal.NewFromInt(10))))

	replaced = priceData.ReplaceOtherRatios(map[string]float64{
		"zero": 0,
		"nan":  math.NaN(),
	})

	require.False(t, replaced)
	assert.Nil(t, priceData.OtherRatios())
	assert.Equal(t, 1.0, priceData.OtherRatioMultiplier())
}

func TestTaskBillingOtherFiltersHistoricalOtherRatios(t *testing.T) {
	task := makeTask(1, 1, 100, 0, BillingSourceWallet, 0)
	profitGeneration := int64(3)
	task.PrivateData.BillingContext.ProfitGeneration = &profitGeneration
	task.PrivateData.BillingContext.OtherRatios = map[string]float64{
		"seconds":  2,
		"identity": 1,
		"zero":     0,
		"negative": -1,
		"nan":      math.NaN(),
		"inf":      math.Inf(1),
	}

	other := taskBillingOther(task)

	assert.Equal(t, 2.0, other["seconds"])
	assert.Equal(t, 1.0, other["identity"])
	assert.NotContains(t, other, "zero")
	assert.NotContains(t, other, "negative")
	assert.NotContains(t, other, "nan")
	assert.NotContains(t, other, "inf")
	assert.Equal(t, "video", other["media_type"])
	assert.Equal(t, task.TaskID, other["task_id"])
	assert.Equal(t, int64(3), other["profit_generation"])
}

func TestLogTaskConsumptionRecordsVideoMetadata(t *testing.T) {
	truncate(t)
	seedUser(t, 1, 1000)
	seedChannel(t, 1)

	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/pg/videos", nil)
	context.Set("token_name", "playground-default")
	profitGeneration := int64(3)

	info := &relaycommon.RelayInfo{
		UserId:           1,
		OriginModelName:  "sora-2",
		UsingGroup:       "default",
		ProfitGeneration: &profitGeneration,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 1,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_media_log",
			Action:       constant.TaskActionTextGenerate,
		},
	}

	LogTaskConsumption(context, info)

	var log model.Log
	require.NoError(t, model.DB.Where("user_id = ?", 1).First(&log).Error)
	var other map[string]any
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	assert.Equal(t, "video", other["media_type"])
	assert.Equal(t, "task_media_log", other["task_id"])
	assert.Equal(t, float64(3), other["profit_generation"])
}

func TestTaskBillingContextPriceDataFiltersMultiplier(t *testing.T) {
	priceData := taskBillingContextPriceData(&model.TaskBillingContext{
		OtherRatios: map[string]float64{
			"seconds":  2,
			"size":     3,
			"identity": 1,
			"zero":     0,
			"negative": -1,
			"nan":      math.NaN(),
			"inf":      math.Inf(1),
		},
	})

	require.NotNil(t, priceData)
	assert.Equal(t, 6.0, priceData.OtherRatioMultiplier())
	assert.Equal(t, map[string]float64{
		"seconds":  2,
		"size":     3,
		"identity": 1,
	}, priceData.OtherRatios())
}

// ---------------------------------------------------------------------------
// Read-back helpers
// ---------------------------------------------------------------------------

func getUserQuota(t *testing.T, id int) int {
	t.Helper()
	var user model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", id).First(&user).Error)
	return user.Quota
}

func getTokenRemainQuota(t *testing.T, id int) int {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.Select("remain_quota").Where("id = ?", id).First(&token).Error)
	return token.RemainQuota
}

func getTokenUsedQuota(t *testing.T, id int) int {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.Select("used_quota").Where("id = ?", id).First(&token).Error)
	return token.UsedQuota
}

func getSubscriptionUsed(t *testing.T, id int) int64 {
	t.Helper()
	var sub model.UserSubscription
	require.NoError(t, model.DB.Select("amount_used").Where("id = ?", id).First(&sub).Error)
	return sub.AmountUsed
}

func getLastLog(t *testing.T) *model.Log {
	t.Helper()
	var log model.Log
	err := model.LOG_DB.Order("id desc").First(&log).Error
	if err != nil {
		return nil
	}
	return &log
}

func countLogs(t *testing.T) int64 {
	t.Helper()
	var count int64
	model.LOG_DB.Model(&model.Log{}).Count(&count)
	return count
}

// ===========================================================================
// RefundTaskQuota tests
// ===========================================================================

func TestRefundTaskQuota_Wallet(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 1, 1, 1
	const initQuota, preConsumed = 10000, 3000
	const tokenRemain = 5000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-test-key", tokenRemain)
	seedChannel(t, channelID)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"used_quota":    preConsumed,
		"request_count": 1,
	}).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("used_quota", preConsumed).Error)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)

	RefundTaskQuota(ctx, task, "task failed: upstream error")

	// User quota should increase by preConsumed
	assert.Equal(t, initQuota+preConsumed, getUserQuota(t, userID))

	// Token remain_quota should increase, used_quota should decrease
	assert.Equal(t, tokenRemain+preConsumed, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, -preConsumed, getTokenUsedQuota(t, tokenID))

	// A refund log should be created
	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
	assert.Equal(t, preConsumed, log.Quota)
	assert.Equal(t, "test-model", log.ModelName)

	var user model.User
	require.NoError(t, model.DB.Select("used_quota", "request_count").First(&user, userID).Error)
	assert.Zero(t, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)

	var channel model.Channel
	require.NoError(t, model.DB.Select("used_quota").First(&channel, channelID).Error)
	assert.Zero(t, channel.UsedQuota)
}

func TestRefundTaskQuota_Subscription(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID, subID = 2, 2, 2, 1
	const preConsumed = 2000
	const subTotal, subUsed int64 = 100000, 50000
	const tokenRemain = 8000

	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, "sk-sub-key", tokenRemain)
	seedChannel(t, channelID)
	seedSubscription(t, subID, userID, subTotal, subUsed)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceSubscription, subID)

	RefundTaskQuota(ctx, task, "subscription task failed")

	// Subscription used should decrease by preConsumed
	assert.Equal(t, subUsed-int64(preConsumed), getSubscriptionUsed(t, subID))

	// Token should also be refunded
	assert.Equal(t, tokenRemain+preConsumed, getTokenRemainQuota(t, tokenID))

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}

func TestRefundTaskQuota_ZeroQuota(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID = 3
	seedUser(t, userID, 5000)

	task := makeTask(userID, 0, 0, 0, BillingSourceWallet, 0)

	RefundTaskQuota(ctx, task, "zero quota task")

	// No change to user quota
	assert.Equal(t, 5000, getUserQuota(t, userID))

	// No log created
	assert.Equal(t, int64(0), countLogs(t))
}

func TestRefundTaskQuota_NoToken(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, channelID = 4, 4
	const initQuota, preConsumed = 10000, 1500

	seedUser(t, userID, initQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, 0, BillingSourceWallet, 0) // TokenId=0

	RefundTaskQuota(ctx, task, "no token task failed")

	// User quota refunded
	assert.Equal(t, initQuota+preConsumed, getUserQuota(t, userID))

	// Log created
	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}

// ===========================================================================
// RecalculateTaskQuota tests
// ===========================================================================

func TestRecalculate_PositiveDelta(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 10, 10, 10
	const initQuota, preConsumed = 10000, 2000
	const actualQuota = 3000 // under-charged by 1000
	const tokenRemain = 5000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-recalc-pos", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)

	RecalculateTaskQuota(ctx, task, actualQuota, "adaptor adjustment")

	// User quota should decrease by the delta (1000 additional charge)
	assert.Equal(t, initQuota-(actualQuota-preConsumed), getUserQuota(t, userID))

	// Token should also be charged the delta
	assert.Equal(t, tokenRemain-(actualQuota-preConsumed), getTokenRemainQuota(t, tokenID))

	// task.Quota should be updated to actualQuota
	assert.Equal(t, actualQuota, task.Quota)

	// Log type should be Consume (additional charge)
	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, actualQuota-preConsumed, log.Quota)
}

func TestRecalculate_NegativeDelta(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 11, 11, 11
	const initQuota, preConsumed = 10000, 5000
	const actualQuota = 3000 // over-charged by 2000
	const tokenRemain = 5000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-recalc-neg", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"used_quota":    preConsumed,
		"request_count": 1,
	}).Error)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("used_quota", preConsumed).Error)

	RecalculateTaskQuota(ctx, task, actualQuota, "adaptor adjustment")

	// User quota should increase by abs(delta) = 2000 (refund overpayment)
	assert.Equal(t, initQuota+(preConsumed-actualQuota), getUserQuota(t, userID))

	// Token should be refunded the difference
	assert.Equal(t, tokenRemain+(preConsumed-actualQuota), getTokenRemainQuota(t, tokenID))

	// task.Quota updated
	assert.Equal(t, actualQuota, task.Quota)

	// Log type should be Refund
	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
	assert.Equal(t, preConsumed-actualQuota, log.Quota)

	var user model.User
	require.NoError(t, model.DB.Select("used_quota", "request_count").First(&user, userID).Error)
	assert.Equal(t, actualQuota, user.UsedQuota)
	assert.Equal(t, 1, user.RequestCount)

	var channel model.Channel
	require.NoError(t, model.DB.Select("used_quota").First(&channel, channelID).Error)
	assert.EqualValues(t, actualQuota, channel.UsedQuota)
}

func TestRecalculate_ZeroDelta(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID = 12
	const initQuota, preConsumed = 10000, 3000

	seedUser(t, userID, initQuota)

	task := makeTask(userID, 0, preConsumed, 0, BillingSourceWallet, 0)

	RecalculateTaskQuota(ctx, task, preConsumed, "exact match")

	// No change to user quota
	assert.Equal(t, initQuota, getUserQuota(t, userID))

	// No log created (delta is zero)
	assert.Equal(t, int64(0), countLogs(t))
}

func TestRecalculate_ActualQuotaZero(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID = 13
	const initQuota = 10000

	seedUser(t, userID, initQuota)

	task := makeTask(userID, 0, 5000, 0, BillingSourceWallet, 0)

	RecalculateTaskQuota(ctx, task, 0, "zero actual")

	// No change (early return)
	assert.Equal(t, initQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(0), countLogs(t))
}

func TestRecalculate_Subscription_NegativeDelta(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID, subID = 14, 14, 14, 2
	const preConsumed = 5000
	const actualQuota = 2000 // over-charged by 3000
	const subTotal, subUsed int64 = 100000, 50000
	const tokenRemain = 8000

	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, "sk-sub-recalc", tokenRemain)
	seedChannel(t, channelID)
	seedSubscription(t, subID, userID, subTotal, subUsed)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceSubscription, subID)

	RecalculateTaskQuota(ctx, task, actualQuota, "subscription over-charge")

	// Subscription used should decrease by delta (refund 3000)
	assert.Equal(t, subUsed-int64(preConsumed-actualQuota), getSubscriptionUsed(t, subID))

	// Token refunded
	assert.Equal(t, tokenRemain+(preConsumed-actualQuota), getTokenRemainQuota(t, tokenID))

	assert.Equal(t, actualQuota, task.Quota)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}

// ===========================================================================
// CAS + Billing integration tests
// Simulates the flow in updateVideoSingleTask (service/task_polling.go)
// ===========================================================================

// simulatePollBilling reproduces the CAS + billing logic from updateVideoSingleTask.
// It takes a persisted task (already in DB), applies the new status, and performs
// the conditional update + billing exactly as the polling loop does.
func simulatePollBilling(ctx context.Context, task *model.Task, newStatus model.TaskStatus, actualQuota int) {
	snap := task.Snapshot()

	shouldRefund := false
	shouldSettle := false
	quota := task.Quota

	task.Status = newStatus
	switch string(newStatus) {
	case model.TaskStatusSuccess:
		task.Progress = "100%"
		task.FinishTime = 9999
		shouldSettle = true
	case model.TaskStatusFailure:
		task.Progress = "100%"
		task.FinishTime = 9999
		task.FailReason = "upstream error"
		if quota != 0 {
			shouldRefund = true
		}
	default:
		task.Progress = "50%"
	}

	isDone := task.Status == model.TaskStatus(model.TaskStatusSuccess) || task.Status == model.TaskStatus(model.TaskStatusFailure)
	if isDone && snap.Status != task.Status {
		won, err := task.UpdateWithStatus(snap.Status)
		if err != nil {
			shouldRefund = false
			shouldSettle = false
		} else if !won {
			shouldRefund = false
			shouldSettle = false
		}
	} else if !snap.Equal(task.Snapshot()) {
		_, _ = task.UpdateWithStatus(snap.Status)
	}

	if shouldSettle && actualQuota > 0 {
		RecalculateTaskQuota(ctx, task, actualQuota, "test settle")
	}
	if shouldRefund {
		RefundTaskQuota(ctx, task, task.FailReason)
	}
}

func TestCASGuardedRefund_Win(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 20, 20, 20
	const initQuota, preConsumed = 10000, 4000
	const tokenRemain = 6000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-cas-refund-win", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatus(model.TaskStatusInProgress)
	require.NoError(t, model.DB.Create(task).Error)

	simulatePollBilling(ctx, task, model.TaskStatus(model.TaskStatusFailure), 0)

	// CAS wins: task in DB should now be FAILURE
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)

	// Refund should have happened
	assert.Equal(t, initQuota+preConsumed, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+preConsumed, getTokenRemainQuota(t, tokenID))

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}

func TestCASGuardedRefund_Lose(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 21, 21, 21
	const initQuota, preConsumed = 10000, 4000
	const tokenRemain = 6000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-cas-refund-lose", tokenRemain)
	seedChannel(t, channelID)

	// Create task with IN_PROGRESS in DB
	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatus(model.TaskStatusInProgress)
	require.NoError(t, model.DB.Create(task).Error)

	// Simulate another process already transitioning to FAILURE
	model.DB.Model(&model.Task{}).Where("id = ?", task.ID).Update("status", model.TaskStatusFailure)

	// Our process still has the old in-memory state (IN_PROGRESS) and tries to transition
	// task.Status is still IN_PROGRESS in the snapshot
	simulatePollBilling(ctx, task, model.TaskStatus(model.TaskStatusFailure), 0)

	// CAS lost: user quota should NOT change (no double refund)
	assert.Equal(t, initQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain, getTokenRemainQuota(t, tokenID))

	// No billing log should be created
	assert.Equal(t, int64(0), countLogs(t))
}

func TestCASGuardedSettle_Win(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 22, 22, 22
	const initQuota, preConsumed = 10000, 5000
	const actualQuota = 3000 // over-charged, should get partial refund
	const tokenRemain = 8000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-cas-settle-win", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.Status = model.TaskStatus(model.TaskStatusInProgress)
	require.NoError(t, model.DB.Create(task).Error)

	simulatePollBilling(ctx, task, model.TaskStatus(model.TaskStatusSuccess), actualQuota)

	// CAS wins: task should be SUCCESS
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.EqualValues(t, model.TaskStatusSuccess, reloaded.Status)

	// Settlement should refund the over-charge (5000 - 3000 = 2000 back to user)
	assert.Equal(t, initQuota+(preConsumed-actualQuota), getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+(preConsumed-actualQuota), getTokenRemainQuota(t, tokenID))

	// task.Quota should be updated to actualQuota
	assert.Equal(t, actualQuota, task.Quota)
}

func TestNonTerminalUpdate_NoBilling(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, channelID = 23, 23
	const initQuota, preConsumed = 10000, 3000

	seedUser(t, userID, initQuota)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, 0, BillingSourceWallet, 0)
	task.Status = model.TaskStatus(model.TaskStatusInProgress)
	task.Progress = "20%"
	require.NoError(t, model.DB.Create(task).Error)

	// Simulate a non-terminal poll update (still IN_PROGRESS, progress changed)
	simulatePollBilling(ctx, task, model.TaskStatus(model.TaskStatusInProgress), 0)

	// User quota should NOT change
	assert.Equal(t, initQuota, getUserQuota(t, userID))

	// No billing log
	assert.Equal(t, int64(0), countLogs(t))

	// Task progress should be updated in DB
	var reloaded model.Task
	require.NoError(t, model.DB.First(&reloaded, task.ID).Error)
	assert.Equal(t, "50%", reloaded.Progress)
}

// ===========================================================================
// Mock adaptor for settleTaskBillingOnComplete tests
// ===========================================================================

type mockAdaptor struct {
	adjustReturn int
}

func (m *mockAdaptor) Init(_ *relaycommon.RelayInfo) {}
func (m *mockAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	return nil, nil
}
func (m *mockAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) { return nil, nil }
func (m *mockAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return m.adjustReturn
}

// ===========================================================================
// PerCallBilling tests — settleTaskBillingOnComplete
// ===========================================================================

func TestSettle_PerCallBilling_SkipsAdaptorAdjust(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 30, 30, 30
	const initQuota, preConsumed = 10000, 5000
	const tokenRemain = 8000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-percall-adaptor", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.PrivateData.BillingContext.PerCallBilling = true

	adaptor := &mockAdaptor{adjustReturn: 2000}
	taskResult := &relaycommon.TaskInfo{Status: model.TaskStatusSuccess}

	settleTaskBillingOnComplete(ctx, adaptor, task, taskResult)

	// Per-call: no adjustment despite adaptor returning 2000
	assert.Equal(t, initQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, preConsumed, task.Quota)
	assert.Equal(t, int64(0), countLogs(t))
}

func TestSettle_PerCallBilling_SkipsTotalTokens(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 31, 31, 31
	const initQuota, preConsumed = 10000, 4000
	const tokenRemain = 7000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-percall-tokens", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.PrivateData.BillingContext.PerCallBilling = true

	adaptor := &mockAdaptor{adjustReturn: 0}
	taskResult := &relaycommon.TaskInfo{Status: model.TaskStatusSuccess, TotalTokens: 9999}

	settleTaskBillingOnComplete(ctx, adaptor, task, taskResult)

	// Per-call: no recalculation by tokens
	assert.Equal(t, initQuota, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain, getTokenRemainQuota(t, tokenID))
	assert.Equal(t, preConsumed, task.Quota)
	assert.Equal(t, int64(0), countLogs(t))
}

func TestSettle_NonPerCallBilling_AppliesAdaptorAdjustment(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 32, 32, 32
	const initQuota, preConsumed = 10000, 5000
	const adaptorQuota = 3000
	const tokenRemain = 8000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-nonpercall-adj", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	// PerCallBilling defaults to false

	adaptor := &mockAdaptor{adjustReturn: adaptorQuota}
	taskResult := &relaycommon.TaskInfo{Status: model.TaskStatusSuccess}

	settleTaskBillingOnComplete(ctx, adaptor, task, taskResult)

	// Non-per-call: adaptor adjustment applies (refund 2000)
	assert.Equal(t, initQuota+(preConsumed-adaptorQuota), getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+(preConsumed-adaptorQuota), getTokenRemainQuota(t, tokenID))
	assert.Equal(t, adaptorQuota, task.Quota)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeRefund, log.Type)
}
