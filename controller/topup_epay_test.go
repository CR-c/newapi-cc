package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEpayNotifyReturnsFailWhenWalletCreditFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.QuotaLedger{}))

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainDBType, originalLogDBType := common.MainDatabaseType(), common.LogDatabaseType()
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	paymentSetting := operation_setting.GetPaymentSetting()
	originalCompliance := *paymentSetting
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainDBType, originalLogDBType)
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		*paymentSetting = originalCompliance
	})
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	operation_setting.PayAddress = "https://epay.test"
	operation_setting.EpayId = "merchant-test"
	operation_setting.EpayKey = "secret-test"
	operation_setting.PayMethods = []map[string]string{{"type": "alipay", "name": "Alipay"}}
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	user := &model.User{
		Id: 993, Username: "epay-wallet-corrupt", Status: common.UserStatusEnabled,
		Quota: 100, WalletVersion: model.CurrentWalletVersion,
	}
	require.NoError(t, db.Create(user).Error)
	topUp := &model.TopUp{
		UserId: user.Id, Amount: 1, Money: 1, TradeNo: "epay-wallet-credit-fails",
		PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay,
		Status: common.TopUpStatusPending, PrincipalQuota: 100, SnapshotVersion: model.CurrentTopUpSnapshotVersion,
	}
	require.NoError(t, db.Create(topUp).Error)

	params := map[string]string{
		"pid": operation_setting.EpayId, "type": "alipay",
		"out_trade_no": topUp.TradeNo, "trade_no": "upstream-trade",
		"name": "Top up", "money": "1.00", "trade_status": epay.StatusTradeSuccess,
	}
	epay.GenerateParams(params, operation_setting.EpayKey)
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/epay/notify?"+query.Encode(), nil)
	EpayNotify(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "fail", recorder.Body.String())
	require.NoError(t, db.First(topUp, topUp.Id).Error)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
}
