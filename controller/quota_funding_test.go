package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveAdminQuotaCreditBucketDefaultsToPromo(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   model.WalletBucket
	}{
		{name: "missing", source: "", want: model.WalletBucketPromo},
		{name: "promo", source: "promo", want: model.WalletBucketPromo},
		{name: "paid", source: "paid", want: model.WalletBucketPaid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bucket, err := resolveAdminQuotaCreditBucket(test.source)
			require.NoError(t, err)
			assert.Equal(t, test.want, bucket)
		})
	}

	_, err := resolveAdminQuotaCreditBucket("legacy_unknown")
	require.Error(t, err)
}

func TestValidateRedemptionQuotaBounds(t *testing.T) {
	require.Error(t, validateRedemptionQuota(0))
	require.Error(t, validateRedemptionQuota(-1))
	require.NoError(t, validateRedemptionQuota(1))
	require.NoError(t, validateRedemptionQuota(common.MaxQuota))
	require.Error(t, validateRedemptionQuota(common.MaxQuota+1))
}

func TestNormalizeNewRedemptionFundingSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "missing defaults to paid", source: "", want: model.RedemptionFundingSourcePaid},
		{name: "paid card", source: model.RedemptionFundingSourcePaid, want: model.RedemptionFundingSourcePaid},
		{name: "gift code", source: model.RedemptionFundingSourcePromo, want: model.RedemptionFundingSourcePromo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeNewRedemptionFundingSource(tt.source)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	for _, source := range []string{model.RedemptionFundingSourceLegacyUnknown, "cash", "PAID"} {
		_, err := normalizeNewRedemptionFundingSource(source)
		require.Error(t, err)
	}
}

func TestRedemptionMutationsRequireRoot(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	handlers := []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{name: "create", handler: AddRedemption},
		{name: "update", handler: UpdateRedemption},
		{name: "delete", handler: DeleteRedemption},
		{name: "delete invalid", handler: DeleteInvalidRedemption},
	}

	for _, tt := range handlers {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/redemption", nil)
			ctx.Set("role", common.RoleAdminUser)

			tt.handler(ctx)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success)
			assert.NotEmpty(t, response.Message)
		})
	}
}

func TestRootCanReclassifyExistingUserBalance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:controller_wallet_reclassify?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.QuotaLedger{}, &model.Log{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.RedisEnabled = originalRedisEnabled
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		require.NoError(t, sqlDB.Close())
	})

	user := model.User{
		Id: 99, Username: "wallet-reclassify-user", Password: "password", Role: common.RoleCommonUser,
		Quota: 100, PaidQuota: 100, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, db.Create(&user).Error)
	body, err := common.Marshal(ManageRequest{
		Id: user.Id, Action: "reclassify_wallet",
		ExpectedWallet: model.WalletAllocation{PaidQuota: 100},
		TargetWallet:   model.WalletAllocation{PromoQuota: 100},
		Reason:         "manual gift allocation",
	})
	require.NoError(t, err)
	adminRecorder := httptest.NewRecorder()
	adminCtx, _ := gin.CreateTestContext(adminRecorder)
	adminCtx.Request = httptest.NewRequest(http.MethodPost, "/api/user/manage", bytes.NewReader(body))
	adminCtx.Set("role", common.RoleAdminUser)
	adminCtx.Set("id", 2)

	ManageUser(adminCtx)

	var adminResponse struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(adminRecorder.Body.Bytes(), &adminResponse))
	assert.False(t, adminResponse.Success)
	var unchanged model.User
	require.NoError(t, db.First(&unchanged, user.Id).Error)
	assert.Equal(t, 100, unchanged.PaidQuota)
	assert.Zero(t, unchanged.PromoQuota)
	var ledgerCount int64
	require.NoError(t, db.Model(&model.QuotaLedger{}).Count(&ledgerCount).Error)
	assert.Zero(t, ledgerCount)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/manage", bytes.NewReader(body))
	ctx.Set("role", common.RoleRootUser)
	ctx.Set("id", 1)
	ctx.Set(common.RequestIdKey, "wallet-reclassify-request")

	ManageUser(ctx)

	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, recorder.Body.String())

	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, 100, stored.Quota)
	assert.Zero(t, stored.PaidQuota)
	assert.Equal(t, 100, stored.PromoQuota)
	assert.Zero(t, stored.LegacyUnknownQuota)
}
